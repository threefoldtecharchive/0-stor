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

package metastor

import (
	"errors"

	"github.com/threefoldtech/0-stor/client/processing"

	dbp "github.com/threefoldtech/0-stor/client/metastor/db"
	"github.com/threefoldtech/0-stor/client/metastor/encoding"
	"github.com/threefoldtech/0-stor/client/metastor/encoding/proto"
	"github.com/threefoldtech/0-stor/client/metastor/metatypes"
)

var (
	// ErrNotFound is the error returned by a metastor client,
	// in case metadata requested couldn't be found.
	ErrNotFound = dbp.ErrNotFound

	// ErrNilKey is the error returned by a metastor client,
	// in case a nil key is given as part of a request.
	ErrNilKey = errors.New("nil key given")
)

// Config is used to create a (metastor) Client.
type Config struct {
	// Database is required,
	// and is used to define te actual KV storage logic,
	// of the metadata in binary form.
	//
	// A client cannot be constructed if no database is given.
	Database dbp.DB

	// MarshalFuncPair is optional,
	// and is used to define custom marshal/unmarshal logic,
	// which transforms a Metadata struct to binary form and visa versa.
	// The (gogo) Proto(buf) marshal/unmarshal logic is used if no pair is given.
	//
	// A pair always have to given complete,
	// and a panic will be triggered if a partial one is given.
	MarshalFuncPair *encoding.MarshalFuncPair

	// ProcessorConstructor is optional,
	// and is used to pre- and postprocess the binary data,
	// prior to storage and just after fetching it.
	//
	// No pre- and postprocessing is applied,
	// in case no constructor is given.
	ProcessorConstructor ProcessorConstructor
}

// NewClientFromConfig creates a new metastor client from the given config
func NewClientFromConfig(namespace []byte, cfg Config) (*Client, error) {
	if cfg.Database == nil {
		return nil, errors.New("NewClient: no metastor database given")
	}

	// create the base encoder/decoder
	var (
		encode encoding.MarshalMetadata
		decode encoding.UnmarshalMetadata
	)
	if cfg.MarshalFuncPair != nil {
		if cfg.MarshalFuncPair.Marshal == nil {
			return nil, errors.New("NewClient: marshal function missing")
		}
		encode = cfg.MarshalFuncPair.Marshal
		if cfg.MarshalFuncPair.Unmarshal == nil {
			return nil, errors.New("NewClient: unmarshal function missing")
		}
		decode = cfg.MarshalFuncPair.Unmarshal
	} else {
		encode, decode = proto.MarshalMetadata, proto.UnmarshalMetadata
	}

	// if a processor is given,
	// we'll want to expand our encoder/decoders with some pre/post processing
	if cfg.ProcessorConstructor != nil {
		marshal := encode
		encode = func(md metatypes.Metadata) ([]byte, error) {
			processor, err := cfg.ProcessorConstructor()
			if err != nil {
				return nil, err
			}

			bytes, err := marshal(md)
			if err != nil {
				return nil, err
			}
			return processor.WriteProcess(bytes)
		}

		unmarshal := decode
		decode = func(bytes []byte, md *metatypes.Metadata) error {
			processor, err := cfg.ProcessorConstructor()
			if err != nil {
				return err
			}
			bytes, err = processor.ReadProcess(bytes)
			if err != nil {
				return err
			}
			return unmarshal(bytes, md)
		}
	}

	// return the created client
	return &Client{
		namespace: namespace,
		db:        cfg.Database,
		encode:    encode,
		decode:    decode,
	}, nil
}

// NewClient creates new client from the given DB.
// If privKey is not empty, it uses default encryption with the given encryptKey
// as private key
func NewClient(namespace string, db dbp.DB, privKey string) (*Client, error) {
	var (
		err    error
		config = Config{Database: db}
	)

	if len(privKey) == 0 {
		// create potentially insecure metastor storage
		return NewClientFromConfig([]byte(namespace), config)
	}

	// create the constructor which will create our encrypter-decrypter when needed
	config.ProcessorConstructor = func() (processing.Processor, error) {
		return processing.NewEncrypterDecrypter(
			processing.DefaultEncryptionType, []byte(privKey))
	}
	// ensure the constructor is valid,
	// as most errors (if not all) are static, and will only fail due to the given input,
	// meaning that if it can be created it now, it should be fine later on as well
	_, err = config.ProcessorConstructor()
	if err != nil {
		return nil, err
	}

	// create our full-configured metastor client,
	// including encryption support for our metadata in binary form
	return NewClientFromConfig([]byte(namespace), config)
}

