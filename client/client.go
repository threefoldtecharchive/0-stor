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
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/threefoldtech/0-stor/client/datastor"
	"github.com/threefoldtech/0-stor/client/datastor/pipeline"
	"github.com/threefoldtech/0-stor/client/datastor/pipeline/storage"
	"github.com/threefoldtech/0-stor/client/datastor/zerodb"
	"github.com/threefoldtech/0-stor/client/metastor"
	"github.com/threefoldtech/0-stor/client/metastor/metatypes"

	log "github.com/sirupsen/logrus"
)

var (
	// ErrNilKey is an error returned in case a nil key is given to a client method.
	ErrNilKey = errors.New("Client: nil/empty key given")
	// ErrNilContext is an error returned in case a context given to a client method is nil.
	ErrNilContext = errors.New("Client: nil context given")

	// ErrRepairSupport is returned when data is not stored using replication or distribution
	ErrRepairSupport = errors.New("data is not stored using replication or distribution, repair impossible")

	// ErrInvalidReadRange is returned when given read range is not valid
	ErrInvalidReadRange = errors.New("invalid read range")
)

// Client defines 0-stor client
type Client struct {
	dataPipeline   pipeline.Pipeline
	metastorClient *metastor.Client
}

// NewClientFromConfig creates new 0-stor client using the given config.
//
// If JobCount is 0 or negative, the default JobCount will be used,
// as defined by the pipeline package.
func NewClientFromConfig(cfg Config, metastorClient *metastor.Client, jobCount int) (*Client, error) {
	// create datastor cluster
	datastorCluster, err := createDataClusterFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	// create data pipeline, using our datastor cluster
	dataPipeline, err := pipeline.NewPipeline(cfg.DataStor.Pipeline, datastorCluster, jobCount)
	if err != nil {
		return nil, err
	}

	return NewClient(metastorClient, dataPipeline), nil
}

func createDataClusterFromConfig(cfg Config) (datastor.Cluster, error) {
	// optionally create the global datastor TLS config
	tlsConfig, err := createTLSConfigFromDatastorTLSConfig(&cfg.DataStor.TLS)
	if err != nil {
		return nil, err
	}
	return zerodb.NewCluster(cfg.DataStor.Shards, cfg.Password, cfg.Namespace, tlsConfig)
}

func createTLSConfigFromDatastorTLSConfig(config *DataStorTLSConfig) (*tls.Config, error) {
	if config == nil || !config.Enabled {
		return nil, nil
	}
	tlsConfig := &tls.Config{
		MinVersion: config.MinVersion.VersionTLSOrDefault(tls.VersionTLS11),
		MaxVersion: config.MaxVersion.VersionTLSOrDefault(tls.VersionTLS12),
	}

	if config.ServerName != "" {
		tlsConfig.ServerName = config.ServerName
	} else {
		log.Warning("TLS is configured to skip verificaitons of certs, " +
			"making the client susceptible to man-in-the-middle attacks!!!")
		tlsConfig.InsecureSkipVerify = true
	}

	if config.RootCA == "" {
		var err error
		tlsConfig.RootCAs, err = x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("failed to create datastor TLS config: %v", err)
		}
	} else {
		tlsConfig.RootCAs = x509.NewCertPool()
		caFile, err := ioutil.ReadFile(config.RootCA)
		if err != nil {
			return nil, err
		}
		if !tlsConfig.RootCAs.AppendCertsFromPEM(caFile) {
			return nil, fmt.Errorf("error reading CA file '%s', while creating datastor TLS config: %v",
				config.RootCA, err)
		}
	}

	return tlsConfig, nil
}

