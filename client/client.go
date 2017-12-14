package client

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"runtime"
	"sync"
	"time"

	"github.com/zero-os/0-stor/client/metastor/etcd"

	"github.com/zero-os/0-stor/client/components/chunker"
	"github.com/zero-os/0-stor/client/components/crypto"
	"github.com/zero-os/0-stor/client/components/encrypt"
	"github.com/zero-os/0-stor/client/components/storage"
	"github.com/zero-os/0-stor/client/datastor"
	storgrpc "github.com/zero-os/0-stor/client/datastor/grpc"
	"github.com/zero-os/0-stor/client/itsyouonline"
	"github.com/zero-os/0-stor/client/metastor"

	log "github.com/Sirupsen/logrus"
	"github.com/golang/snappy"
)

var (
	errWriteFChunkerOnly    = errors.New("WriteF only support chunker as first pipe")
	errReadFChunkerOnly     = errors.New("ReadF only support chunker as first pipe")
	errNoDataShardAvailable = errors.New("no more data shard available")
)

var _ (itsyouonline.IYOClient) = (*Client)(nil) // build time check that we implement itsyouonline.IYOClient interface

// Client defines 0-stor client
type Client struct {
	policy Policy

	metaCli metastor.Client
	iyoCl   itsyouonline.IYOClient

	storage storage.ObjectStorage
	// does not have to be closed by client,
	// as storage already closes it
	cluster clusterCloser
}

type clusterCloser interface {
	datastor.Cluster

	Close() error
}

// New creates new client from the given config
func New(policy Policy) (*Client, error) {
	var iyoCl itsyouonline.IYOClient
	if policy.Organization != "" && policy.IYOAppID != "" && policy.IYOSecret != "" {
		iyoCl = itsyouonline.NewClient(policy.Organization, policy.IYOAppID, policy.IYOSecret)
	}

	return newClient(policy, iyoCl)
}

func newClient(policy Policy, iyoCl itsyouonline.IYOClient) (*Client, error) {
	var (
		err      error
		iyoToken string
	)

	if iyoCl != nil {
		iyoToken, err = iyoCl.CreateJWT(policy.Namespace, itsyouonline.Permission{
			Write:  true,
			Read:   true,
			Delete: true,
		})
		if err != nil {
			log.Error(err.Error())
			return nil, err
		}
	}

	cluster, err := storgrpc.NewCluster(policy.DataShards, policy.Namespace, iyoToken)
	if err != nil {
		return nil, err
	}

	var objectStorage storage.ObjectStorage
	switch {
	case policy.DistributionEnabled():
		objectStorage, err = storage.NewDistributedObjectStorage(
			cluster, policy.DistributionNr, policy.DistributionRedundancy, 0)
		if err != nil {
			return nil, err
		}

	case policy.ReplicationNr > 1:
		objectStorage, err = storage.NewReplicatedObjectStorage(
			cluster, policy.ReplicationNr, 0)
		if err != nil {
			return nil, err
		}

	default:
		objectStorage, err = storage.NewRandomObjectStorage(cluster)
		if err != nil {
			return nil, err
		}
	}

	client := Client{
		policy:  policy,
		iyoCl:   iyoCl,
		storage: objectStorage,
		cluster: cluster,
	}

	// instantiate meta client
	if len(policy.MetaShards) > 0 {
		// meta client
		metaCli, err := etcd.NewClient(policy.MetaShards)
		if err != nil {
			return nil, err
		}
		client.metaCli = metaCli
	}

	return &client, nil
}

// Close the client
func (c *Client) Close() error {
	if c.metaCli != nil {
		c.metaCli.Close()
	}
	return c.cluster.Close()
}

// Write write the value to the the 0-stors configured by the client policy
func (c *Client) Write(key, value []byte, refList []string) (*metastor.Data, error) {
	return c.WriteWithMeta(key, value, nil, nil, nil, refList)
}

func (c *Client) WriteF(key []byte, r io.Reader, refList []string) (*metastor.Data, error) {
	return c.writeFWithMeta(key, r, nil, nil, nil, refList)
}