// Client defines the client API of a metadata server.
// It is used to set, get and delete metadata.
// It is also used as an optional part of the the main 0-stor client,
// in order to fetch the metadata automatically for a given key.
//
// A Client is thread-safe.
type Client struct {
	namespace []byte
	db        dbp.DB
	encode    encoding.MarshalMetadata
	decode    encoding.UnmarshalMetadata
}

type (
	// UpdateMetadataFunc defines a function which receives an already stored metadata,
	// and which can modify the metadate, safely, prior to returning it.
	// In worst case it can return an error,
	// and that error will be propagated back to the user.
	UpdateMetadataFunc func(md metatypes.Metadata) (*metatypes.Metadata, error)

	// ProcessorConstructor is a constructor type which is used to create a unique
	// Processor for each goroutine where the Processor is needed within a pipeline.
	// This is required as a Processor is not thread-safe.
	ProcessorConstructor func() (processing.Processor, error)
)

// SetMetadata sets the metadata,
// using the key defined as part of the given metadata.
//
// An error is returned in case the metadata couldn't be set.
func (c *Client) SetMetadata(md metatypes.Metadata) error {
	if len(md.Key) == 0 {
		return ErrNilKey
	}
	md.Namespace = c.namespace

	bytes, err := c.encode(md)
	if err != nil {
		return err
	}

	return c.db.Set(md.Namespace, md.Key, bytes)
}

// UpdateMetadata updates already existing metadata,
// returning an error in case there is no metadata to be found for the given key.
// See `UpdateMetadataFunc` for more information about the required callback.
//
// UpdateMetadata panics when no callback is given.
func (c *Client) UpdateMetadata(key []byte, cb UpdateMetadataFunc) (*metatypes.Metadata, error) {
	if len(key) == 0 {
		return nil, ErrNilKey
	}

	metadata := new(metatypes.Metadata)
	err := c.db.Update(c.namespace, key, func(bytes []byte) ([]byte, error) {
		// decode the (fetched) metadata, so we can update it
		err := c.decode(bytes, metadata)
		if err != nil {
			return nil, err
		}

		// update the metadata, using the user-defined cb
		metadata, err = cb(*metadata)
		if err != nil {
			return nil, err
		}

		// encode the metadata once again,
		// and return it back for storage (if no error occurred)
		return c.encode(*metadata)
	})
	return metadata, err
}

// GetMetadata returns the metadata linked to the given key.
//
// An error is returned in case the linked data couldn't be found.
// ErrNotFound is returned in case the key couldn't be found.
// The returned data will always be non-nil in case no error was returned.
func (c *Client) GetMetadata(key []byte) (*metatypes.Metadata, error) {
	if len(key) == 0 {
		return nil, ErrNilKey
	}

	bytes, err := c.db.Get(c.namespace, key)
	if err != nil {
		return nil, err
	}

	var metadata metatypes.Metadata
	err = c.decode(bytes, &metadata)
	if err != nil {
		return nil, err
	}
	return &metadata, nil
}

// DeleteMetadata deletes the metadata linked to the given key.
// It is not considered an error if the metadata was already deleted.
//
// If an error is returned it should be assumed
// that the data couldn't be deleted and might still exist.
func (c *Client) DeleteMetadata(key []byte) error {
	if len(key) == 0 {
		return ErrNilKey
	}
	return c.db.Delete(c.namespace, key)
}

// ListKeys list all keys in the namespace,
// and exectute the given callback against each keys.
// Keys are sorted in lexicographically order
func (c *Client) ListKeys(cb dbp.ListCallback) error {
	return c.db.ListKeys(c.namespace, cb)
}

// Close any open resources of this metadata client.
func (c *Client) Close() error {
	return c.db.Close()
}
