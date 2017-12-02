package server

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"golang.org/x/net/context"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/db/badger"
	"github.com/zero-os/0-stor/server/manager"
	pb "github.com/zero-os/0-stor/server/schema"
)

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

	return NewObjectAPI(db), clean
}

func populateDB(require *require.Assertions, db db.DB) (string, [][]byte) {
	nsMgr := manager.NewNamespaceManager(db)
	objMgr := manager.NewObjectManager(label, db)
	err := nsMgr.Create(label)
	require.NoError(err)

	bufList := make([][]byte, 10)

	for i := 0; i < 10; i++ {
		bufList[i] = make([]byte, 1024*1024)
		_, err = rand.Read(bufList[i])
		require.NoError(err)

		refList := []string{
			"user1", "user2",
		}
		key := fmt.Sprintf("testkey%d", i)

		err = objMgr.Set([]byte(key), bufList[i], refList)
		require.NoError(err)
	}

	return label, bufList
}

func TestCreateObject(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	api, clean := getTestObjectAPI(require)
	defer clean()

	nsMgr := manager.NewNamespaceManager(api.db)
	objMgr := manager.NewObjectManager(label, api.db)
	err := nsMgr.Create(label)
	require.NoError(err)

	buf := make([]byte, 1024*1024)
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

	obj, err := objMgr.Get([]byte("testkey"))
	require.NoError(err, "fail to get the object from db")

	strRefList, err := obj.GetreferenceListStr()
	require.NoError(err, "fail to get string reference list")

	data, err := obj.Data()
	require.NoError(err, "fail to get data")

	assert.Equal(buf, data, "data is not the same")
	require.Equal(2, len(strRefList), "reference list not correct size")
	assert.EqualValues("user1", strRefList[0])
	assert.EqualValues("user2", strRefList[1])
}

func TestGetObject(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	api, clean := getTestObjectAPI(require)
	defer clean()

	label, bufList := populateDB(require, api.db)

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
		assert.Equal(bufList[0], obj.GetValue())
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

	label, bufList := populateDB(require, api.db)

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

func TestDeleteObject(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	api, clean := getTestObjectAPI(require)
	defer clean()

	label, _ := populateDB(require, api.db)
	objMgr := manager.NewObjectManager(label, api.db)

	t.Run("valid", func(t *testing.T) {
		req := &pb.DeleteObjectRequest{
			Label: label,
			Key:   []byte("testkey1"),
		}

		_, err := api.Delete(context.Background(), req)
		require.NoError(err)

		exists, err := objMgr.Exists([]byte(req.Key))
		require.NoError(err)
		assert.False(exists)
	})

	// deleting a non existing object doesn't return an error.
	t.Run("non exists", func(t *testing.T) {
		req := &pb.DeleteObjectRequest{
			Label: label,
			Key:   []byte("nonexists"),
		}

		_, err := api.Delete(context.Background(), req)
		require.NoError(err)

		exists, err := objMgr.Exists([]byte(req.Key))
		require.NoError(err)
		assert.False(exists)
	})
}

func TestCheckObject(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	api, clean := getTestObjectAPI(require)
	defer clean()

	label, _ := populateDB(require, api.db)
	objMgr := manager.NewObjectManager(label, api.db)

	tt := []struct {
		name           string
		key            []byte
		expectedStatus manager.CheckStatus
	}{
		{
			name:           "valid",
			key:            []byte("testkey1"),
			expectedStatus: manager.CheckStatusOK,
		},
		{
			name:           "missing",
			key:            []byte("dontexsits"),
			expectedStatus: manager.CheckStatusMissing,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			status, err := objMgr.Check(tc.key)
			require.NoError(err, "failed to check status of %v", tc.key)
			assert.Equal(tc.expectedStatus, status)
		})
	}
}
