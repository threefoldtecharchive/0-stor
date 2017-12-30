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
	"crypto/rand"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/zero-os/0-stor/server"
	"github.com/zero-os/0-stor/server/api/grpc/rpctypes"
	pb "github.com/zero-os/0-stor/server/api/grpc/schema"
	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/db/badger"
	"github.com/zero-os/0-stor/server/encoding"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestNewObjectAPI(t *testing.T) {
	require.Panics(t, func() {
		NewObjectAPI(nil, 0)
	}, "no db given")
}

func requireGRPCError(t *testing.T, expectedErr, receivedErr error) {
	require := require.New(t)
	require.Error(receivedErr)
	require.NotEqual(expectedErr, receivedErr)
	receivedErr = rpctypes.Error(receivedErr)
	require.Equal(expectedErr, receivedErr)
}

func TestCreateObjectErrors(t *testing.T) {
	api, clean := getTestObjectAPI(require.New(t))
	defer clean()

	req := &pb.CreateObjectRequest{}

	ctx := context.Background()
	_, err := api.CreateObject(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilLabel, err)

	ctx = contextWithLabel(ctx, label)
	_, err = api.CreateObject(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilData, err)

	req.Data = []byte("someData")
	_, err = api.CreateObject(ctx, req)
	require.NoError(t, err)
}

func TestCreateObject(t *testing.T) {
	require := require.New(t)

	api, clean := getTestObjectAPI(require)
	defer clean()

	data, err := encoding.EncodeNamespace(server.Namespace{Label: []byte(label)})
	require.NoError(err)
	err = api.db.Set(db.NamespaceKey([]byte(label)), data)
	require.NoError(err)

	buf := make([]byte, 128)
	_, err = rand.Read(buf)
	require.NoError(err)

	req := &pb.CreateObjectRequest{
		Data: buf,
	}

	resp, err := api.CreateObject(contextWithLabel(nil, label), req)
	require.NoError(err)
	require.NotNil(resp)

	// get data and validate it's correct
	key := db.DataKey([]byte(label), resp.Key)
	objRawData, err := api.db.Get(key)
	require.NoErrorf(err, "key: %s", key)
	require.NotNil(objRawData)
	obj, err := encoding.DecodeObject(objRawData)
	require.NoError(err)
	require.Equal(req.Data, obj.Data)
}

func TestGetObjectErrors(t *testing.T) {
	api, clean := getTestObjectAPI(require.New(t))
	defer clean()

	req := &pb.GetObjectRequest{}

	ctx := context.Background()
	_, err := api.GetObject(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilLabel, err)

	ctx = contextWithLabel(ctx, label)
	_, err = api.GetObject(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilKey, err)

	req.Key = []byte("myKey")
	_, err = api.GetObject(ctx, req)
	requireGRPCError(t, rpctypes.ErrKeyNotFound, err)

	err = api.db.Set(db.DataKey([]byte(label), []byte("myKey")), []byte("someCorruptedData"))
	require.NoError(t, err)
	_, err = api.GetObject(ctx, req)
	requireGRPCError(t, rpctypes.ErrObjectDataCorrupted, err)

	data, err := encoding.EncodeObject(server.Object{Data: []byte("someData")})
	require.NoError(t, err)
	err = api.db.Set(db.DataKey([]byte(label), []byte("myKey")), data)
	require.NoError(t, err)
	_, err = api.GetObject(ctx, req)
	require.NoError(t, err)
}

func TestGetObject(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	api, clean := getTestObjectAPI(require)
	defer clean()

	bufList := populateDB(t, label, api.db)

	t.Run("valid", func(t *testing.T) {
		key := []byte("testkey0")
		req := &pb.GetObjectRequest{
			Key: key,
		}

		resp, err := api.GetObject(contextWithLabel(nil, label), req)
		require.NoError(err)

		assert.Equal(bufList["testkey0"], resp.GetData())
	})

	t.Run("non existing", func(t *testing.T) {
		req := &pb.GetObjectRequest{
			Key: []byte("notexistingkey"),
		}

		_, err := api.GetObject(contextWithLabel(nil, label), req)
		require.Error(err)
		err = rpctypes.Error(err)
		assert.Equal(rpctypes.ErrKeyNotFound, err)
	})
}

func TestListObjectkeys(t *testing.T) {
	require := require.New(t)

	api, clean := getTestObjectAPI(require)
	defer clean()

	bufList := populateDB(t, label, api.db)
	require.NotEmpty(bufList)

	req := pb.ListObjectKeysRequest{}

	keyMapping := make(map[string]struct{}, len(bufList))
	for key := range bufList {
		keyMapping[key] = struct{}{}
	}

	stream := listServerStream{
		ServerStream: nil,
		keyMapping:   keyMapping,
	}

	err := api.ListObjectKeys(&req, &stream)
	requireGRPCError(t, rpctypes.ErrNilLabel, err)

	stream.label = label
	err = api.ListObjectKeys(&req, &stream)
	require.NoError(err)
}

type listServerStream struct {
	grpc.ServerStream

	label      string
	keyMapping map[string]struct{}
}

