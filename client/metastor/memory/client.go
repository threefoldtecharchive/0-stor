package memory

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
	return &Client{md: make(map[string]metastor.Data)}
}

// Client defines client to store metadata,
// storing the metadata directly in an in-memory map.
//
// This client is only meant for development and testing purposes,
// and shouldn't be used for anything serious,
// given that it will lose all data as soon as it goes out of scope.
type Client struct {
	md  map[string]metastor.Data
	mux sync.RWMutex
}

// SetMetadata implements metastor.Client.SetMetadata
func (c *Client) SetMetadata(data metastor.Data) error {
	if data.Key == nil {
		return metastor.ErrNilKey
	}

	c.mux.Lock()
	c.md[string(data.Key)] = data
	c.mux.Unlock()

	return nil
}

// GetMetadata implements metastor.Client.GetMetadata
func (c *Client) GetMetadata(key []byte) (*metastor.Data, error) {
	if key == nil {
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
	if key == nil {
		return metastor.ErrNilKey
	}

	c.mux.Lock()
	delete(c.md, string(key))
	c.mux.Unlock()

	return nil
}

// Close implements metastor.Client.Close
func (c *Client) Close() error {
	c.mux.Lock()
	c.md = make(map[string]metastor.Data)
	c.mux.Unlock()

	return nil
}

var (
	_ metastor.Client = (*Client)(nil)
)
