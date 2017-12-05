package grpc

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"google.golang.org/grpc"

	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/server"
	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/db/badger"
	"github.com/zero-os/0-stor/server/encoding"
	pb "github.com/zero-os/0-stor/server/schema"
)

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
		Label: label,
		Object: &pb.Object{
			Key:           []byte("testkey"),
			Value:         buf,
			ReferenceList: []string{"user1", "user2"},
		},
	}

	_, err = api.Create(context.Background(), req)
	require.NoError(err)

	// get data and validate it's correct
	objRawData, err := api.db.Get(db.DataKey([]byte(label), []byte("testkey")))
	require.NoError(err)
	require.NotNil(objRawData)
	obj, err := encoding.DecodeObject(objRawData)
	require.NoError(err)
	require.Equal(req.Object.Value, obj.Data)

	// get reference list, and validate it's correct
	refListRawData, err := api.db.Get(db.ReferenceListKey([]byte(label), []byte("testkey")))
	require.NoError(err)
	require.NotNil(refListRawData)
	refList, err := encoding.DecodeReferenceList(refListRawData)
	require.NoError(err)
	require.Len(refList, len(req.Object.ReferenceList))
	require.Subset(req.Object.ReferenceList, refList)
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
			Label: label,
			Key:   key,
		}

		resp, err := api.Get(context.Background(), req)
		require.NoError(err)

		obj := resp.GetObject()

		assert.Equal(key, obj.GetKey())
		assert.Equal(bufList["testkey0"], obj.GetValue())
		assert.Equal([]string{"user1", "user2"}, obj.GetReferenceList())
	})

	t.Run("non existing", func(t *testing.T) {
		req := &pb.GetObjectRequest{
			Label: label,
			Key:   []byte("notexistingkey"),
		}

		_, err := api.Get(context.Background(), req)
		assert.Equal(db.ErrNotFound, err)
	})
}

func TestExistsObject(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	api, clean := getTestObjectAPI(require)
	defer clean()

	bufList := populateDB(t, label, api.db)

	for i := 0; i < len(bufList); i++ {
		key := fmt.Sprintf("testkey%d", i)
		t.Run(key, func(t *testing.T) {
			req := &pb.ExistsObjectRequest{
				Label: label,
				Key:   []byte(key),
			}

			resp, err := api.Exists(context.Background(), req)
			require.NoError(err)
			assert.True(resp.Exists, fmt.Sprintf("Key %s should exists", key))
		})
	}

	t.Run("non exists", func(t *testing.T) {
		req := &pb.ExistsObjectRequest{
			Label: label,
			Key:   []byte("nonexists"),
		}

		resp, err := api.Exists(context.Background(), req)
		require.NoError(err)
		assert.False(resp.Exists, fmt.Sprint("Key nonexists should not exists"))
	})
}

func TestListObjects(t *testing.T) {
	require := require.New(t)

	api, clean := getTestObjectAPI(require)
	defer clean()

	bufList := populateDB(t, label, api.db)
	require.NotEmpty(bufList)

	// remove the reference list for half of them
	ctx := context.Background()
	var index int
	refList := make(map[string]struct{}, len(bufList))
	for key := range bufList {
		index++
		if index%2 == 0 {
			refList[key] = struct{}{}
			continue
		}

		_, err := api.RemoveReferenceList(ctx, &pb.UpdateReferenceListRequest{
			Label:         label,
			Key:           []byte(key),
			ReferenceList: []string{"user1", "user2"},
		})
		require.NoError(err)
	}

	req := pb.ListObjectsRequest{Label: label}
	stream := listServerStream{
		ServerStream: nil,
		label:        label,
		mapping:      bufList,
		refMapping:   refList,
	}
	require.NoError(api.List(&req, &stream))
}

type listServerStream struct {
	grpc.ServerStream

	label      string
	mapping    map[string][]byte
	refMapping map[string]struct{}
}

