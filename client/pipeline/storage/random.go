package storage

import (
	"errors"

	log "github.com/Sirupsen/logrus"
	"github.com/zero-os/0-stor/client/datastor"
)

// NewRandomObjectStorage creates a new RandomObjectStorage.
// See `RandomObjectStorage` for more information.
func NewRandomObjectStorage(cluster datastor.Cluster) (*RandomObjectStorage, error) {
	if cluster == nil {
		panic("no cluster given")
	}
	if cluster.ListedShardCount() < 1 {
		return nil, errors.New("RandomObjectStorage: at least one listed shard is required")
	}
	return &RandomObjectStorage{cluster: cluster}, nil
}

// RandomObjectStorage is the most simplest Storage implementation.
// For writing it only writes to one shard, randomly chosen.
// For reading it expects just, and only, one shard, to read from.
// Repairing is not supported for this storage for obvious reasons.
type RandomObjectStorage struct {
	cluster datastor.Cluster
}

// Write implements storage.ObjectStorage.Write
func (rs *RandomObjectStorage) Write(object datastor.Object) (ObjectConfig, error) {
	var (
		err   error
		shard datastor.Shard
	)

	// go through all shards, in pseudo-random fashion,
	// until the object could be written to one of them.
	it := rs.cluster.GetRandomShardIterator(nil)
	for it.Next() {
		shard = it.Shard()
		err = shard.SetObject(object)
		if err == nil {
			return ObjectConfig{
				Key:      object.Key,
				Shards:   []string{shard.Identifier()},
				DataSize: len(object.Data),
			}, nil
		}
		log.Errorf("failed to write %q to random shard %q: %v",
			object.Key, shard.Identifier(), err)
	}
	return ObjectConfig{}, ErrShardsUnavailable
}

// Read implements storage.ObjectStorage.Read
func (rs *RandomObjectStorage) Read(cfg ObjectConfig) (datastor.Object, error) {
	if len(cfg.Shards) != 1 {
		return datastor.Object{}, ErrUnexpectedShardsCount
	}

	shard, err := rs.cluster.GetShard(cfg.Shards[0])
	if err != nil {
		return datastor.Object{}, err
	}

	object, err := shard.GetObject(cfg.Key)
	if err != nil {
		return datastor.Object{}, err
	}

	if len(object.Data) != cfg.DataSize {
		return *object, ErrInvalidDataSize
	}
	return *object, nil
}

// Check implements storage.ObjectStorage.Check
func (rs *RandomObjectStorage) Check(cfg ObjectConfig, fast bool) (ObjectCheckStatus, error) {
	if len(cfg.Shards) != 1 {
		return ObjectCheckStatusInvalid, ErrUnexpectedShardsCount
	}

	shard, err := rs.cluster.GetShard(cfg.Shards[0])
	if err != nil {
		return ObjectCheckStatusInvalid, nil
	}

	status, err := shard.GetObjectStatus(cfg.Key)
	if err != nil || status != datastor.ObjectStatusOK {
		return ObjectCheckStatusInvalid, nil
	}

	return ObjectCheckStatusOptimal, nil
}

// Repair implements storage.ObjectStorage.Repair
func (rs *RandomObjectStorage) Repair(cfg ObjectConfig) (ObjectConfig, error) {
	return ObjectConfig{}, ErrNotSupported
}

var (
	_ ObjectStorage = (*RandomObjectStorage)(nil)
)
