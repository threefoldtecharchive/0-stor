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

package grpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/zero-os/0-stor/client/datastor"
	"github.com/zero-os/0-stor/server/api/grpc/rpctypes"
	pb "github.com/zero-os/0-stor/server/api/grpc/schema"

	log "github.com/Sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

var _ (datastor.Client) = (*Client)(nil)

// Client defines a data client,
// to connect to a zstordb using the GRPC interface.
type Client struct {
	conn             *grpc.ClientConn
	objService       pb.ObjectManagerClient
	namespaceService pb.NamespaceManagerClient

	contextConstructor func(context.Context) (context.Context, error)

	label string
}

// NewClient create a new data client,
// which allows you to connect to a zstordb using a GRPC interface.
// The address to the zstordb server is required,
// and so is the label, as the latter serves as the identifier of the to be used namespace.
// The jwtToken is required, only if the connected zstordb server requires this.
func NewClient(addr, namespace string, jwtTokenGetter datastor.JWTTokenGetter) (*Client, error) {
	if len(addr) == 0 {
		return nil, errors.New("no/empty zstordb address given")
	}
	if len(namespace) == 0 {
		return nil, errors.New("no/empty namespace given")
	}

	// ensure that we have a valid connection
	conn, err := grpc.Dial(addr,
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(math.MaxInt32),
			grpc.MaxCallSendMsgSize(math.MaxInt32),
		))
	if err != nil {
		return nil, fmt.Errorf(
			"couldn't connect to zstordb server %s: %v",
			addr, err)
	}

	client := &Client{
		conn:             conn,
		objService:       pb.NewObjectManagerClient(conn),
		namespaceService: pb.NewNamespaceManagerClient(conn),
	}

	if jwtTokenGetter == nil {
		client.contextConstructor = defaultContextConstructor(namespace)
		client.label = namespace
		return client, nil
	}

	label, err := jwtTokenGetter.GetLabel(namespace)
	if err != nil {
		return nil, err
	}
	client.contextConstructor = func(ctx context.Context) (context.Context, error) {
		jwtToken, err := jwtTokenGetter.GetJWTToken(namespace)
		if err != nil {
			return nil, err
		}
		if ctx == nil {
			ctx = context.Background()
		}
		md := metadata.Pairs(
			rpctypes.MetaAuthKey, jwtToken,
			rpctypes.MetaLabelKey, label)
		return metadata.NewOutgoingContext(ctx, md), nil
	}
	client.label = label
	return client, nil
}

