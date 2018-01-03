/*
 * Copyright (C) 2017-2018 GIG Technology NV and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package client

import (
	"errors"
	"io"
	"time"

	"github.com/zero-os/0-stor/client/datastor"
	storgrpc "github.com/zero-os/0-stor/client/datastor/grpc"
	"github.com/zero-os/0-stor/client/itsyouonline"
	"github.com/zero-os/0-stor/client/metastor"
	"github.com/zero-os/0-stor/client/metastor/etcd"
	"github.com/zero-os/0-stor/client/pipeline"
	"github.com/zero-os/0-stor/client/pipeline/storage"
)

var (
	// ErrNilKey is an error returned in case a nil key is given to a client method.
	ErrNilKey = errors.New("Client: nil/empty key given")
	// ErrNilContext is an error returned in case a context given to a client method is nil.
	ErrNilContext = errors.New("Client: nil context given")

	// ErrRepairSupport is returned when data is not stored using replication or distribution
	ErrRepairSupport = errors.New("data is not stored using replication or distribution, repair impossible")
)

// Client defines 0-stor client
type Client struct {
	dataPipeline   pipeline.Pipeline
	metastorClient metastor.Client
}

// NewClientFromConfig creates new 0-stor client using the given config,
// with (JWT Token) caching enabled only if required.
//
// JWT Token caching is required only if IYO credentials have been configured
// in the given config, which are to be used to create tokens using the IYO Web API.
//
// If JobCount is 0 or negative, the default JobCount will be used,
// as defined by the pipeline package.
func NewClientFromConfig(cfg Config, jobCount int) (*Client, error) {
	return newClientFromConfig(&cfg, jobCount, true)
}

// NewClientFromConfigWithoutCaching creates new 0-stor client using the given config,
// and with (JWT Token) caching disabled.
//
// If JobCount is 0 or negative, the default JobCount will be used,
// as defined by the pipeline package.
func NewClientFromConfigWithoutCaching(cfg Config, jobCount int) (*Client, error) {
	return newClientFromConfig(&cfg, jobCount, false)
}

func newClientFromConfig(cfg *Config, jobCount int, enableCaching bool) (*Client, error) {
	// create datastor cluster
	datastorCluster, err := createDataClusterFromConfig(cfg, enableCaching)
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
	metastorClient, err := etcd.NewClient(cfg.MetaStor.Shards, nil)
	if err != nil {
		return nil, err
	}
	return NewClient(metastorClient, dataPipeline), nil
}

func createDataClusterFromConfig(cfg *Config, enableCaching bool) (datastor.Cluster, error) {
	if cfg.IYO == (itsyouonline.Config{}) {
		// create datastor cluster without the use of IYO-backed JWT Tokens,
		// this will only work if all shards use zstordb servers that
		// do not require any authentication
		return storgrpc.NewCluster(cfg.DataStor.Shards, cfg.Namespace, nil)
	}

	// create IYO client
	client, err := itsyouonline.NewClient(cfg.IYO)
	if err != nil {
		return nil, err
	}

	var tokenGetter datastor.JWTTokenGetter
	// create JWT Token Getter (Using the earlier created IYO Client)
	tokenGetter, err = datastor.JWTTokenGetterUsingIYOClient(cfg.IYO.Organization, client)
	if err != nil {
		return nil, err
	}

	if enableCaching {
		// create cached token getter from this getter, using the default bucket size and count
		tokenGetter, err = datastor.CachedJWTTokenGetter(tokenGetter, -1, -1)
		if err != nil {
			return nil, err
		}
	}

	// create datastor cluster, with the use of IYO-backed JWT Tokens
	return storgrpc.NewCluster(cfg.DataStor.Shards, cfg.Namespace, tokenGetter)
}

// NewClient creates a 0-stor client,
// with the data (zstordb) cluster already created,
// used to read/write object data, as well as the metastor client,
// which is used to read/write the metadata of the objects.
func NewClient(metaClient metastor.Client, dataPipeline pipeline.Pipeline) *Client {
	if metaClient == nil {
		panic("0-stor Client: no metastor client given")
	}
	if dataPipeline == nil {
		panic("0-stor Client: no data pipeline given")
	}
	return &Client{
		dataPipeline:   dataPipeline,
		metastorClient: metaClient,
	}
}

// Write writes the data to a 0-stor cluster,
// storing the metadata using the internal metastor client.
func (c *Client) Write(key []byte, r io.Reader) (*metastor.Metadata, error) {
	if len(key) == 0 {
		return nil, ErrNilKey // ensure a key is given
	}

	// process and write the data
	chunks, err := c.dataPipeline.Write(r)
	if err != nil {
		return nil, err
	}

	// create new metadata, as we'll overwrite either way
	now := EpochNow()
	md := metastor.Metadata{
		Key:            key,
		CreationEpoch:  now,
		LastWriteEpoch: now,
	}

	// set/update chunks and size in metadata
	md.Chunks = chunks
	for _, chunk := range chunks {
		md.Size += chunk.Size
	}

	// store metadata
	err = c.metastorClient.SetMetadata(md)
	return &md, err
}

// Read reads the data, from the 0-stor cluster,
// using the reference information fetched from the storage-retrieved metadata
// (which is linked to the given key).
func (c *Client) Read(key []byte, w io.Writer) error {
	if len(key) == 0 {
		return ErrNilKey // ensure a key is given
	}
	meta, err := c.metastorClient.GetMetadata(key)
	if err != nil {
		return err
	}
	return c.dataPipeline.Read(meta.Chunks, w)
}

// ReadWithMeta reads the data, from the 0-stor cluster,
// using the reference information fetched from the given metadata.
func (c *Client) ReadWithMeta(meta metastor.Metadata, w io.Writer) error {
	return c.dataPipeline.Read(meta.Chunks, w)
}

// Delete deletes the data, from the 0-stor cluster,
// using the reference information fetched from the metadata (which is linked to the given key).
func (c *Client) Delete(key []byte) error {
	if len(key) == 0 {
		return ErrNilKey // ensure a key is given
	}
	meta, err := c.metastorClient.GetMetadata(key)
	if err != nil {
		return err
	}
	return c.DeleteWithMeta(*meta)
}

// DeleteWithMeta deletes the data, from the 0-stor cluster,
// using the reference information fetched from the given metadata
// (which is linked to the given key).
func (c *Client) DeleteWithMeta(meta metastor.Metadata) error {
	// delete data
	err := c.dataPipeline.Delete(meta.Chunks)
	if err != nil {
		return err
	}
	// delete metadata
	return c.metastorClient.DeleteMetadata(meta.Key)
}

// Check gets the status of data stored in a 0-stor cluster.
// It does so using the chunks stored as metadata, after fetching those, using the metastor client.
// If the metadata cannot be fetched or the status of a/the data chunk(s) cannot be retrieved,
// an error will be returned. Otherwise CheckStatusInvalid indicates the data is invalid and non-repairable,
// Any other value indicates the data is readable, but if it's not optimal, it could use a repair.
func (c *Client) Check(key []byte, fast bool) (storage.CheckStatus, error) {
	if len(key) == 0 {
		return storage.CheckStatus(0), ErrNilKey // ensure a key is given
	}
	meta, err := c.metastorClient.GetMetadata(key)
	if err != nil {
		return storage.CheckStatus(0), err
	}
	return c.dataPipeline.Check(meta.Chunks, fast)
}

// CheckWithMeta gets the status of data stored in a 0-stor cluster.
// It does so using the chunks stored as metadata, after fetching those, using the metastor client.
// If the metadata cannot be fetched or the status of a/the data chunk(s) cannot be retrieved,
// an error will be returned. Otherwise CheckStatusInvalid indicates the data is invalid and non-repairable,
// Any other value indicates the data is readable, but if it's not optimal, it could use a repair.
func (c *Client) CheckWithMeta(meta metastor.Metadata, fast bool) (storage.CheckStatus, error) {
	return c.dataPipeline.Check(meta.Chunks, fast)
}

// Repair repairs broken data, whether it's needed or not.
//
// If the data is distributed and the amount of corrupted chunks is acceptable,
// we recreate the missing chunks.
//
// Id the data is replicated and we still have one valid replication, we create the missing replications
// until we reach the replication number configured in the config.
//
// if the data has not been distributed or replicated, we can't repair it,
// or if not enough shards are available we cannot repair it either.
func (c *Client) Repair(key []byte) (*metastor.Metadata, error) {
	if len(key) == 0 {
		return nil, ErrNilKey // ensure a key is given
	}

	// because of conflicts, the callback might be called multiple times,
	// hence why we want to only do the actual repairing once
	var (
		repairedChunks       []metastor.Chunk
		totalSizeAfterRepair int64
		repairEpoch          int64
	)
	return c.metastorClient.UpdateMetadata(key,
		func(meta metastor.Metadata) (*metastor.Metadata, error) {
			// repair if not yet repaired
			if repairEpoch == 0 {
				var err error
				// repair the chunks (if possible)
				repairedChunks, err = c.dataPipeline.Repair(meta.Chunks)
				if err != nil {
					if err == storage.ErrNotSupported {
						return nil, ErrRepairSupport
					}
					return nil, err
				}
				// create the last-write epoch here,
				// such that this time is correct,
				// even when we have to retry multiple times, due to conflicts
				repairEpoch = EpochNow()
				// do the size computation here,
				// such that we only have to compute it once
				for _, chunk := range repairedChunks {
					totalSizeAfterRepair += chunk.Size
				}
			}

			// update chunks
			meta.Chunks = repairedChunks
			// update total size
			meta.Size = totalSizeAfterRepair
			// update last write epoch, as we have written while repairing
			meta.LastWriteEpoch = repairEpoch

			// return the updated metadata
			return &meta, nil
		})
}

// Close the client and all its used (internal/indirect) resources.
func (c *Client) Close() error {
	var ce closeErrors
	err := c.metastorClient.Close()
	if err != nil {
		ce = append(ce, err)
	}
	err = c.dataPipeline.Close()
	if err != nil {
		ce = append(ce, err)
	}
	if len(ce) > 0 {
		return ce
	}
	return nil
}

// EpochNow returns the current time,
// expressed in nano seconds, within the UTC timezone, in the epoch (unix) format.
func EpochNow() int64 {
	return time.Now().UTC().UnixNano()
}

type closeErrors []error

// Error implements error.Error
func (ce closeErrors) Error() string {
	var str string
	for _, e := range ce {
		str += e.Error() + ";"
	}
	return str
}
