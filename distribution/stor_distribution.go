package distribution

import (
	"fmt"

	"github.com/zero-os/0-stor-lib/hash"
	"github.com/zero-os/0-stor-lib/stor"
)

// StorDistributor defines distributor that write
// the encoded data to 0-stor
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
	var scs []stor.Client
	for _, shard := range shards {
		storClient, err := stor.NewClient(shard, org, namespace, "")
		if err != nil {
			return nil, err
		}
		scs = append(scs, storClient)
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
		storClients: scs,
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
