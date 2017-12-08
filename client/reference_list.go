package client

import (
	"fmt"
	"sync"

	"github.com/zero-os/0-stor/client/metastor"
)

// SetReferenceList replace the complete reference list for the object pointed by key
func (c *Client) SetReferenceList(key []byte, refList []string) error {
	md, err := c.metaCli.GetMetadata(key)
	if err != nil {
		return err
	}
	return c.SetReferenceListWithMeta(md, refList)
}

// SetReferenceListWithMeta is the same as SetReferenceList but take metadata instead of key
// as argument
func (c *Client) SetReferenceListWithMeta(md *metastor.Data, refList []string) error {
	return c.updateRefListWithMeta(md, refList, refListOpSet)
}

// AppendToReferenceList appends some reference to the (non-)existing reference list
// of the object pointed by key.
func (c *Client) AppendToReferenceList(key []byte, refList []string) error {
	md, err := c.metaCli.GetMetadata(key)
	if err != nil {
		return err
	}
	return c.AppendToReferenceListWithMeta(md, refList)
}

// AppendToReferenceListWithMeta is the same as AppendToReferenceList
// but take metadata instead of key as argument
func (c *Client) AppendToReferenceListWithMeta(md *metastor.Data, refList []string) error {
	return c.updateRefListWithMeta(md, refList, refListOpAppendTo)
}

// DeleteFromReferenceList deletes some reference from the (non-)existing
// reference list of the object pointed by key.
// It wont return error in case of the object doesn't have some elements of the `refList`.
func (c *Client) DeleteFromReferenceList(key []byte, refList []string) error {
	md, err := c.metaCli.GetMetadata(key)
	if err != nil {
		return err
	}
	return c.DeleteFromReferenceListWithMeta(md, refList)
}

// DeleteFromReferenceListWithMeta is the same as DeleteFromReferenceList but take metadata
// instead of key as argument
func (c *Client) DeleteFromReferenceListWithMeta(md *metastor.Data, refList []string) error {
	return c.updateRefListWithMeta(md, refList, refListOpDeleteFrom)
}

// DeleteReferenceList deletes the (non-)existing
// reference list of the object pointed by key.
func (c *Client) DeleteReferenceList(key []byte) error {
	md, err := c.metaCli.GetMetadata(key)
	if err != nil {
		return err
	}
	return c.DeleteReferenceListWithMeta(md)
}

// DeleteReferenceListWithMeta is the same as DeleteReferenceList but take metadata
// instead of key as argument
func (c *Client) DeleteReferenceListWithMeta(md *metastor.Data) error {
	return c.updateRefListWithMeta(md, nil, refListOpDelete)
}

// TODO: support GetReferenceList?!

func (c *Client) updateRefListWithMeta(md *metastor.Data, refList []string, op int) error {
	for _, chunk := range md.Chunks {

		var (
			wg    sync.WaitGroup
			errCh = make(chan error, len(chunk.Shards))
		)

		wg.Add(len(chunk.Shards))
		for _, shard := range chunk.Shards {
			go func(shard string) {
				defer wg.Done()

				// get stor client
				storCli, err := c.getStor(shard)
				if err != nil {
					errCh <- err
					return
				}

				// do the work
				switch op {
				case refListOpSet:
					err = storCli.SetReferenceList(chunk.Key, refList)
				case refListOpAppendTo:
					err = storCli.AppendToReferenceList(chunk.Key, refList)
				case refListOpDeleteFrom:
					// TODO: return the count value to the user?!
					_, err = storCli.DeleteFromReferenceList(chunk.Key, refList)
				case refListOpDelete:
					err = storCli.DeleteReferenceList(chunk.Key)
				default:
					err = fmt.Errorf("wrong operation: %v", op)
				}
				if err != nil {
					errCh <- err
					return
				}
			}(shard)
		}

		wg.Wait()

		if len(errCh) > 0 {
			err := <-errCh
			return err
		}
	}
	return nil
}

const (
	_ = iota
	refListOpSet
	refListOpAppendTo
	refListOpDeleteFrom
	refListOpDelete
)
