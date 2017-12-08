package grpc

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/server"
	"github.com/zero-os/0-stor/server/api/grpc/rpctypes"
	pb "github.com/zero-os/0-stor/server/api/grpc/schema"
	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/db/badger"
	"github.com/zero-os/0-stor/server/encoding"
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

func TestSetObjectErrors(t *testing.T) {
	api, clean := getTestObjectAPI(require.New(t))
	defer clean()

	req := &pb.SetObjectRequest{}

	ctx := context.Background()
	_, err := api.SetObject(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilLabel, err)

	ctx = contextWithLabel(ctx, label)
	_, err = api.SetObject(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilKey, err)

	req.Key = []byte("myKey")
	_, err = api.SetObject(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilData, err)

	req.Data = []byte("someData")
	_, err = api.SetObject(ctx, req)
	require.NoError(t, err)
}

func TestSetObject(t *testing.T) {
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

	req := &pb.SetObjectRequest{
		Key:           []byte("testkey"),
		Data:          buf,
		ReferenceList: []string{"user1", "user2"},
	}

	_, err = api.SetObject(contextWithLabel(nil, label), req)
	require.NoError(err)

	// get data and validate it's correct
	objRawData, err := api.db.Get(db.DataKey([]byte(label), []byte("testkey")))
	require.NoError(err)
	require.NotNil(objRawData)
	obj, err := encoding.DecodeObject(objRawData)
	require.NoError(err)
	require.Equal(req.Data, obj.Data)

	// get reference list, and validate it's correct
	refListRawData, err := api.db.Get(db.ReferenceListKey([]byte(label), []byte("testkey")))
	require.NoError(err)
	require.NotNil(refListRawData)
	refList, err := encoding.DecodeReferenceList(refListRawData)
	require.NoError(err)
	require.Len(refList, len(req.ReferenceList))
	require.Subset(req.ReferenceList, refList)
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

	err = api.db.Set(db.ReferenceListKey([]byte(label), []byte("myKey")), []byte("someCorruptedRefList"))
	require.NoError(t, err)
	_, err = api.GetObject(ctx, req)
	requireGRPCError(t, rpctypes.ErrObjectRefListCorrupted, err)
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
		assert.Equal([]string{"user1", "user2"}, resp.GetReferenceList())
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

	// remove the reference list for half of them
	var index int
	keyMapping := make(map[string]struct{}, len(bufList))
	for key := range bufList {
		keyMapping[key] = struct{}{}
		index++
		if index%2 == 0 {
			continue
		}

		_, err := api.DeleteReferenceList(contextWithLabel(nil, label),
			&pb.DeleteReferenceListRequest{Key: []byte(key)})
		require.NoError(err)
	}

	req := pb.ListObjectKeysRequest{}

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
	key := []byte("testkey1")

	// create a key
	_, err := api.SetObject(ctx, &pb.SetObjectRequest{
		Key:           key,
		Data:          []byte{1, 2, 3, 4},
		ReferenceList: []string{"user1"},
	})
	require.NoError(err)

	t.Run("valid", func(t *testing.T) {
		req := &pb.DeleteObjectRequest{
			Key: key,
		}

		_, err := api.DeleteObject(ctx, req)
		require.NoError(err)

		reply, err := api.GetObjectStatus(ctx, &pb.GetObjectStatusRequest{
			Key: key,
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
			Key: key,
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

	err = api.db.Set(db.ReferenceListKey([]byte(label), req.Key), []byte("someCorruptedRefList"))
	require.NoError(t, err)
	resp, err = api.GetObjectStatus(ctx, req)
	require.NoError(t, err)
	require.Equal(t, pb.ObjectStatusCorrupted, resp.GetStatus())

	data, err = encoding.EncodeReferenceList(server.ReferenceList{"user1"})
	require.NoError(t, err)
	err = api.db.Set(db.ReferenceListKey([]byte(label), []byte("myKey")), data)
	require.NoError(t, err)
	resp, err = api.GetObjectStatus(ctx, req)
	require.NoError(t, err)
	require.Equal(t, pb.ObjectStatusOK, resp.GetStatus())
}

// we'll append one ref at a time, one 255 different goroutines at once,
// as to ensure that conflicts are resolved correctly
func TestAppendToReferenceListAsync(t *testing.T) {
	// first create our database and object
	require := require.New(t)

	api, clean := getTestObjectAPI(require)
	defer clean()

	ctx := contextWithLabel(nil, label)
	key := []byte("testkey1")
	value := []byte{1, 2, 3, 4}

	_, err := api.SetObject(ctx, &pb.SetObjectRequest{
		Key:  key,
		Data: value,
	})
	require.NoError(err)

	// now append our reference list
	group, ctx := errgroup.WithContext(ctx)
	var expectedList []string
	for i := 0; i < 256; i++ {
		userID := fmt.Sprintf("user%d", i)
		expectedList = append(expectedList, userID)
		group.Go(func() error {
			_, err := api.AppendToReferenceList(ctx, &pb.AppendToReferenceListRequest{
				Key:           key,
				ReferenceList: []string{userID},
			})
			return err
		})
	}
	require.NoError(group.Wait())

	// now ensure our ref list is idd correct, even though we don't know the order
	rawRefList, err := api.db.Get(db.ReferenceListKey([]byte(label), key))
	require.NoError(err)
	require.NotNil(rawRefList)
	refList, err := encoding.DecodeReferenceList(rawRefList)
	require.NoError(err)
	require.Len(refList, len(expectedList))
	require.Subset(expectedList, refList)
}

// we'll append one ref at a time, one 255 different goroutines at once,
// as to ensure that conflicts are resolved correctly
func TestDeleteFromReferenceListAsync(t *testing.T) {
	// first create our database and object
	require := require.New(t)

	api, clean := getTestObjectAPI(require)
	defer clean()

	ctx := contextWithLabel(nil, label)
	key := []byte("testkey1")
	value := []byte{1, 2, 3, 4}

	const refCount = 256

	var startRefList []string
	for i := 0; i < refCount; i++ {
		startRefList = append(startRefList, fmt.Sprintf("user%d", i))
	}

	_, err := api.SetObject(ctx, &pb.SetObjectRequest{
		Key:           key,
		Data:          value,
		ReferenceList: startRefList,
	})
	require.NoError(err)

	// ensure we have our ref list
	rawRefList, err := api.db.Get(db.ReferenceListKey([]byte(label), key))
	require.NoError(err)
	require.NotNil(rawRefList)
	refList, err := encoding.DecodeReferenceList(rawRefList)
	require.NoError(err)
	require.Len(refList, len(startRefList))
	require.Subset(startRefList, refList)

	// now remove from our reference list, one by one
	group, ctx := errgroup.WithContext(ctx)
	for i := 0; i < refCount; i++ {
		userID := fmt.Sprintf("user%d", i)
		group.Go(func() error {
			_, err := api.DeleteFromReferenceList(ctx, &pb.DeleteFromReferenceListRequest{
				Key:           key,
				ReferenceList: []string{userID},
			})
			return err
		})
	}
	require.NoError(group.Wait())

	// now ensure our ref list is now gone
	_, err = api.db.Get(db.ReferenceListKey([]byte(label), key))
	require.Equal(db.ErrNotFound, err)
}

func TestSetReferenceListErrors(t *testing.T) {
	api, clean := getTestObjectAPI(require.New(t))
	defer clean()

	req := &pb.SetReferenceListRequest{}

	ctx := context.Background()
	_, err := api.SetReferenceList(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilLabel, err)

	ctx = contextWithLabel(ctx, label)
	_, err = api.SetReferenceList(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilKey, err)

	req.Key = []byte("myKey")
	_, err = api.SetReferenceList(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilRefList, err)

	req.ReferenceList = []string{"user1"}
	_, err = api.SetReferenceList(ctx, req)
	require.NoError(t, err)
}

func TestGetReferenceListErrors(t *testing.T) {
	api, clean := getTestObjectAPI(require.New(t))
	defer clean()

	req := &pb.GetReferenceListRequest{}

	ctx := context.Background()
	_, err := api.GetReferenceList(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilLabel, err)

	ctx = contextWithLabel(ctx, label)
	_, err = api.GetReferenceList(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilKey, err)

	req.Key = []byte("myKey")
	_, err = api.GetReferenceList(ctx, req)
	requireGRPCError(t, rpctypes.ErrKeyNotFound, err)

	err = api.db.Set(db.ReferenceListKey([]byte(label), req.Key), []byte("someCorruptedRefList"))
	require.NoError(t, err)
	_, err = api.GetReferenceList(ctx, req)
	requireGRPCError(t, rpctypes.ErrObjectRefListCorrupted, err)

	data, err := encoding.EncodeReferenceList(server.ReferenceList{"user1"})
	require.NoError(t, err)
	err = api.db.Set(db.ReferenceListKey([]byte(label), req.Key), data)
	require.NoError(t, err)
	resp, err := api.GetReferenceList(ctx, req)
	require.NoError(t, err)
	require.Equal(t, []string{"user1"}, resp.GetReferenceList())
}

func TestGetReferenceCountErrors(t *testing.T) {
	api, clean := getTestObjectAPI(require.New(t))
	defer clean()

	req := &pb.GetReferenceCountRequest{}

	ctx := context.Background()
	_, err := api.GetReferenceCount(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilLabel, err)

	ctx = contextWithLabel(ctx, label)
	_, err = api.GetReferenceCount(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilKey, err)

	req.Key = []byte("myKey")
	resp, err := api.GetReferenceCount(ctx, req)
	require.NoError(t, err)
	require.Equal(t, int64(0), resp.GetCount())

	err = api.db.Set(db.ReferenceListKey([]byte(label), req.Key), []byte("someCorruptedRefList"))
	require.NoError(t, err)
	_, err = api.GetReferenceCount(ctx, req)
	requireGRPCError(t, rpctypes.ErrObjectRefListCorrupted, err)
}

func TestGetReferenceCount(t *testing.T) {
	api, clean := getTestObjectAPI(require.New(t))
	defer clean()

	req := &pb.GetReferenceCountRequest{Key: []byte("myKey")}
	ctx := contextWithLabel(nil, label)

	resp, err := api.GetReferenceCount(ctx, req)
	require.NoError(t, err)
	require.Equal(t, int64(0), resp.GetCount())

	data, err := encoding.EncodeReferenceList(server.ReferenceList{"user1"})
	require.NoError(t, err)
	err = api.db.Set(db.ReferenceListKey([]byte(label), req.Key), data)
	require.NoError(t, err)
	resp, err = api.GetReferenceCount(ctx, req)
	require.NoError(t, err)
	require.Equal(t, int64(1), resp.GetCount())

	data, err = encoding.EncodeReferenceList(server.ReferenceList{"user1", "user3", "user5"})
	require.NoError(t, err)
	err = api.db.Set(db.ReferenceListKey([]byte(label), req.Key), data)
	require.NoError(t, err)
	resp, err = api.GetReferenceCount(ctx, req)
	require.NoError(t, err)
	require.Equal(t, int64(3), resp.GetCount())
}

func TestAppendToReferenceListErrors(t *testing.T) {
	api, clean := getTestObjectAPI(require.New(t))
	defer clean()

	req := &pb.AppendToReferenceListRequest{}

	ctx := context.Background()
	_, err := api.AppendToReferenceList(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilLabel, err)

	ctx = contextWithLabel(ctx, label)
	_, err = api.AppendToReferenceList(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilKey, err)

	req.Key = []byte("myKey")
	_, err = api.AppendToReferenceList(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilRefList, err)

	err = api.db.Set(db.ReferenceListKey([]byte(label), req.Key), []byte("someCorruptedRefList"))
	require.NoError(t, err)
	req.ReferenceList = []string{"user1"}
	_, err = api.AppendToReferenceList(ctx, req)
	requireGRPCError(t, rpctypes.ErrObjectRefListCorrupted, err)

	err = api.db.Delete(db.ReferenceListKey([]byte(label), req.Key))
	require.NoError(t, err)
	_, err = api.AppendToReferenceList(ctx, req)
	require.NoError(t, err)
}

func TestDeleteFromReferenceListErrors(t *testing.T) {
	api, clean := getTestObjectAPI(require.New(t))
	defer clean()

	req := &pb.DeleteFromReferenceListRequest{}

	ctx := context.Background()
	_, err := api.DeleteFromReferenceList(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilLabel, err)

	ctx = contextWithLabel(ctx, label)
	_, err = api.DeleteFromReferenceList(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilKey, err)

	req.Key = []byte("myKey")
	_, err = api.DeleteFromReferenceList(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilRefList, err)

	err = api.db.Set(db.ReferenceListKey([]byte(label), req.Key), []byte("someCorruptedRefList"))
	require.NoError(t, err)
	req.ReferenceList = []string{"user1"}
	_, err = api.DeleteFromReferenceList(ctx, req)
	requireGRPCError(t, rpctypes.ErrObjectRefListCorrupted, err)

	err = api.db.Delete(db.ReferenceListKey([]byte(label), req.Key))
	require.NoError(t, err)
	resp, err := api.DeleteFromReferenceList(ctx, req)
	require.NoError(t, err)
	require.Equal(t, int64(0), resp.GetCount())
}

func TestDeleteReferenceListErrors(t *testing.T) {
	api, clean := getTestObjectAPI(require.New(t))
	defer clean()

	req := &pb.DeleteReferenceListRequest{}

	ctx := context.Background()
	_, err := api.DeleteReferenceList(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilLabel, err)

	ctx = contextWithLabel(ctx, label)
	_, err = api.DeleteReferenceList(ctx, req)
	requireGRPCError(t, rpctypes.ErrNilKey, err)

	req.Key = []byte("myKey")
	_, err = api.DeleteReferenceList(ctx, req)
	require.NoError(t, err)
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
