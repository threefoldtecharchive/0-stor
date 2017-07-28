package client

import (
	"fmt"
	"os"

	"github.com/zero-os/0-stor/client/config"
	"github.com/zero-os/0-stor/client/itsyouonline"
	"github.com/zero-os/0-stor/client/lib/block"
	"github.com/zero-os/0-stor/client/meta"
	"github.com/zero-os/0-stor/client/pipe"
)

// Client defines 0-stor client
type Client struct {
	conf       *config.Config
	iyoClient  *itsyouonline.Client
	metaCli    *meta.Client
	storWriter block.Writer
}

// New creates new client
func New(confFile string) (*Client, error) {
	// read config
	f, err := os.Open(confFile)
	if err != nil {
		return nil, err
	}
	conf, err := config.NewFromReader(f)
	if err != nil {
		return nil, err
	}

	// create IYO client
	iyoClient := itsyouonline.NewClient(conf.Organization, conf.IyoClientID, conf.IyoSecret)

	// stor writer
	storWriter, err := pipe.NewWritePipe(conf, nil)
	if err != nil {
		return nil, err
	}

	// meta client
	metaCli, err := meta.NewClient(conf.MetaShards)
	if err != nil {
		return nil, err
	}

	return &Client{
		conf:       conf,
		metaCli:    metaCli,
		iyoClient:  iyoClient,
		storWriter: storWriter,
	}, nil

}

// Store stores payload with key=key
func (c *Client) Store(key, payload []byte) error {
	resp := c.storWriter.WriteBlock(payload)
	if resp.Err != nil {
		return resp.Err
	}
	if resp.Meta == nil {
		return nil
	}
	fmt.Printf("data stored with key (in metaserver) = %v\n", string(key))
	return c.metaCli.Put(string(key), *resp.Meta)
}

// Get fetch data for given key
func (c *Client) Get(key []byte) ([]byte, error) {
	rp, err := pipe.NewReadPipe(c.conf)
	if err != nil {
		return nil, err
	}

	// get the meta
	meta, err := c.metaCli.Get(string(key))
	if err != nil {
		return nil, err
	}

	metaBytes, err := meta.Bytes()
	if err != nil {
		return nil, err
	}

	return rp.ReadFull(metaBytes)
}
