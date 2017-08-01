package rest

import (
	"encoding/base64"
	"fmt"

	client "github.com/zero-os/0-stor/client/goraml"
)

// Client is 0-stor client.
// it use go-raml generated client underneath
type Client struct {
	client *client.The_0_Stor
	nsid   string // namespace ID
}

// NewClient creates Client
func NewClient(addr, org, namespace, iyoJWTToken string) *Client {
	client := client.NewThe_0_Stor()
	client.BaseURI = addr

	if iyoJWTToken != "" {
		client.AuthHeader = "Bearer " + iyoJWTToken
	}

	return &Client{
		client: client,
		nsid:   namespace,
	}
}

// Store store the val to 0-stor with given key as id
func (c *Client) Store(key, val []byte) (string, error) {
	obj := client.Object{
		Id:   base64.URLEncoding.EncodeToString(key),
		Data: base64.StdEncoding.EncodeToString(val),
	}
	_, resp, err := c.client.Namespaces.CreateObject(c.nsid, obj, nil, nil)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 && resp.StatusCode > 300 {
		return "", fmt.Errorf("invalid status code: %v", resp.StatusCode)
	}
	return obj.Id, nil
}

// Get gets data from 0-stor server
func (c *Client) Get(key []byte) ([]byte, error) {
	obj, resp, err := c.client.Namespaces.GetObject(string(key), c.nsid, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 && resp.StatusCode > 300 {
		return nil, fmt.Errorf("invalid status code: %v", resp.StatusCode)
	}
	return base64.StdEncoding.DecodeString(obj.Data)
}
