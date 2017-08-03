package grpc

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"golang.org/x/net/context"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pb "github.com/zero-os/0-stor/grpc_store"
	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/db/badger"
	"github.com/zero-os/0-stor/server/manager"
	"github.com/zero-os/0-stor/server/test"
)

// We can't test the List method in here since it uses stream.
// the test for the list method is located in the integration test at "github.com/zero-os/0-stor/server/test/grpc"

func getTestObjectAPI(t *testing.T) (*ObjectAPI, func()) {
	tmpDir, err := ioutil.TempDir("", "0stortest")
	require.NoError(t, err)

	db, err := badger.New(path.Join(tmpDir, "data"), path.Join(tmpDir, "meta"))
	if err != nil {
		require.NoError(t, err)
	}

	clean := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return NewObjectAPI(db), clean
}

func TestCreateObject(t *testing.T) {
	api, clean := getTestObjectAPI(t)
	defer clean()

	label := "testnamespace"

	nsMgr := manager.NewNamespaceManager(api.db)
	objMgr := manager.NewObjectManager(label, api.db)
	err := nsMgr.Create(label)
	require.NoError(t, err)

	buf := make([]byte, 1024*1024)
	_, err = rand.Read(buf)
	require.NoError(t, err)

	req := &pb.CreateObjectRequest{
		Label: label,
		Object: &pb.Object{
			Key:           "testkey",
			Value:         buf,
			ReferenceList: []string{"user1", "user2"},
		},
	}

	_, err = api.Create(context.Background(), req)
	require.NoError(t, err)

	obj, err := objMgr.Get([]byte("testkey"))
	require.NoError(t, err)
	assert.Equal(t, buf, obj.Data)
	assert.EqualValues(t, []byte("user1"), bytes.Trim(obj.ReferenceList[0][:], "\x00"))
	assert.EqualValues(t, []byte("user2"), bytes.Trim(obj.ReferenceList[1][:], "\x00"))
	assert.EqualValues(t, []byte(nil), bytes.Trim(obj.ReferenceList[2][:], "\x00"))
}

func TestGetObject(t *testing.T) {
	api, clean := getTestObjectAPI(t)
	defer clean()

	label, bufList := test.PopulateDB(t, api.db)

	t.Run("valid", func(t *testing.T) {

		req := &pb.GetObjectRequest{
			Label: label,
			Key:   "testkey0",
		}

		resp, err := api.Get(context.Background(), req)
		require.NoError(t, err)

		obj := resp.GetObject()

		assert.Equal(t, "testkey0", obj.GetKey())
		assert.Equal(t, bufList[0], obj.GetValue())
		assert.Equal(t, []string{"user1", "user2"}, obj.GetReferenceList())
	})

	t.Run("non existing", func(t *testing.T) {
		req := &pb.GetObjectRequest{
			Label: label,
			Key:   "notexistingkey",
		}

		_, err := api.Get(context.Background(), req)
		assert.Equal(t, db.ErrNotFound, err)
	})
}

func TestExistsObject(t *testing.T) {
	api, clean := getTestObjectAPI(t)
	defer clean()

	label, bufList := test.PopulateDB(t, api.db)

	for i := 0; i < len(bufList); i++ {
		key := fmt.Sprintf("testkey%d", i)
		t.Run(key, func(t *testing.T) {
			req := &pb.ExistsObjectRequest{
				Label: label,
				Key:   key,
			}

			resp, err := api.Exists(context.Background(), req)
			require.NoError(t, err)
			assert.True(t, resp.Exists, fmt.Sprintf("Key %s should exists", key))
		})
	}

	t.Run("non exists", func(t *testing.T) {
		req := &pb.ExistsObjectRequest{
			Label: label,
			Key:   "nonexists",
		}

		resp, err := api.Exists(context.Background(), req)
		require.NoError(t, err)
		assert.False(t, resp.Exists, fmt.Sprint("Key nonexists should not exists"))
	})
}

func TestDeleteObject(t *testing.T) {
	api, clean := getTestObjectAPI(t)
	defer clean()

	label, _ := test.PopulateDB(t, api.db)
	objMgr := manager.NewObjectManager(label, api.db)

	t.Run("valid", func(t *testing.T) {
		req := &pb.DeleteObjectRequest{
			Label: label,
			Key:   "testkey1",
		}

		_, err := api.Delete(context.Background(), req)
		require.NoError(t, err)

		exists, err := objMgr.Exists([]byte(req.Key))
		require.NoError(t, err)
		assert.False(t, exists)
	})

	// deleting a non existing object doesn't return an error.
	t.Run("non exists", func(t *testing.T) {
		req := &pb.DeleteObjectRequest{
			Label: label,
			Key:   "nonexists",
		}

		_, err := api.Delete(context.Background(), req)
		require.NoError(t, err)

		exists, err := objMgr.Exists([]byte(req.Key))
		require.NoError(t, err)
		assert.False(t, exists)
	})
}
