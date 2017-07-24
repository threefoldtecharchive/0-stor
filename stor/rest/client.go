package rest

type Client struct {
}

func NewClient(addr string) *Client {
	return &Client{}
}

func (c *Client) Store(key, val []byte) error {
	return nil
}

func (c *Client) Get(key []byte) ([]byte, error) {
	return nil, nil
}
