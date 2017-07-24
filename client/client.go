package client

import (
	"io"
	"os"

	"github.com/zero-os/0-stor-lib/client/itsyouonline"
	"github.com/zero-os/0-stor-lib/config"
	"github.com/zero-os/0-stor-lib/distribution"
)

type Client struct {
	conf       *config.Config
	iyoClient  *itsyouonline.Client
	storWriter io.Writer
	ecEncoder  *distribution.Encoder
}

func NewClient(confFile string) (*Client, error) {
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
	storWriter, err := conf.CreatePipeWriter(nil)
	if err != nil {
		return nil, err
	}

	return &Client{
		conf:       conf,
		iyoClient:  iyoClient,
		storWriter: storWriter,
	}, nil

}

func (c *Client) Store(payload []byte) (int, error) {
	return c.storWriter.Write(payload)
}
