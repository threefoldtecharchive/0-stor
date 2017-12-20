package client

import (
	"bytes"
	"errors"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/zero-os/0-stor/client/pipeline"

	"github.com/zero-os/0-stor/client/metastor/etcd"

	"github.com/zero-os/0-stor/client/datastor"
	storgrpc "github.com/zero-os/0-stor/client/datastor/grpc"
	"github.com/zero-os/0-stor/client/itsyouonline"
	"github.com/zero-os/0-stor/client/metastor"
	"github.com/zero-os/0-stor/client/pipeline/storage"

	log "github.com/Sirupsen/logrus"
)

var (
	errWriteFChunkerOnly    = errors.New("WriteF only support chunker as first pipe")
	errReadFChunkerOnly     = errors.New("ReadF only support chunker as first pipe")
	errNoDataShardAvailable = errors.New("no more data shard available")
)

// Client defines 0-stor client
type Client struct {
	datastorCluster datastor.Cluster
	dataPipeline    pipeline.Pipeline

	metastorClient metastor.Client
}

// NewClientFromConfig creates new 0-stor client using the given config.
func NewClientFromConfig(cfg Config, jobCount int) (*Client, error) {
	var (
		err             error
		datastorCluster datastor.Cluster
	)
	// create datastor cluster
	if cfg.IYO != (itsyouonline.Config{}) {
		client, err := itsyouonline.NewClient(cfg.IYO)
		if err == nil {
			tokenGetter := jwtTokenGetterFromIYOClient(
				cfg.IYO.Organization, client)
			datastorCluster, err = storgrpc.NewCluster(cfg.DataStor.Shards, cfg.Namespace, tokenGetter)
		}
	} else {
		datastorCluster, err = storgrpc.NewCluster(cfg.DataStor.Shards, cfg.Namespace, nil)
	}
	if err != nil {
		return nil, err
	}

	// create data pipeline, using our datastor cluster
	dataPipeline, err := pipeline.NewPipeline(cfg.Pipeline, datastorCluster, jobCount)
	if err != nil {
		return nil, err
	}

	// if no metadata shards are given, return an error,
	// as we require a metastor client
	// TODO: allow a more flexible kind of metastor client configuration,
	// so we can also allow other types of metastor clients,
	// as we do really need one.
	if len(cfg.MetaStor.Shards) == 0 {
		return nil, errors.New("no metadata storage given")
	}

	// create metastor client first,
	// and than create our master 0-stor client with all features.
	metastorClient, err := etcd.NewClient(cfg.MetaStor.Shards)
	if err != nil {
		return nil, err
	}
	return NewClient(datastorCluster, metastorClient, dataPipeline)
}

// NewClient creates a 0-stor client,
// with the data (zstordb) cluster already created,
// used to read/write object data, as well as the metastor client,
// which is used to read/write the metadata of the objects.
//
// The given data pipeline is optional and a default one will be created should it not be defined.
func NewClient(dataCluster datastor.Cluster, metaClient metastor.Client, dataPipeline pipeline.Pipeline) (*Client, error) {
	if dataCluster == nil {
		panic("0-stor Client: no datastor cluster given")
	}
	if metaClient == nil {
		panic("0-stor Client: no metastor client given")
	}

	// create default pipeline
	if dataPipeline == nil {
		pipeline, err := pipeline.NewPipeline(pipeline.Config{}, dataCluster, -1)
		if err != nil {
			return nil, err
		}
		dataPipeline = pipeline
	}

	return &Client{
		datastorCluster: dataCluster,
		dataPipeline:    dataPipeline,
		metastorClient:  metaClient,
	}, nil
}

// Close the client
func (c *Client) Close() error {
	c.metastorClient.Close()
	if closer, ok := c.datastorCluster.(interface {
		Close() error
	}); ok {
		return closer.Close()
	}
	return nil
}

// Write write the value to the the 0-stors configured by the client config
func (c *Client) Write(key, value []byte) (*metastor.Data, error) {
	return c.WriteWithMeta(key, value, nil, nil, nil)
}

func (c *Client) WriteF(key []byte, r io.Reader) (*metastor.Data, error) {
	return c.writeFWithMeta(key, r, nil, nil, nil)
}

// WriteWithMeta writes the key-value to the configured pipes.
// Metadata linked list will be build if prevKey is not nil
// prevMeta is optional previous metadata, to be used in case of user already has the prev metastor.
// So the client won't need to fetch it back from the metadata server.
// prevKey still need to be set it prevMeta is set
// initialMeta is optional metadata, if user want to set his own initial metadata for example set own epoch
func (c *Client) WriteWithMeta(key, val []byte, prevKey []byte, prevMeta, md *metastor.Data) (*metastor.Data, error) {
	r := bytes.NewReader(val)
	return c.writeFWithMeta(key, r, prevKey, prevMeta, md)
}

func (c *Client) WriteFWithMeta(key []byte, r io.Reader, prevKey []byte, prevMeta, md *metastor.Data) (*metastor.Data, error) {
	return c.writeFWithMeta(key, r, prevKey, prevMeta, md)
}