// WriteWithMeta writes the key-value to the configured pipes.
// Metadata linked list will be build if prevKey is not nil
// prevMeta is optional previous metadata, to be used in case of user already has the prev metastor.
// So the client won't need to fetch it back from the metadata server.
// prevKey still need to be set it prevMeta is set
// initialMeta is optional metadata, if user want to set his own initial metadata for example set own epoch
func (c *Client) WriteWithMeta(key, val []byte, prevKey []byte, prevMeta, md *metastor.Data, refList []string) (*metastor.Data, error) {
	r := bytes.NewReader(val)
	return c.writeFWithMeta(key, r, prevKey, prevMeta, md, refList)
}

func (c *Client) WriteFWithMeta(key []byte, r io.Reader, prevKey []byte, prevMeta, md *metastor.Data, refList []string) (*metastor.Data, error) {
	return c.writeFWithMeta(key, r, prevKey, prevMeta, md, refList)
}

func (c *Client) writeFWithMeta(key []byte, r io.Reader, prevKey []byte, prevMeta, md *metastor.Data, refList []string) (*metastor.Data, error) {
	var (
		blockSize int
		err       error
		aesgm     encrypt.EncrypterDecrypter
	)

	blakeH, err := crypto.NewDefaultHasher256([]byte(c.policy.EncryptKey))
	if err != nil {
		return nil, err
	}

	if c.policy.Encrypt {
		aesgm, err = encrypt.NewEncrypterDecrypter(encrypt.Config{PrivKey: c.policy.EncryptKey, Type: encrypt.TypeAESGCM})
		if err != nil {
			return nil, err
		}
	}

	if len(prevKey) > 0 && prevMeta == nil {
		// get the prev meta now than later
		// to avoid making processing and then
		// just found that prev meta is invalid
		prevMeta, err = c.metaCli.GetMetadata(prevKey)
		if err != nil {
			return nil, err
		}
	}

	// create new metadata if not given
	if md == nil {
		md = &metastor.Data{
			Key:   key,
			Epoch: time.Now().UnixNano(),
		}
	}

	// define the block size to use
	// if policy block size is set to 0:
	//		 we read all content of r, get the size of the data
	// 		 and configuer the chuner with this size, so there is going to be only one chunk
	// else use the block size from the policy
	if c.policy.BlockSize <= 0 {
		b, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		blockSize = len(b)
		r = bytes.NewReader(b)
	} else {
		blockSize = c.policy.BlockSize
	}

	rd := chunker.NewReader(r, chunker.Config{ChunkSize: blockSize})

	var storCfg storage.ObjectConfig
	for rd.Next() {
		block := rd.Value()

		hashed := blakeH.HashBytes(block)

		chunkKey := hashed[:]
		chunk := &metastor.Chunk{Key: chunkKey}

		if c.policy.Encrypt {
			block, err = aesgm.Encrypt(block)
			chunk.Size = int64(len(block))
		}

		if c.policy.Compress {
			block = snappy.Encode(nil, block)
			chunk.Size = int64(len(block))
		}

		storCfg, err = c.storage.Write(datastor.Object{
			Key:           chunkKey,
			Data:          block,
			ReferenceList: refList,
		})
		if err != nil {
			return nil, err
		}

		chunk.Key = storCfg.Key
		chunk.Size = int64(storCfg.DataSize)
		chunk.Shards = storCfg.Shards
		md.Chunks = append(md.Chunks, chunk)
	}

	err = c.linkMeta(md, prevMeta, key, prevKey)
	if err != nil {
		return md, err
	}

	return md, nil
}

// Read reads value with given key from the 0-stors configured by the client policy
// it will first try to get the metadata associated with key from the Metadata servers.
// It returns the value and it's reference list
func (c *Client) Read(key []byte) ([]byte, []string, error) {
	md, err := c.metaCli.GetMetadata(key)
	if err != nil {
		return nil, nil, err
	}

	w := &bytes.Buffer{}
	refList, err := c.readFWithMeta(md, w)
	if err != nil {
		return nil, nil, err
	}

	return w.Bytes(), refList, nil
}

