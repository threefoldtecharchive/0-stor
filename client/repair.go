package client

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/zero-os/0-stor/client/components/storage"
)

var (
	// ErrAllReplicateCorrupted is returned when a  All the replicate are corrupted
	ErrAllReplicateCorrupted = fmt.Errorf("All the replicate are corrupted, repair impossible")
	// ErrRepairSupport is returned when a block is not using replication or distribution
	ErrRepairSupport = fmt.Errorf("block is not using replication or distribution, repair impossible")
)

// Repair repairs a broken file.
// If the file is distributed and the ammout of corrupted chunks is acceptable,
// we recreate the missing chunks.
// Id the file is replicated and we still have one valid replicate, we create the missing replicate
// till we reach the replication number configured in the policy
// if the file as not been distributed or replicated, we can't repair it
func (c *Client) Repair(key []byte) error {
	log.Infof("Start repair of %x", key)
	meta, err := c.metaCli.GetMetadata(key)
	if err != nil {
		log.Errorf("repair %x, error getting metadata :%v", key, err)
		return err
	}

	for _, chunk := range meta.Chunks {
		cfg, err := c.storage.Repair(storage.ObjectConfig{
			Key:      chunk.Key,
			Shards:   chunk.Shards,
			DataSize: int(chunk.Size),
		})
		if err != nil {
			if err == storage.ErrNotSupported {
				return ErrRepairSupport
			}
			return err
		}
		chunk.Shards = cfg.Shards
		chunk.Size = int64(cfg.DataSize)
		chunk.Key = cfg.Key
	}

	if err := c.metaCli.SetMetadata(*meta); err != nil {
		log.Errorf("error writing metadata after repair: %v", err)
		return err
	}

	return nil
}
