package client

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/zero-os/0-stor/client/pipeline/storage"
)

var (
	// ErrAllReplicateCorrupted is returned when a  All the replicate are corrupted
	ErrAllReplicateCorrupted = fmt.Errorf("All the replicate are corrupted, repair impossible")
	// ErrRepairSupport is returned when data is not stored using replication or distribution
	ErrRepairSupport = fmt.Errorf("data is not stored using replication or distribution, repair impossible")
)

// Repair repairs a broken file.
// If the file is distributed and the amount of corrupted chunks is acceptable,
// we recreate the missing chunks.
// Id the file is replicated and we still have one valid replicate, we create the missing replicate
// till we reach the replication number configured in the config
// if the file as not been distributed or replicated, we can't repair it,
// or if not enough shards are available we cannot repair it either.
func (c *Client) Repair(key []byte) error {
	log.Infof("Start repair of %x", key)
	meta, err := c.metastorClient.GetMetadata(key)
	if err != nil {
		log.Errorf("repair %x, error getting metadata :%v", key, err)
		return err
	}

	chunkStorage := c.dataPipeline.GetChunkStorage()
	for _, chunk := range meta.Chunks {
		cfg, err := chunkStorage.RepairChunk(storage.ChunkConfig{
			Size:    chunk.Size,
			Objects: chunk.Objects,
		})
		if err != nil {
			if err == storage.ErrNotSupported {
				return ErrRepairSupport
			}
			return err
		}
		chunk.Size = cfg.Size
		chunk.Objects = cfg.Objects
	}

	// update last write epoch, as we have written while repairing
	meta.LastWriteEpoch = time.Now().UnixNano()

	if err := c.metastorClient.SetMetadata(*meta); err != nil {
		log.Errorf("error writing metadata after repair: %v", err)
		return err
	}

	return nil
}