// NewClient creates a 0-stor client,
// with the data (zstordb) cluster already created,
// used to read/write object data, as well as the metastor client,
// which is used to read/write the metadata of the objects.
// if metaClient is not nil, the metadata will be written at Write operation
// using the given metaClient.
func NewClient(metaClient *metastor.Client, dataPipeline pipeline.Pipeline) *Client {
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
func (c *Client) Write(key []byte, r io.Reader) (*metatypes.Metadata, error) {
	return c.write(key, r, nil)
}

// WriteWithUserMeta writes the data to a 0-stor cluster,
// storing the metadata using the internal metastor client.
// The given user defined metadata will be stored in the `UserDefined` field
// of the metadata.
func (c *Client) WriteWithUserMeta(key []byte, r io.Reader, userDefined map[string]string) (*metatypes.Metadata, error) {
	return c.write(key, r, userDefined)
}

func (c *Client) write(key []byte, r io.Reader, userDefinedMeta map[string]string) (*metatypes.Metadata, error) {
	if len(key) == 0 {
		return nil, ErrNilKey // ensure a key is given
	}

	if r == nil {
		return nil, errors.New("no reader given to read from")
	}

	// used to count the total size of bytes read from r
	rc := &readCounter{r: r}

	// process and write the data
	chunks, err := c.dataPipeline.Write(rc)
	if err != nil {
		return nil, err
	}

	// create new metadata, as we'll overwrite either way
	now := EpochNow()
	md := metatypes.Metadata{
		Key:            key,
		Size:           rc.Size(),
		CreationEpoch:  now,
		LastWriteEpoch: now,
		ChunkSize:      int32(c.dataPipeline.ChunkSize()),
		UserDefined:    userDefinedMeta,
	}

	// set/update chunks and size in metadata
	md.Chunks = chunks
	for _, chunk := range chunks {
		md.StorageSize += chunk.Size
	}

	// store metadata
	if c.metastorClient != nil {
		err = c.metastorClient.SetMetadata(md)
	}
	return &md, err
}

// Read reads the data, from the 0-stor cluster,
// using the reference information fetched from the storage-retrieved metadata
// (which is linked to the given key).
/*func (c *Client) Read(key []byte, w io.Writer) error {
	if len(key) == 0 {
		return ErrNilKey // ensure a key is given
	}
	meta, err := c.metastorClient.GetMetadata(key)
	if err != nil {
		return err
	}
	return c.dataPipeline.Read(meta.Chunks, w)
}*/

// Read reads the data, from the 0-stor cluster,
// using the reference information fetched from the given metadata.
func (c *Client) Read(meta metatypes.Metadata, w io.Writer) error {
	return c.dataPipeline.Read(meta.Chunks, w)
}

// ReadRange reads data with the given offset & length.
func (c *Client) ReadRange(meta metatypes.Metadata, w io.Writer, offset, length int64) error {
	// in case we don't split the data,
	// no need to worry about which chunk to proceed
	if meta.ChunkSize == 0 {
		return c.dataPipeline.Read(meta.Chunks, &rangeWriter{
			w:      w,
			offset: offset,
			length: length,
		})
	}

	endOffset := offset + length

	// make sure it has valid range
	if endOffset > meta.Size {
		return ErrInvalidReadRange
	}

	var (
		startChunkIdx = int(offset / int64(meta.ChunkSize))
		endChunkIdx   = int(endOffset / int64(meta.ChunkSize))
	)
	if int(endOffset)%int(meta.ChunkSize) > 0 {
		endChunkIdx++
	}

	rw := &rangeWriter{
		offset: offset % int64(meta.ChunkSize),
		length: length,
		w:      w,
	}
	return c.dataPipeline.Read(meta.Chunks[startChunkIdx:endChunkIdx], rw)
}

// range writer is writer that only write data
// in the given offset & length range
type rangeWriter struct {
	offset int64
	length int64
	w      io.Writer
}

// Write implements io.Writer.Write
func (rw *rangeWriter) Write(p []byte) (int, error) {
	if rw.length <= 0 {
		return 0, io.EOF
	}

	var (
		startIdx = int(rw.offset) // default start data index is current offset
		endIdx   = len(p)         // default end data index is length of the data
		dataLen  = len(p)
	)

	// if offset > 0 dataLen, we can ignore the data
	if rw.offset > int64(dataLen) {
		rw.offset -= int64(dataLen)
		return dataLen, nil // we need to return len(p) in case of err==nil
	}

	if rw.length+int64(startIdx) < int64(endIdx) {
		endIdx = startIdx + int(rw.length)
	}

	// call the underlying writer
	// propagate the error if happens
	n, err := rw.w.Write(p[startIdx:endIdx])

	// modify internal length & offset
	rw.length -= int64(n)
	if rw.offset > 0 {
		rw.offset = 0
	}

	if err != nil {
		return n, err
	}

	return dataLen, nil // always return len p in case of err == nil
}

// Delete deletes the data, from the 0-stor cluster,
// using the reference information fetched from the given metadata
// (which is linked to the given key).
func (c *Client) Delete(meta metatypes.Metadata) error {
	// delete data
	err := c.dataPipeline.Delete(meta.Chunks)
	if err != nil {
		return err
	}
	// delete metadata
	if c.metastorClient != nil {
		return c.metastorClient.DeleteMetadata(meta.Key)
	}

	return nil
}

// Check gets the status of data stored in a 0-stor cluster.
// It does so using the chunks stored as metadata
// CheckStatusInvalid indicates the data is invalid and non-repairable,
// Any other value indicates the data is readable, but if it's not optimal, it could use a repair.
func (c *Client) Check(meta metatypes.Metadata, fast bool) (storage.CheckStatus, error) {
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
func (c *Client) Repair(md metatypes.Metadata) (*metatypes.Metadata, error) {

	// because of conflicts, the callback might be called multiple times,
	// hence why we want to only do the actual repairing once
	var (
		repairedChunks       []metatypes.Chunk
		totalSizeAfterRepair int64
		repairEpoch          int64
	)
	repair := func(meta metatypes.Metadata) (*metatypes.Metadata, error) {
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
		meta.StorageSize = totalSizeAfterRepair
		// update last write epoch, as we have written while repairing
		meta.LastWriteEpoch = repairEpoch

		// return the updated metadata
		return &meta, nil
	}

	if c.metastorClient != nil {
		return c.metastorClient.UpdateMetadata(md.Key, repair)
	}

	return repair(md)
}

// Close the client and all its used (internal/indirect) resources.
func (c *Client) Close() error {
	var ce closeErrors

	if c.metastorClient != nil {
		if err := c.metastorClient.Close(); err != nil {
			ce = append(ce, err)
		}
	}

	if err := c.dataPipeline.Close(); err != nil {
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

// readCounter is a io.Reader wrapper that counts the
// total of bytes read by the underlying reader
type readCounter struct {
	r    io.Reader
	size int64
}

// Read implement the io.Reader interface
func (rc *readCounter) Read(p []byte) (n int, err error) {
	n, err = rc.r.Read(p)
	rc.size += int64(n)
	return n, err
}

// Size return the total of bytes read by the underlying reader
func (rc *readCounter) Size() int64 {
	return rc.size
}