// ReadF similar as Read but write the data to w instead of returning a slice of bytes
func (c *Client) ReadF(key []byte, w io.Writer) ([]string, error) {
	md, err := c.metaCli.GetMetadata(key)
	if err != nil {
		return nil, err
	}
	return c.readFWithMeta(md, w)

}

// ReadWithMeta reads the value described by md
func (c *Client) ReadWithMeta(md *metastor.Data) ([]byte, []string, error) {
	w := &bytes.Buffer{}
	refList, err := c.readFWithMeta(md, w)
	if err != nil {
		return nil, nil, err
	}

	return w.Bytes(), refList, nil
}

func (c *Client) readFWithMeta(md *metastor.Data, w io.Writer) (refList []string, err error) {
	var (
		aesgm encrypt.EncrypterDecrypter
		block []byte
		obj   datastor.Object
	)

	if c.policy.Encrypt {
		aesgm, err = encrypt.NewEncrypterDecrypter(encrypt.Config{PrivKey: c.policy.EncryptKey, Type: encrypt.TypeAESGCM})
		if err != nil {
			return
		}
	}

	for _, chunk := range md.Chunks {
		obj, err = c.storage.Read(storage.ObjectConfig{
			Key:      chunk.Key,
			Shards:   chunk.Shards,
			DataSize: int(chunk.Size),
		})
		if err != nil {
			return
		}

		block = obj.Data
		refList = obj.ReferenceList

		if c.policy.Compress {
			block, err = snappy.Decode(nil, block)
			if err != nil {
				return
			}
		}

		if c.policy.Encrypt {
			block, err = aesgm.Decrypt(block)
			if err != nil {
				return
			}
		}

		_, err = w.Write(block)
		if err != nil {
			return
		}
	}

	return
}

// Delete deletes object from the 0-stor server pointed by the key
// It also deletes the metadatastor.
func (c *Client) Delete(key []byte) error {
	meta, err := c.metaCli.GetMetadata(key)
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
				shard, err := c.cluster.GetShard(job.shard)
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
	if err := c.metaCli.DeleteMetadata(meta.Key); err != nil {
		log.Errorf("error deleting metadata :%v", err)
		return err
	}

	return nil
}

func (c *Client) Check(key []byte) (storage.ObjectCheckStatus, error) {
	meta, err := c.metaCli.GetMetadata(key)
	if err != nil {
		log.Errorf("fail to get metadata for check: %v", err)
		return storage.ObjectCheckStatus(0), err
	}

	for _, chunk := range meta.Chunks {
		status, err := c.storage.Check(storage.ObjectConfig{
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
		return c.metaCli.SetMetadata(*curMd)
	}

	// point next key of previous meta to new meta
	prevMd.Next = curKey

	// point prev key of new meta to previous one
	curMd.Previous = prevKey

	// update prev meta
	if err := c.metaCli.SetMetadata(*prevMd); err != nil {
		return err
	}

	// update new meta
	return c.metaCli.SetMetadata(*curMd)
}

func (c *Client) CreateJWT(namespace string, perm itsyouonline.Permission) (string, error) {
	return c.iyoCl.CreateJWT(namespace, perm)
}
func (c *Client) CreateNamespace(namespace string) error {
	return c.iyoCl.CreateNamespace(namespace)
}
func (c *Client) DeleteNamespace(namespace string) error {
	return c.iyoCl.DeleteNamespace(namespace)
}
func (c *Client) GivePermission(namespace, userID string, perm itsyouonline.Permission) error {
	return c.iyoCl.GivePermission(namespace, userID, perm)
}
func (c *Client) RemovePermission(namespace, userID string, perm itsyouonline.Permission) error {
	return c.iyoCl.RemovePermission(namespace, userID, perm)
}
func (c *Client) GetPermission(namespace, userID string) (itsyouonline.Permission, error) {
	return c.iyoCl.GetPermission(namespace, userID)
}
