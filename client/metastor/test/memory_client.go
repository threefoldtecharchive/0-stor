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

package test

import (
	"sync"

	"github.com/zero-os/0-stor/client/metastor"
)

// NewClient creates new Metadata client,
// using an nothing but an in-memory map as its storage medium.
//
// This client is only meant for development and testing purposes,
// and shouldn't be used for anything serious,
// given that it will lose all data as soon as it goes out of scope.
func NewClient() *Client {
	return &Client{
		md:       make(map[string]metastor.Metadata),
		versions: make(map[string]uint64),
	}
}

// Client defines client to store metadata,
// storing the metadata directly in an in-memory map.
//
// This client is only meant for development and testing purposes,
// and shouldn't be used for anything serious,
// given that it will lose all data as soon as it goes out of scope.
type Client struct {
	md       map[string]metastor.Metadata
	versions map[string]uint64
	mux      sync.RWMutex
}

// SetMetadata implements metastor.Client.SetMetadata
func (c *Client) SetMetadata(data metastor.Metadata) error {
	if len(data.Key) == 0 {
		return metastor.ErrNilKey
	}

	keyStr := string(data.Key)
	c.mux.Lock()
	c.md[keyStr] = data
	c.versions[keyStr]++
	c.mux.Unlock()

	return nil
}

// UpdateMetadata implements metastor.Client.UpdateMetadata
func (c *Client) UpdateMetadata(key []byte, cb metastor.UpdateMetadataFunc) (*metastor.Metadata, error) {
	if cb == nil {
		panic("Metastor (Memory) Client: required UpdateMetadata CB is not given")
	}
	if len(key) == 0 {
		return nil, metastor.ErrNilKey
	}

	var (
		output *metastor.Metadata
		keyStr = string(key)
	)
	for {
		c.mux.RLock()
		input, ok := c.md[keyStr]
		version := c.versions[keyStr]
		c.mux.RUnlock()
		if !ok {
			return nil, metastor.ErrNotFound
		}

		var err error
		output, err = cb(input)
		if err != nil {
			return nil, err
		}

		c.mux.Lock()
		if c.versions[keyStr] != version {
			c.mux.Unlock()
			continue // retry once again
		}

		c.md[keyStr] = *output
		c.versions[keyStr]++
		c.mux.Unlock()
		break
	}

	// return actual stored output
	return output, nil
}

// GetMetadata implements metastor.Client.GetMetadata
func (c *Client) GetMetadata(key []byte) (*metastor.Metadata, error) {
	if len(key) == 0 {
		return nil, metastor.ErrNilKey
	}

	c.mux.RLock()
	data, ok := c.md[string(key)]
	c.mux.RUnlock()
	if !ok {
		return nil, metastor.ErrNotFound
	}

	return &data, nil
}

// DeleteMetadata implements metastor.Client.DeleteMetadata
func (c *Client) DeleteMetadata(key []byte) error {
	if len(key) == 0 {
		return metastor.ErrNilKey
	}

	keyStr := string(key)
	c.mux.Lock()
	delete(c.md, keyStr)
	delete(c.versions, keyStr)
	c.mux.Unlock()

	return nil
}

// Close implements metastor.Client.Close
func (c *Client) Close() error {
	c.mux.Lock()
	c.md = make(map[string]metastor.Metadata)
	c.versions = make(map[string]uint64)
	c.mux.Unlock()

	return nil
}

var (
	_ metastor.Client = (*Client)(nil)
)
