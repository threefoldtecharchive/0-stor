package distribution

import (
	"fmt"

	"github.com/zero-os/0-stor/client/itsyouonline"
	"github.com/zero-os/0-stor/client/lib/block"
	"github.com/zero-os/0-stor/client/lib/hash"
	"github.com/zero-os/0-stor/client/meta"
	"github.com/zero-os/0-stor/client/stor"
)

// StorDistributor defines distributor that use 0-stor rest/grpc clients
// as Writers.
// It create one stor.Client for each shard
type StorDistributor struct {
	enc         *Encoder
	hasher      *hash.Hasher
	storClients []stor.Client
	shards      []string
	w           block.Writer
}

// NewStorDistributor creates new StorDistributor
func NewStorDistributor(w block.Writer, conf Config, org, namespace string) (*StorDistributor, error) {
	if len(conf.Shards) < conf.NumPieces() {
		return nil, fmt.Errorf("invalid number of shards=%v, expected=%v", len(conf.Shards), conf.NumPieces())
	}

	// stor clients
	storClients, err := createStorClients(conf, conf.Shards, org, namespace)
	if err != nil {
		return nil, err
	}

	// encoder
	enc, err := NewEncoder(conf.Data, conf.Parity)
	if err != nil {
		return nil, err
	}

	// hasher
	hasher, err := hash.NewHasher(hash.Config{Type: hash.TypeBlake2})
	if err != nil {
		return nil, err
	}

	return &StorDistributor{
		storClients: storClients,
		enc:         enc,
		hasher:      hasher,
		shards:      conf.Shards,
		w:           w,
	}, nil
}

// Write implements io.Writer
func (sd StorDistributor) Write(data []byte) (int, error) {
	key := sd.hasher.Hash(data)
	encoded, err := sd.enc.Encode(data)
	if err != nil {
		return 0, err
	}

	for i, piece := range encoded {
		if _, err = sd.storClients[i].ObjectCreate(key, piece, nil); err != nil {
			return 0, err
		}
	}
	return len(data), nil
}

// WriteBlock implements block.Writer interface.
func (sd StorDistributor) WriteBlock(key, value []byte) (int, error) {
	hashedKey := sd.hasher.Hash(value)
	encoded, err := sd.enc.Encode(value)
	if err != nil {
		return 0, err
	}

	for i, piece := range encoded {
		_, err = sd.storClients[i].ObjectCreate(hashedKey, piece, nil)
		if err != nil {
			return 0, err
		}
	}

	md, err := meta.New(hashedKey, uint64(len(value)), sd.shards)
	if err != nil {
		return 0, err
	}

	mdBytes, err := md.Bytes()
	if err != nil {
		return 0, err
	}

	return sd.w.WriteBlock(key, mdBytes)
}

// StorRestorer defines distributor that get the data
// from 0-stor
type StorRestorer struct {
	dec         *Decoder
	storClients []stor.Client
}

// NewStorRestorer creates new StorRestorer
func NewStorRestorer(conf Config, org, namespace string) (*StorRestorer, error) {
	// stor clients
	storClients, err := createStorClients(conf, conf.Shards, org, namespace)
	if err != nil {
		return nil, err
	}

	dec, err := NewDecoder(conf.Data, conf.Parity)
	if err != nil {
		return nil, err
	}

	return &StorRestorer{
		dec:         dec,
		storClients: storClients,
	}, nil
}

// ReadBlock implements block.Reader
// The input is raw metadata
func (sr StorRestorer) ReadBlock(rawMeta []byte) ([]byte, error) {
	// decode the meta
	meta, err := meta.Decode(rawMeta)
	if err != nil {
		return nil, err
	}

	chunks := make([][]byte, sr.dec.k+sr.dec.m)

	// read all chunks from stor.Clients
	for i, sc := range sr.storClients {
		key, err := meta.Key()
		if err != nil {
			return nil, err
		}

		obj, err := sc.ObjectGet(key)
		if err != nil {
			return nil, err
		} else {
			chunks[i] = obj.Value
		}
	}

	// decode
	decoded, err := sr.dec.Decode(chunks, int(meta.Size()))
	return decoded, err
}

func createStorClients(conf Config, shards []string, org, namespace string) ([]stor.Client, error) {
	var scs []stor.Client
	var token string
	var err error

	// create IYO JWT token
	if conf.withIYoCredentials() {
		token, err = createJWTToken(conf, org, namespace)
		if err != nil {
			return nil, err
		}
	}

	// create stor clients
	storConf := stor.Config{
		Protocol: conf.Protocol,
	}
	for _, shard := range shards {
		storConf.Shard = shard
		storClient, err := stor.NewClientWithToken(&storConf, org, namespace, token)
		if err != nil {
			return nil, err
		}
		scs = append(scs, storClient)
	}
	return scs, nil
}

func createJWTToken(conf Config, org, namespace string) (string, error) {
	iyoClient := itsyouonline.NewClient(org, conf.IyoClientID, conf.IyoSecret)
	return iyoClient.CreateJWT(namespace, conf.iyoPerm())
}