// CreateObject implements datastor.Client.CreateObject
func (c *Client) CreateObject(data []byte) (key []byte, err error) {
	ctx, err := c.contextConstructor(nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.objService.CreateObject(ctx, &pb.CreateObjectRequest{Data: data})
	if err != nil {
		return nil, toErr(err)
	}
	return resp.Key, nil
}

// GetObject implements datastor.Client.GetObject
func (c *Client) GetObject(key []byte) (*datastor.Object, error) {
	ctx, err := c.contextConstructor(nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.objService.GetObject(ctx, &pb.GetObjectRequest{Key: key})
	if err != nil {
		return nil, toErr(err)
	}

	dataObject := &datastor.Object{
		Key:  key,
		Data: resp.GetData(),
	}
	if len(dataObject.Data) == 0 {
		return nil, datastor.ErrMissingData
	}
	return dataObject, nil
}

// DeleteObject implements datastor.Client.DeleteObject
func (c *Client) DeleteObject(key []byte) error {
	ctx, err := c.contextConstructor(nil)
	if err != nil {
		return err
	}
	// delete the objects from the server
	_, err = c.objService.DeleteObject(ctx, &pb.DeleteObjectRequest{Key: key})
	if err != nil {
		return toErr(err)
	}
	return err
}

// GetObjectStatus implements datastor.Client.GetObjectStatus
func (c *Client) GetObjectStatus(key []byte) (datastor.ObjectStatus, error) {
	ctx, err := c.contextConstructor(nil)
	if err != nil {
		return datastor.ObjectStatus(0), err
	}
	resp, err := c.objService.GetObjectStatus(ctx, &pb.GetObjectStatusRequest{Key: key})
	if err != nil {
		return datastor.ObjectStatus(0), toErr(err)
	}
	return convertStatus(resp.GetStatus())
}

// ExistObject implements datastor.Client.ExistObject
func (c *Client) ExistObject(key []byte) (bool, error) {
	status, err := c.GetObjectStatus(key)
	if err != nil {
		return false, err
	}
	switch status {
	case datastor.ObjectStatusOK:
		return true, nil
	case datastor.ObjectStatusCorrupted:
		return false, datastor.ErrObjectCorrupted
	default:
		return false, nil
	}
}

// GetNamespace implements datastor.Client.GetNamespace
func (c *Client) GetNamespace() (*datastor.Namespace, error) {
	ctx, err := c.contextConstructor(nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.namespaceService.GetNamespace(ctx, &pb.GetNamespaceRequest{})
	if err != nil {
		return nil, toErr(err)
	}

	ns := &datastor.Namespace{Label: resp.GetLabel()}
	if ns.Label != c.label {
		return nil, datastor.ErrInvalidLabel
	}

	ns.ReadRequestPerHour = resp.GetReadRequestPerHour()
	ns.WriteRequestPerHour = resp.GetWriteRequestPerHour()
	ns.NrObjects = resp.GetNrObjects()
	return ns, nil
}

// ListObjectKeyIterator implements datastor.Client.ListObjectKeyIterator
func (c *Client) ListObjectKeyIterator(ctx context.Context) (<-chan datastor.ObjectKeyResult, error) {
	// ensure a context is given
	if ctx == nil {
		panic("no context given")
	}

	group, ctx := errgroup.WithContext(ctx)
	grpcCtx, err := c.contextConstructor(ctx)
	if err != nil {
		return nil, err
	}

	// create stream
	stream, err := c.objService.ListObjectKeys(grpcCtx, &pb.ListObjectKeysRequest{})
	if err != nil {
		return nil, toContextErr(grpcCtx, err)
	}

	// create output channel and start fetching from the stream
	ch := make(chan datastor.ObjectKeyResult, 1)
	group.Go(func() error {
		// fetch all objects possible
		var (
			input *pb.ListObjectKeysResponse
		)
		for {
			select {
			case <-ctx.Done():
				return nil
			default:
			}

			// receive the next object, and check error as a first task to do
			input, err = stream.Recv()
			if err != nil {
				if err == io.EOF {
					// stream is done
					return nil
				}
				err = toContextErr(ctx, err)

				// an unexpected error has happened, exit with an error
				log.Errorf(
					"an error was received while receiving the key of an object for: %v",
					err)
				select {
				case ch <- datastor.ObjectKeyResult{Error: err}:
				case <-ctx.Done():
				}
				return err
			}

			// create the error/valid data result
			result := datastor.ObjectKeyResult{Key: input.GetKey()}
			if len(result.Key) == 0 {
				result.Error = datastor.ErrMissingKey
			}

			// return the result for the given key
			select {
			case ch <- result:
			case <-ctx.Done():
				return result.Error
			}
			if result.Error != nil {
				// if the result was an error, return also the error
				return result.Error
			}
		}
	})

	// launch the err group routine,
	// to close the output ch
	go func() {
		defer close(ch)
		err := group.Wait()
		if err != nil {
			log.Errorf(
				"ExistObjectIterator job group has exited with an error: %v",
				err)
		}
	}()

	return ch, nil
}

// Close implements datastor.Client.Close
func (c *Client) Close() error {
	return c.conn.Close()
}

func defaultContextConstructor(namespace string) func(ctx context.Context) (context.Context, error) {
	return func(ctx context.Context) (context.Context, error) {
		if ctx == nil {
			ctx = context.Background()
		}
		md := metadata.Pairs(rpctypes.MetaLabelKey, namespace)
		return metadata.NewOutgoingContext(ctx, md), nil
	}
}

func toErr(err error) error {
	err = rpctypes.Error(err)
	if _, ok := err.(rpctypes.ZStorError); ok {
		if err, ok := expectedErrorMapping[err]; ok {
			return err
		}
		return err
	}
	return err
}

var expectedErrorMapping = map[error]error{
	rpctypes.ErrKeyNotFound:         datastor.ErrKeyNotFound,
	rpctypes.ErrObjectDataCorrupted: datastor.ErrObjectCorrupted,
	rpctypes.ErrPermissionDenied:    datastor.ErrPermissionDenied,
}

func toContextErr(ctx context.Context, err error) error {
	if err := toErr(err); err != nil {
		return err
	}
	code := grpc.Code(err)
	switch code {
	case codes.DeadlineExceeded, codes.Canceled:
		if ctx.Err() != nil {
			err = ctx.Err()
		}
	case codes.FailedPrecondition:
		err = grpc.ErrClientConnClosing
	}
	return err
}

// convertStatus converts pb.ObjectStatus datastor.ObjectStatus
func convertStatus(status pb.ObjectStatus) (datastor.ObjectStatus, error) {
	s, ok := _ProtoObjectStatusMapping[status]
	if !ok {
		log.Debugf("invalid (proto) object status %d received", status)
		return datastor.ObjectStatus(0), datastor.ErrInvalidStatus
	}
	return s, nil
}

var _ProtoObjectStatusMapping = map[pb.ObjectStatus]datastor.ObjectStatus{
	pb.ObjectStatusOK:        datastor.ObjectStatusOK,
	pb.ObjectStatusMissing:   datastor.ObjectStatusMissing,
	pb.ObjectStatusCorrupted: datastor.ObjectStatusCorrupted,
}