func (stream *listServerStream) Send(resp *pb.ListObjectKeysResponse) error {
	if resp == nil {
		return errors.New("no object given")
	}

	key := resp.GetKey()
	if key == nil {
		return errors.New("no key given")
	}
	_, ok := stream.keyMapping[string(key)]
	if !ok {
		return fmt.Errorf(
			"key %q was not found in expected mapping",
			key)
	}

	return nil
}

func (stream *listServerStream) Context() context.Context {
	if stream.label == "" {
		return context.Background()
	}
	return contextWithLabel(nil, stream.label)
}

func TestDeleteObjectErrors(t *testing.T) {
	api, clean := getTestObjectAPI(require.New(t))
	defer clean()

	req := &pb.DeleteObjectRequest{}

	ctx := context.Background()
	_, err := api.DeleteObject(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilLabel, err)

	ctx = contextWithLabel(ctx, label)
	_, err = api.DeleteObject(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilKey, err)

	req.Key = []byte("myKey")
	_, err = api.DeleteObject(ctx, req)
	require.NoError(t, err)
}

func TestDeleteObject(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	api, clean := getTestObjectAPI(require)
	defer clean()

	ctx := contextWithLabel(nil, label)

	// create a key
	resp, err := api.CreateObject(ctx, &pb.CreateObjectRequest{
		Data: []byte{1, 2, 3, 4},
	})
	require.NoError(err)
	require.NotNil(resp)

	t.Run("valid", func(t *testing.T) {
		req := &pb.DeleteObjectRequest{
			Key: resp.Key,
		}

		_, err := api.DeleteObject(ctx, req)
		require.NoError(err)

		reply, err := api.GetObjectStatus(ctx, &pb.GetObjectStatusRequest{
			Key: resp.Key,
		})
		require.NoError(err)
		assert.Equal(pb.ObjectStatusMissing, reply.GetStatus())
	})

	// deleting a non existing object doesn't return an error.
	t.Run("non exists", func(t *testing.T) {
		req := &pb.DeleteObjectRequest{
			Key: []byte("nonexists"),
		}

		_, err := api.DeleteObject(ctx, req)
		require.NoError(err)

		reply, err := api.GetObjectStatus(ctx, &pb.GetObjectStatusRequest{
			Key: []byte("nonexists"),
		})
		require.NoError(err)
		assert.Equal(pb.ObjectStatusMissing, reply.GetStatus())
	})
}

func TestGetObjectStatusErrors(t *testing.T) {
	api, clean := getTestObjectAPI(require.New(t))
	defer clean()

	req := &pb.GetObjectStatusRequest{}

	ctx := context.Background()
	_, err := api.GetObjectStatus(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilLabel, err)

	ctx = contextWithLabel(ctx, label)
	_, err = api.GetObjectStatus(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilKey, err)

	req.Key = []byte("myKey")
	resp, err := api.GetObjectStatus(ctx, req)
	require.NoError(t, err)
	require.Equal(t, pb.ObjectStatusMissing, resp.GetStatus())
}

func TestGetObjectStatus(t *testing.T) {
	api, clean := getTestObjectAPI(require.New(t))
	defer clean()

	req := &pb.GetObjectStatusRequest{Key: []byte("myKey")}
	ctx := contextWithLabel(nil, label)

	resp, err := api.GetObjectStatus(ctx, req)
	require.NoError(t, err)
	require.Equal(t, pb.ObjectStatusMissing, resp.GetStatus())

	err = api.db.Set(db.DataKey([]byte(label), req.Key), []byte("someCorruptedData"))
	require.NoError(t, err)
	resp, err = api.GetObjectStatus(ctx, req)
	require.NoError(t, err)
	require.Equal(t, pb.ObjectStatusCorrupted, resp.GetStatus())

	data, err := encoding.EncodeObject(server.Object{Data: []byte("someData")})
	require.NoError(t, err)
	err = api.db.Set(db.DataKey([]byte(label), []byte("myKey")), data)
	require.NoError(t, err)
	resp, err = api.GetObjectStatus(ctx, req)
	require.NoError(t, err)
	require.Equal(t, pb.ObjectStatusOK, resp.GetStatus())
}

func TestConvertStatus(t *testing.T) {
	require := require.New(t)

	// valid responses
	require.Equal(pb.ObjectStatusOK, convertStatus(server.ObjectStatusOK))
	require.Equal(pb.ObjectStatusMissing, convertStatus(server.ObjectStatusMissing))
	require.Equal(pb.ObjectStatusCorrupted, convertStatus(server.ObjectStatusCorrupted))

	// all other responses should panic
	for i := 3; i < 256; i++ {
		require.Panics(func() {
			convertStatus(server.ObjectStatus(i))
		})
	}
}

func getTestObjectAPI(require *require.Assertions) (*ObjectAPI, func()) {
	tmpDir, err := ioutil.TempDir("", "0stortest")
	require.NoError(err)

	db, err := badger.New(path.Join(tmpDir, "data"), path.Join(tmpDir, "meta"))
	if err != nil {
		require.NoError(err)
	}

	clean := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return NewObjectAPI(db, 0), clean
}

func contextWithLabel(ctx context.Context, label string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	md := metadata.Pairs(rpctypes.MetaLabelKey, label)
	return metadata.NewIncomingContext(ctx, md)
}