func (c *Client) writeFWithMeta(key []byte, r io.Reader, prevKey []byte, prevMeta, md *metastor.Data) (*metastor.Data, error) {
	chunks, err := c.dataPipeline.Write(r)
	if err != nil {
		return nil, err
	}

	// create new metadata if not given
	if md == nil {
		md = &metastor.Data{
			Key:   key,
			Epoch: time.Now().UnixNano(),
		}
	}

	// update chunks in metadata
	// TODO: fix this (pointer) difference in API
	md.Chunks = make([]*metastor.Chunk, len(chunks))
	md.Size = 0
	for index := range chunks {
		chunk := &chunks[index]
		md.Chunks[index] = chunk
		md.Size += chunk.Size
	}

	err = c.linkMeta(md, prevMeta, key, prevKey)
	if err != nil {
		return md, err
	}

	return md, nil
}

// Read reads value with given key from the 0-stors configured by the client cnfig
// it will first try to get the metadata associated with key from the Metadata servers.
func (c *Client) Read(key []byte) ([]byte, error) {
	md, err := c.metastorClient.GetMetadata(key)
	if err != nil {
		return nil, err
	}

	w := &bytes.Buffer{}
	err = c.readFWithMeta(md, w)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

// ReadF similar as Read but write the data to w instead of returning a slice of bytes
func (c *Client) ReadF(key []byte, w io.Writer) error {
	md, err := c.metastorClient.GetMetadata(key)
	if err != nil {
		return err
	}
	return c.readFWithMeta(md, w)

}

// ReadWithMeta reads the value described by md
func (c *Client) ReadWithMeta(md *metastor.Data) ([]byte, error) {
	w := &bytes.Buffer{}
	err := c.readFWithMeta(md, w)
	if err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

func (c *Client) readFWithMeta(md *metastor.Data, w io.Writer) error {
	// get chunks from metadata
	// TODO: fix this (pointer) difference in API
	chunks := make([]metastor.Chunk, len(md.Chunks))
	for index := range md.Chunks {
		chunks[index] = *md.Chunks[index]
	}
	return c.dataPipeline.Read(chunks, w)
}

// Delete deletes object from the 0-stor server pointed by the key
// It also deletes the metadatastor.
func (c *Client) Delete(key []byte) error {
	meta, err := c.metastorClient.GetMetadata(key)
	if err != nil {
		log.Errorf("fail to read metadata: %v", err)
		return err
	}
	return c.DeleteWithMeta(meta)
}

// DeleteWithMeta deletes object from the 0-stor server pointed by the
// given metadata
// It also deletes the metadatastor.
func (c *Client) DeleteWithMeta(meta *metastor.Data) error {
	type job struct {
		key   []byte
		shard string
	}

	var (
		wg   = sync.WaitGroup{}
		cJob = make(chan job, runtime.NumCPU())
	)

	// create some worker that will delete all shards
	wg.Add(runtime.NumCPU())
	for i := 0; i < runtime.NumCPU(); i++ {
		go func(cJob chan job) {
			defer wg.Done()

			for job := range cJob {
				shard, err := c.datastorCluster.GetShard(job.shard)
				if err != nil {
					// FIXME: probably want to handle this
					log.Errorf("error deleting object:%v", err)
					continue
				}
				if err := shard.DeleteObject(job.key); err != nil {
					// FIXME: probably want to handle this
					log.Errorf("error deleting object:%v", err)
					continue
				}
			}

		}(cJob)
	}

	// send job to the workers
	for _, chunk := range meta.Chunks {
		for _, shard := range chunk.Shards {
			cJob <- job{
				key:   chunk.Key,
				shard: shard,
			}
		}
	}
	close(cJob)

	// wait for all shards to be delete
	wg.Wait()

	// delete metadata
	if err := c.metastorClient.DeleteMetadata(meta.Key); err != nil {
		log.Errorf("error deleting metadata :%v", err)
		return err
	}

	return nil
}

func (c *Client) Check(key []byte) (storage.ObjectCheckStatus, error) {
	meta, err := c.metastorClient.GetMetadata(key)
	if err != nil {
		log.Errorf("fail to get metadata for check: %v", err)
		return storage.ObjectCheckStatus(0), err
	}

	for _, chunk := range meta.Chunks {
		status, err := c.dataPipeline.GetObjectStorage().Check(storage.ObjectConfig{
			Key:      chunk.Key,
			Shards:   chunk.Shards,
			DataSize: int(chunk.Size),
		}, false)
		if err != nil || status == storage.ObjectCheckStatusInvalid {
			return status, err
		}
	}
	return storage.ObjectCheckStatusValid, nil
}

func (c *Client) linkMeta(curMd, prevMd *metastor.Data, curKey, prevKey []byte) error {
	if len(prevKey) == 0 {
		return c.metastorClient.SetMetadata(*curMd)
	}

	// point next key of previous meta to new meta
	prevMd.Next = curKey

	// point prev key of new meta to previous one
	curMd.Previous = prevKey

	// update prev meta
	if err := c.metastorClient.SetMetadata(*prevMd); err != nil {
		return err
	}

	// update new meta
	return c.metastorClient.SetMetadata(*curMd)
}
