package storage

import (
	"errors"

	"github.com/zero-os/0-stor/client/metastor"

	log "github.com/Sirupsen/logrus"
	"github.com/zero-os/0-stor/client/datastor"
)

// NewRandomChunkStorage creates a new RandomChunkStorage.
// See `RandomChunkStorage` for more information.
func NewRandomChunkStorage(cluster datastor.Cluster) (*RandomChunkStorage, error) {
	if cluster == nil {
		panic("no cluster given")
	}
	if cluster.ListedShardCount() < 1 {
		return nil, errors.New("RandomChunkStorage: at least one listed shard is required")
	}
	return &RandomChunkStorage{cluster: cluster}, nil
}

// RandomChunkStorage is the most simplest Storage implementation.
// For writing it only writes to one shard, randomly chosen.
// For reading it expects just, and only, one shard, to read from.
// Repairing is not supported for this storage for obvious reasons.
type RandomChunkStorage struct {
	cluster datastor.Cluster
}

// WriteChunk implements storage.ObjectStorage.WriteChunk
func (rs *RandomChunkStorage) WriteChunk(data []byte) (*ChunkConfig, error) {
	var (
		key   []byte
		err   error
		shard datastor.Shard
	)

	// go through all shards, in pseudo-random fashion,
	// until the data could be written to one of them.
	it := rs.cluster.GetRandomShardIterator(nil)
	for it.Next() {
		shard = it.Shard()
		key, err = shard.CreateObject(data)
		if err == nil {
			return &ChunkConfig{
				Size: int64(len(data)),
				Objects: []metastor.Object{
					metastor.Object{
						Key:     key,
						ShardID: shard.Identifier(),
					},
				},
			}, nil
		}
		log.Errorf("failed to write data to random shard %q: %v",
			shard.Identifier(), err)
	}
	return nil, ErrShardsUnavailable
}

// ReadChunk implements storage.ObjectStorage.ReadChunk
func (rs *RandomChunkStorage) ReadChunk(cfg ChunkConfig) ([]byte, error) {
	if len(cfg.Objects) != 1 {
		return nil, ErrUnexpectedObjectCount
	}
	obj := &cfg.Objects[0]

	shard, err := rs.cluster.GetShard(obj.ShardID)
	if err != nil {
		return nil, err
	}

	object, err := shard.GetObject(obj.Key)
	if err != nil {
		return nil, err
	}

	if int64(len(object.Data)) != cfg.Size {
		return object.Data, ErrInvalidDataSize
	}
	return object.Data, nil
}

// CheckChunk implements storage.ObjectStorage.CheckChunk
func (rs *RandomChunkStorage) CheckChunk(cfg ChunkConfig, fast bool) (CheckStatus, error) {
	if len(cfg.Objects) != 1 {
		return CheckStatusInvalid, ErrUnexpectedObjectCount
	}
	obj := &cfg.Objects[0]

	shard, err := rs.cluster.GetShard(obj.ShardID)
	if err != nil {
		return CheckStatusInvalid, nil
	}

	status, err := shard.GetObjectStatus(obj.Key)
	if err != nil || status != datastor.ObjectStatusOK {
		return CheckStatusInvalid, nil
	}

	return CheckStatusOptimal, nil
}

// RepairChunk implements storage.ObjectStorage.RepairChunk
func (rs *RandomChunkStorage) RepairChunk(cfg ChunkConfig) (*ChunkConfig, error) {
	return nil, ErrNotSupported
}

var (
	_ ChunkStorage = (*RandomChunkStorage)(nil)
)
