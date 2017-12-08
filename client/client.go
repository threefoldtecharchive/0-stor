package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/zero-os/0-stor/client/components"
	"github.com/zero-os/0-stor/client/metastor/etcd"

	"github.com/zero-os/0-stor/client/components/chunker"
	"github.com/zero-os/0-stor/client/components/crypto"
	"github.com/zero-os/0-stor/client/components/distribution"
	"github.com/zero-os/0-stor/client/components/encrypt"
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
	policy   Policy
	iyoToken string

	storClients   map[string]datastor.Client
	muStorClients sync.Mutex

	metaCli metastor.Client
	iyoCl   itsyouonline.IYOClient
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
		iyoToken string
		err      error
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

	client := Client{
		policy:      policy,
		iyoToken:    iyoToken,
		iyoCl:       iyoCl,
		storClients: make(map[string]datastor.Client, len(policy.DataShards)),
	}

	// instantiate stor client for each shards.
	for _, shard := range policy.DataShards {
		// getStor keep the created stor in a map
		_, err := client.getStor(shard)
		if err != nil {
			return nil, err
		}
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

	for shard, cl := range c.storClients {
		closer, ok := cl.(io.Closer)
		if ok {
			if err := closer.Close(); err != nil {
				log.Errorf("Error closing stor client to %v", shard)
			}
		}
	}
	return nil
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

	var usedShards []string
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

		switch {
		case c.policy.ReplicationEnabled(len(block)):
			usedShards, err = c.replicateWrite(chunkKey, block, refList)
			if err != nil {
				return nil, err
			}
		case c.policy.DistributionEnabled():
			usedShards, _, err = c.distributeWrite(chunkKey, block, refList)
			if err != nil {
				return nil, err
			}
		default:
			shard, err := c.writeRandom(chunkKey, block, refList)
			if err != nil {
				return nil, err
			}
			usedShards = []string{shard}
		}

		chunk.Size = int64(len(block))
		chunk.Shards = usedShards
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
		obj   *datastor.Object
	)

	if c.policy.Encrypt {
		aesgm, err = encrypt.NewEncrypterDecrypter(encrypt.Config{PrivKey: c.policy.EncryptKey, Type: encrypt.TypeAESGCM})
		if err != nil {
			return
		}
	}

	for _, chunk := range md.Chunks {

		switch {
		case c.policy.ReplicationEnabled(int(chunk.Size)):
			obj, err = c.replicateRead(chunk.Key, chunk.Shards)
			if err != nil {
				return
			}
		case c.policy.DistributionEnabled():
			obj, err = c.distributeRead(chunk.Key, int(chunk.Size), chunk.Shards)
			if err != nil {
				return
			}
		default:
			if len(chunk.Shards) <= 0 {
				err = fmt.Errorf("metadata corrupted, can't have a chunk without shard")
				return
			}

			obj, err = c.read(chunk.Key, chunk.Shards[0])
			if err != nil {
				return
			}
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
				stor, err := c.getStor(job.shard)
				if err != nil {
					// FIXME: probably want to handle this
					log.Errorf("error deleting object:%v", err)
					continue
				}
				if err := stor.DeleteObject(job.key); err != nil {
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

func (c *Client) Check(key []byte) (datastor.ObjectStatus, error) {
	meta, err := c.metaCli.GetMetadata(key)
	if err != nil {
		log.Errorf("fail to get metadata for check: %v", err)
		return datastor.ObjectStatus(0), err
	}

	// create a map of chunk per shards, so we can ask all the check for a specific shards in one call
	idsPerShard := make(map[string][][]byte, len(c.policy.DataShards))
	for _, chunk := range meta.Chunks {
		for _, shard := range chunk.Shards {
			if _, ok := idsPerShard[shard]; !ok {
				idsPerShard[shard] = [][]byte{chunk.Key}
			} else {
				idsPerShard[shard] = append(idsPerShard[shard], chunk.Key)
			}
		}
	}

	var (
		cErr  = make(chan error, len(idsPerShard))
		cDone = make(chan struct{})
		// if one block is corrupted, we send a signal on that channel.
		// since we don't care to know what block is corrupted,
		// as soon as something is received on this channel, we can say the file
		// is corrupted
		cStatus     = make(chan datastor.ObjectStatus, len(idsPerShard))
		wg          sync.WaitGroup
		ctx, cancel = context.WithCancel(context.Background())
	)

	// this is called as soon as we know if one block is corrupted
	defer cancel()

	wg.Add(len(idsPerShard))
	for shard, ids := range idsPerShard {
		go func(ctx context.Context, shard string, ids [][]byte, cStatus chan<- datastor.ObjectStatus, cErr chan<- error) {
			defer wg.Done()

			store, err := c.getStor(shard)
			if err != nil {
				log.Errorf("error getting client store for shard %s: %v", shard, err)
				cErr <- err
				return
			}

			select {
			case <-ctx.Done():
				// in case we already know something is corrupted, we don't need
				// to check other blokcs
				return

			default:
				for _, id := range ids {
					status, err := store.GetObjectStatus([]byte(id))
					if err != nil {
						log.Errorf("error getting object status on shard %s: %v", shard, err)
						cErr <- err
						return
					}

					if status != datastor.ObjectStatusOK {
						// signal we found a corrupted or missing block
						cStatus <- status
					}
				}
			}
		}(ctx, shard, ids, cStatus, cErr)
	}

	go func() {
		wg.Wait()
		cDone <- struct{}{}
	}()

	select {
	case err := <-cErr:
		return datastor.ObjectStatusCorrupted, err
	case <-cDone:
		// all is good
		return datastor.ObjectStatusOK, nil
	case state := <-cStatus:
		// something is wrong
		return state, nil
	}
}

func (c *Client) replicateWrite(key, value []byte, referenceList []string) ([]string, error) {
	type Job struct {
		client datastor.Client
		shard  string
	}

	var (
		usedShards = []string{}
		okShards   = make([]string, 0, c.policy.ReplicationNr)
		wg         sync.WaitGroup
		mu         sync.Mutex
		shardErr   = &components.ShardError{}
		cJob       = make(chan *Job)
	)

	wg.Add(1)
	go func() {
		defer wg.Done()

		for job := range cJob {
			wg.Add(1)

			go func(job *Job) {
				defer wg.Done()

				err := job.client.SetObject(datastor.Object{
					Key:           key,
					Data:          value,
					ReferenceList: referenceList,
				})
				if err != nil {
					log.Errorf("replication write: error writing to store %s: %v", job.shard, err)
					shardErr.Add([]string{job.shard}, components.ShardType0Stor, err, 0)
					return
				}
				mu.Lock()
				okShards = append(okShards, job.shard)
				mu.Unlock()
			}(job)
		}
	}()

	for i := 0; i < c.policy.ReplicationNr; i++ {
		cl, shard, err := c.getRandomStor(usedShards)
		if err != nil {
			return nil, err
		}

		cJob <- &Job{
			client: cl,
			shard:  shard,
		}
		usedShards = append(usedShards, shard)
	}

	close(cJob)
	wg.Wait()

	if len(okShards) < c.policy.ReplicationNr {
		// missing some replication, try to send sequentially to remaining store
		for len(okShards) < c.policy.ReplicationNr {
			cl, shard, err := c.getRandomStor(usedShards)
			if err != nil {
				// mean we don't have anymore store available
				if err == errNoDataShardAvailable {
					return usedShards, fmt.Errorf("coudn't replicate data to enough 0stor server, only %d succeeded, %d required", len(okShards), c.policy.ReplicationNr)
				}
				return nil, err
			}

			err = cl.SetObject(datastor.Object{
				Key:           key,
				Data:          value,
				ReferenceList: referenceList,
			})
			if err != nil {
				log.Errorf("replication write: error writing to store %s: %v", shard, err)
				shardErr.Add([]string{shard}, components.ShardType0Stor, err, 0)
				usedShards = append(usedShards, shard)
				continue
			}

			okShards = append(okShards, shard)
			usedShards = append(usedShards, shard)
		}
	}

	// if still not enough, return error, we can't do anything more
	if len(okShards) < c.policy.ReplicationNr {
		return usedShards, fmt.Errorf("coudn't replicate data to enough 0stor server, only %d succeeded, %d required", len(okShards), c.policy.ReplicationNr)
	}

	return usedShards, nil
}

func (c *Client) replicateRead(key []byte, shards []string) (*datastor.Object, error) {
	wg := sync.WaitGroup{}
	cVal := make(chan datastor.Object)
	cAllDone := make(chan struct{})
	cQuit := make(chan struct{})

	// start a gorountine to all possible shard
	// the first stor to respond send the value received to cVal
	// As soon as something is received into cVal, I close cQuit, so all other rountine should exit
	for _, shard := range shards {
		wg.Add(1)
		go func(shard string) {
			defer wg.Done()

			cl, err := c.getStor(shard)
			if err != nil {
				log.Warningf("replication read, error getting client for %s: %v", shard, err)
				return
			}
			obj, err := cl.GetObject(key)
			if err != nil {
				log.Warningf("replication read, error reading from %s: %v", shard, err)
				return
			}

			select {
			case <-cQuit:
			case cVal <- *obj:
			}
			return
		}(shard)
	}

	// wait for all gorountine to exit
	go func() {
		wg.Wait()
		close(cAllDone)
	}()

	select {
	case <-cAllDone:
		// if we recevie this before the value, it means we couln't get the data back
		// from any store
		close(cQuit)
		return nil, fmt.Errorf("can't find a valid replication of the object")
	case val := <-cVal:
		close(cQuit)
		return &val, nil

	}
}

func (c *Client) distributeWrite(key, value []byte, referenceList []string) ([]string, uint64, error) {

	encoder, err := distribution.NewEncoder(c.policy.DistributionNr, c.policy.DistributionRedundancy)
	if err != nil {
		return nil, 0, err
	}

	parts, err := encoder.Encode(value)
	if err != nil {
		return nil, 0, err
	}

	type Job struct {
		client datastor.Client
		part   []byte
		shard  string
	}

	var (
		cJob       = make(chan *Job)
		usedShards = make([]string, 0, len(parts))
		size       = uint64(0)
		shardErr   = &components.ShardError{}
		wg         sync.WaitGroup
	)

	wg.Add(1)
	go func(cJob <-chan *Job) {
		defer wg.Done()
		// gorountine receive work from channel
		// each work object receive start a new goroutine that write the part to the store
		for job := range cJob {
			wg.Add(1)

			go func(job *Job) {
				defer wg.Done()
				err := job.client.SetObject(datastor.Object{
					Key:           key,
					Data:          job.part,
					ReferenceList: referenceList,
				})
				if err != nil {
					log.Errorf("error writing to stor: %v", err)
					shardErr.Add([]string{job.shard}, components.ShardType0Stor, err, 0)
				}
			}(job)
		}
	}(cJob)

	for i, part := range parts {
		cl, shard, err := c.getRandomStor(usedShards)
		if err != nil {
			if err == errNoDataShardAvailable {
				return nil, 0, shardErr
			}
			shardErr.Add([]string{shard}, components.ShardType0Stor, err, 0)
			continue
		}

		cJob <- &Job{
			client: cl,
			shard:  shard,
			part:   part,
		}

		usedShards = append(usedShards, shard)
		if i < c.policy.DistributionNr {
			size += uint64(len(part))
		}
	}
	// close job channel, this will allow job consuming routine to exit
	close(cJob)

	wg.Wait()

	if !shardErr.Nil() {
		log.Errorf("error distributin write: %v", shardErr)
		return usedShards, size, shardErr
	}

	return usedShards, size, nil
}

func (c *Client) distributeRead(key []byte, originalSize int, shards []string) (*datastor.Object, error) {

	dec, err := distribution.NewDecoder(c.policy.DistributionNr, c.policy.DistributionRedundancy)
	if err != nil {
		return nil, err
	}

	var (
		wg           = sync.WaitGroup{}
		shardErr     = &components.ShardError{}
		parts        = make([][]byte, len(shards))
		refListSlice = make([][]string, len(shards))
	)

	wg.Add(len(shards))
	for i, shard := range shards {
		go func(i int, shard string) {
			defer wg.Done()

			cl, err := c.getStor(shard)
			if err != nil {
				shardErr.Add([]string{shard}, components.ShardType0Stor, err, 0)
				return
			}

			obj, err := cl.GetObject(key)
			if err != nil {
				shardErr.Add([]string{shard}, components.ShardType0Stor, err, 0)
				return
			}

			parts[i] = obj.Data
			refListSlice[i] = obj.ReferenceList
		}(i, shard)
	}

	wg.Wait()

	if !shardErr.Nil() && shardErr.Num() > c.policy.DistributionRedundancy {
		return nil, shardErr
	}

	decoded, err := dec.Decode(parts, originalSize)
	if err != nil {
		return nil, err
	}

	// get non empty refList
	// empty refList could be caused by shard error
	var refList []string
	for _, rl := range refListSlice {
		if len(rl) > 0 {
			refList = rl
			break
		}
	}

	return &datastor.Object{
		Key:           key,
		Data:          decoded,
		ReferenceList: refList,
	}, nil
}

func (c *Client) writeRandom(key, value []byte, referenceList []string) (string, error) {
	triedShards := []string{}

	for {
		cl, shard, err := c.getRandomStor(triedShards)
		if err != nil {
			return "", err
		}

		triedShards = append(triedShards, shard)

		err = cl.SetObject(datastor.Object{
			Key:           key,
			Data:          value,
			ReferenceList: referenceList,
		})
		if err == nil {
			return shard, nil
		}
		log.Error(err)
	}
}

func (c *Client) read(key []byte, shard string) (*datastor.Object, error) {
	cl, err := c.getStor(shard)
	if err != nil {
		return nil, err
	}
	return cl.GetObject(key)
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

func (c *Client) getRandomStor(except []string) (datastor.Client, string, error) {
	isIn := func(target string, list []string) bool {
		for _, x := range list {
			if target == x {
				return true
			}
		}
		return false
	}

	possibleShards := []string{}
	for _, shard := range c.policy.DataShards {
		if !isIn(shard, except) {
			possibleShards = append(possibleShards, shard)
		}
	}

	var shard string
	if len(possibleShards) <= 0 {
		return nil, "", errNoDataShardAvailable
	} else if len(possibleShards) == 1 {
		shard = possibleShards[0]
	} else {
		shard = possibleShards[rand.Intn(len(possibleShards)-1)]
	}

	// TODO: find a way to invalidate some client if an error occurs with it

	cl, err := c.getStor(shard)
	return cl, shard, err
}

func (c *Client) getStor(shard string) (datastor.Client, error) {
	c.muStorClients.Lock()
	defer c.muStorClients.Unlock()

	// first check if we don't already have a client to this shard loaded
	cl, ok := c.storClients[shard]
	if ok {
		return cl, nil
	}

	// if not create the client and put in cache
	namespace := fmt.Sprintf("%s_0stor_%s", c.policy.Organization, c.policy.Namespace)
	cl, err := storgrpc.NewClient(shard, namespace, c.iyoToken)
	if err != nil {
		return nil, err
	}
	c.storClients[shard] = cl

	return cl, nil
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