func (stream *listServerStream) Send(obj *pb.Object) error {
	if obj == nil {
		return errors.New("no object given")
	}

	key := obj.GetKey()
	if key == nil {
		return errors.New("no key given")
	}
	value, ok := stream.mapping[string(key)]
	if !ok {
		return fmt.Errorf(
			"key %q was not found in expected mapping",
			key)
	}

	objValue := obj.GetValue()
	if bytes.Compare(value, objValue) != 0 {
		return fmt.Errorf("value %q was expected to be %v, but was %v",
			key, value, objValue)
	}

	if _, ok := stream.refMapping[string(key)]; ok {
		refList := obj.GetReferenceList()
		if len(refList) != 2 || refList[0] != "user1" || refList[1] != "user2" {
			return fmt.Errorf("key %q has an invalid reference list: %v", key, refList)
		}
	} else {
		refList := obj.GetReferenceList()
		if len(refList) != 0 {
			return fmt.Errorf("key %q has an unexpected reference list: %v", key, refList)
		}
	}

	return nil
}

func (stream *listServerStream) Context() context.Context {
	return context.Background()
}

func TestDeleteObject(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	api, clean := getTestObjectAPI(require)
	defer clean()

	ctx := context.Background()
	key := []byte("testkey1")

	// create a key
	_, err := api.Create(ctx, &pb.CreateObjectRequest{
		Label: label,
		Object: &pb.Object{
			Key:           key,
			Value:         []byte{1, 2, 3, 4},
			ReferenceList: []string{"user1"},
		},
	})
	require.NoError(err)

	t.Run("valid", func(t *testing.T) {
		req := &pb.DeleteObjectRequest{
			Label: label,
			Key:   key,
		}

		_, err := api.Delete(ctx, req)
		require.NoError(err)

		existsReply, err := api.Exists(ctx, &pb.ExistsObjectRequest{
			Label: label,
			Key:   key,
		})
		require.NoError(err)
		assert.False(existsReply.Exists)
	})

	// deleting a non existing object doesn't return an error.
	t.Run("non exists", func(t *testing.T) {
		req := &pb.DeleteObjectRequest{
			Label: label,
			Key:   []byte("nonexists"),
		}

		_, err := api.Delete(context.Background(), req)
		require.NoError(err)

		existsReply, err := api.Exists(ctx, &pb.ExistsObjectRequest{
			Label: label,
			Key:   req.Key,
		})
		require.NoError(err)
		assert.False(existsReply.Exists)
	})
}

// we'll append one ref at a time, one 255 different goroutines at once,
// as to ensure that conflicts are resolved correctly
func TestAppendReferenceListAsync(t *testing.T) {
	// first create our database and object
	require := require.New(t)

	api, clean := getTestObjectAPI(require)
	defer clean()

	ctx := context.Background()
	key := []byte("testkey1")
	value := []byte{1, 2, 3, 4}

	_, err := api.Create(ctx, &pb.CreateObjectRequest{
		Label: label,
		Object: &pb.Object{
			Key:   key,
			Value: value,
		},
	})
	require.NoError(err)

	// now append our reference list
	group, ctx := errgroup.WithContext(ctx)
	var expectedList []string
	for i := 0; i < 256; i++ {
		userID := fmt.Sprintf("user%d", i)
		expectedList = append(expectedList, userID)
		group.Go(func() error {
			_, err := api.AppendReferenceList(ctx, &pb.UpdateReferenceListRequest{
				Label:         label,
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
func TestRemoveReferenceListAsync(t *testing.T) {
	// first create our database and object
	require := require.New(t)

	api, clean := getTestObjectAPI(require)
	defer clean()

	ctx := context.Background()
	key := []byte("testkey1")
	value := []byte{1, 2, 3, 4}

	const refCount = 256

	var startRefList []string
	for i := 0; i < refCount; i++ {
		startRefList = append(startRefList, fmt.Sprintf("user%d", i))
	}

	_, err := api.Create(ctx, &pb.CreateObjectRequest{
		Label: label,
		Object: &pb.Object{
			Key:           key,
			Value:         value,
			ReferenceList: startRefList,
		},
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
			_, err := api.RemoveReferenceList(ctx, &pb.UpdateReferenceListRequest{
				Label:         label,
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

func TestConvertStatus(t *testing.T) {
	require := require.New(t)

	// valid responses
	require.Equal(pb.CheckResponse_ok, convertStatus(server.CheckStatusOK))
	require.Equal(pb.CheckResponse_missing, convertStatus(server.CheckStatusMissing))
	require.Equal(pb.CheckResponse_corrupted, convertStatus(server.CheckStatusCorrupted))

	// all other responses should panic
	for i := 3; i < 256; i++ {
		require.Panics(func() {
			convertStatus(server.CheckStatus(i))
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
