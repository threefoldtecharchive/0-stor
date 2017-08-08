package client

import (
	"errors"
	"io"

	"github.com/zero-os/0-stor/client/config"
	"github.com/zero-os/0-stor/client/itsyouonline"
	"github.com/zero-os/0-stor/client/lib/block"
	"github.com/zero-os/0-stor/client/lib/chunker"
	"github.com/zero-os/0-stor/client/meta"
	"github.com/zero-os/0-stor/client/pipe"
)

var (
	errWriteFChunkerOnly = errors.New("WriteF only support chunker as first pipe")
	errReadFChunkerOnly  = errors.New("ReadF only support chunker as first pipe")
)

var _ (itsyouonline.NamespaceManager) = (*Client)(nil) // build time check that we implement stor.NamespaceManager interface

// Client defines 0-stor client
type Client struct {
	conf                          *config.Config
	metaCli                       *meta.Client
	itsyouonline.NamespaceManager //implement the NamespaceManager interface

	storWriter block.Writer
	storReader block.Reader
}

// New creates new client from the given config
func New(conf *config.Config) (*Client, error) {
	// append stor client to the end of pipe if needed
	conf.CheckAppendStorClient()

	// stor writer
	storWriter, err := pipe.NewWritePipe(conf, block.NewNilWriter(), nil)
	if err != nil {
		return nil, err
	}

	// stor reader
	storReader, err := pipe.NewReadPipe(conf)
	if err != nil {
		return nil, err
	}

	client := Client{
		conf:       conf,
		storWriter: storWriter,
		storReader: storReader,
	}

	if len(conf.MetaShards) > 0 {
		// meta client
		metaCli, err := meta.NewClient(conf.MetaShards)
		if err != nil {
			return nil, err
		}
		client.metaCli = metaCli
	}

	client.NamespaceManager = itsyouonline.NewClient(conf.Organization, conf.IYOAppID, conf.IYOSecret)

	return &client, nil
}

// WriteF writes the key with value taken from given io.Reader
// it currently only support `chunker` as first pipe
func (c *Client) WriteF(key []byte, r io.Reader) (*meta.Meta, error) {
	if !c.conf.ChunkerFirstPipe() {
		return nil, errWriteFChunkerOnly
	}

	md, err := meta.New(key, 0, nil)
	if err != nil {
		return md, err
	}

	wp, err := pipe.NewWritePipe(c.conf, block.NewNilWriter(), r)
	if err != nil {
		return nil, err
	}

	return wp.WriteBlock(key, nil, md)
}

// Write writes the key-value to the configured pipes.
func (c *Client) Write(key, val []byte) (*meta.Meta, error) {
	md, err := meta.New(key, 0, nil)
	if err != nil {
		return md, err
	}
	return c.storWriter.WriteBlock(key, val, md)
}

// ReadF read the key and write the result to the given io.Writer
// it currently only support `chunker` as first pipe
func (c *Client) ReadF(key []byte, w io.Writer) ([]byte, error) {
	if !c.conf.ChunkerFirstPipe() {
		return nil, errReadFChunkerOnly
	}

	readPipe, err := pipe.NewReadPipe(c.conf)
	if err != nil {
		return nil, err
	}

	rp0 := readPipe.Readers[len(readPipe.Readers)-1]
	chunkReader := rp0.(*chunker.BlockReader)
	return nil, chunkReader.Restore(key, w, readPipe)

}

// Read reads value with given key from the configured pipes.
func (c *Client) Read(key []byte) ([]byte, error) {
	return c.storReader.ReadBlock(key)
}
