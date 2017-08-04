package rest

import (
	"encoding/base64"
	"fmt"

	client "github.com/zero-os/0-stor/client/goraml"
	"github.com/zero-os/0-stor/client/stor/common"
)

func encodeID(id []byte) string {
	return base64.URLEncoding.EncodeToString(id)
}

func decodeID(id string) ([]byte, error) {
	return base64.URLEncoding.DecodeString(id)
}

func encodeData(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func decodeData(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}

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
		nsid:   org + "_0stor_" + namespace,
	}
}

// Namespace gets detail view about namespace
func (c *Client) NamespaceGet() (common.Namespace, error) {
	return common.Namespace{}, nil
}

// ReservationList return a list of all the existing reservation
func (c *Client) ReservationList() ([]common.Reservation, error) {
	return nil, nil
}

// ReservationCreate creates a reservation.
// size is Storage size you want to reserve in MiB.
// period is number of days the reservation is valid
func (c *Client) ReservationCreate(size, period int) (common.Reservation, error) {
	return common.Reservation{}, nil
}

// ReservationGet returns information about a reservation
func (c *Client) ReservationGet(id []byte) (common.Reservation, error) {
	return common.Reservation{}, nil
}

// ReservationUpdate renew an existing reservation
func (c *Client) ReservationUpdate(id []byte, size, period int) error {
	return nil
}

// ObjectList lists keys of the object in the namespace
func (c *Client) ObjectList(page, perPage int) ([]string, error) {
	return nil, nil
}

// ObjectCreate creates an object
func (c *Client) ObjectCreate(id, data []byte, refList []string) (*common.Object, error) {
	obj := client.Object{
		Id:   encodeID(id),
		Data: encodeData(data),
		//ReferenceList: refList,
	}

	obj, resp, err := c.client.Namespaces.CreateObject(c.nsid, obj, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 && resp.StatusCode > 300 {
		return nil, fmt.Errorf("invalid status code: %v", resp.StatusCode)
	}
	return newCommonObject(id, data, obj.ReferenceList), nil
}

// ObjectGet retrieve object from the store
func (c *Client) ObjectGet(id []byte) (*common.Object, error) {
	obj, resp, err := c.client.Namespaces.GetObject(encodeID(id), c.nsid, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 && resp.StatusCode > 300 {
		return nil, fmt.Errorf("invalid status code: %v", resp.StatusCode)
	}
	decodedData, err := decodeData(obj.Data)
	if err != nil {
		return nil, err
	}
	return newCommonObject(id, decodedData, obj.ReferenceList), nil
}

// ObjectDelete delete object from the store
func (c *Client) ObjectDelete(id []byte) error {
	return nil
}

// ObjectExist tests if an object with this id exists
func (c *Client) ObjectExist(id []byte) (bool, error) {
	return false, nil
}

// ReferenceUpdate updates reference list
func (c *Client) ReferenceUpdate(id []byte, refList []string) error {
	return nil
}

func newCommonObject(id, data []byte, refList []client.ReferenceID) *common.Object {
	obj := &common.Object{
		Key:   id,
		Value: data,
	}
	for _, ref := range refList {
		obj.ReferenceList = append(obj.ReferenceList, string(ref))
	}
	return obj
}
