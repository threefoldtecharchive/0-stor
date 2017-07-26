package distribution

import (
	"fmt"

	"github.com/zero-os/0-stor-lib/hash"
	"github.com/zero-os/0-stor-lib/stor"
)

// StorDistributor defines distributor that use 0-stor rest/grpc clients
// as Writers.
// It create one stor.Client for each shard
type StorDistributor struct {
	enc         *Encoder
	hasher      *hash.Hasher
	storClients []stor.Client
}

// NewStorDistributor creates new StorDistributor
func NewStorDistributor(conf Config, shards []string, org, namespace string) (*StorDistributor, error) {
	if len(shards) < conf.NumPieces() {
		return nil, fmt.Errorf("invalid number of shards=%v, expected=%v", len(shards), conf.NumPieces())
	}

	// stor clients
	storClients, err := createStorClients(shards, org, namespace, "")
	if err != nil {
		return nil, err
	}

	// encoder
	enc, err := NewEncoder(conf.K, conf.M)
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
	}, nil
}

// Write implements io.Writer
func (sd StorDistributor) Write(data []byte) (written int, err error) {
	key := sd.hasher.Hash(data)
	encoded, err := sd.enc.Encode(data)
	if err != nil {
		return
	}

	for i, piece := range encoded {
		if err = sd.storClients[i].Store(key, piece); err != nil {
			return
		}
		written += len(piece)
	}
	return
}

// StorRestorer defines distributor that get the data
// from 0-stor
type StorRestorer struct {
	dec         *Decoder
	storClients []stor.Client
}

func NewStorRestorer(conf Config, shards []string, org, namespace string) (*StorRestorer, error) {
	// stor clients
	storClients, err := createStorClients(shards, org, namespace, "")
	if err != nil {
		return nil, err
	}

	dec, err := NewDecoder(conf.K, conf.M)
	if err != nil {
		return nil, err
	}

	return &StorRestorer{
		dec:         dec,
		storClients: storClients,
	}, nil
}

// ReadAll implements allreader.ReadAll
func (sr StorRestorer) ReadAll(key []byte) ([]byte, error) {
	var chunkLen int
	chunks := make([][]byte, sr.dec.k+sr.dec.m)

	// read all chunks from stor.Clients
	for i, sc := range sr.storClients {
		data, err := sc.Get(key)
		if err != nil {
		} else {
			chunks[i] = data
			chunkLen = len(data)
		}
	}

	origLen := chunkLen * (sr.dec.k) // TODO get from meta
	origLen = 318
	// decode
	decoded, err := sr.dec.Decode(chunks, origLen)
	return decoded, err
}

func createStorClients(shards []string, org, namespace, iyoJWTClient string) ([]stor.Client, error) {
	var scs []stor.Client
	for _, shard := range shards {
		storClient, err := stor.NewClient(shard, org, namespace, iyoJWTClient)
		if err != nil {
			return nil, err
		}
		scs = append(scs, storClient)
	}
	return scs, nil
}
