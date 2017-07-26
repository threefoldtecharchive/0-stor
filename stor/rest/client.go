package rest

import (
	"fmt"

	client "github.com/zero-os/0-stor/store/client/goraml"
)

type Client struct {
	client *client.The_0_Stor
	nsid   string // namespace ID
}

func NewClient(addr, org, namespace, iyoJWTToken string) *Client {
	client := client.NewThe_0_Stor()
	client.BaseURI = addr
	client.AuthHeader = "Bearer " + iyoJWTToken
	return &Client{
		client: client,
		nsid:   org + "." + namespace,
	}
}

func (c *Client) Store(key, val []byte) error {
	obj := client.Object{
		Id:   string(val),
		Data: string(val),
	}
	_, resp, err := c.client.Namespaces.CreateObject(c.nsid, obj, nil, nil)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 && resp.StatusCode > 300 {
		return fmt.Errorf("invalid status code: %v", resp.StatusCode)
	}
	return nil
}

func (c *Client) Get(key []byte) ([]byte, error) {
	obj, resp, err := c.client.Namespaces.GetObject(string(key), c.nsid, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 && resp.StatusCode > 300 {
		return nil, fmt.Errorf("invalid status code: %v", resp.StatusCode)
	}
	return []byte(obj.Data), nil
}
