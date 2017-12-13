package grpc

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/zero-os/0-stor/client/datastor"
	"github.com/zero-os/0-stor/server/api/grpc/rpctypes"

	"github.com/stretchr/testify/require"
)

func TestClientWithServer_API(t *testing.T) {
	require := require.New(t)

	client, _, clean, err := newServerClient()
	require.NoError(err)
	defer clean()
	require.NotNil(client)

	var (
		key = []byte("myKey")
	)

	// all getters should "work",
	// even though the key doesn't exist

	obj, err := client.GetObject(key)
	require.Equal(datastor.ErrKeyNotFound, err)
	require.Nil(obj)

	status, err := client.GetObjectStatus(key)
	require.NoError(err)
	require.Equal(datastor.ObjectStatusMissing, status)

	exists, err := client.ExistObject(key)
	require.NoError(err)
	require.False(exists)

	refCount, err := client.GetReferenceCount(key)
	require.NoError(err)
	require.Equal(int64(0), refCount)

	refList, err := client.GetReferenceList(key)
	require.Equal(datastor.ErrKeyNotFound, err)
	require.Empty(refList)

	// even the list object key iterator should work,
	// although it should close immediately

	ch, err := client.ListObjectKeyIterator(context.Background())
	require.NoError(err)
	select {
	case _, open := <-ch:
		require.False(open)
	case <-time.After(time.Millisecond * 500):
	}

	// now let's set something, but miss ome required params,
	// making these calls invalid and have no affect on the database's state

	err = client.SetObject(datastor.Object{})
	require.Equal(rpctypes.ErrNilKey, err)
	err = client.SetObject(datastor.Object{Key: key})
	require.Equal(rpctypes.ErrNilData, err)

	// let's ensure our object is still missing

	status, err = client.GetObjectStatus(key)
	require.NoError(err)
	require.Equal(datastor.ObjectStatusMissing, status)

	exists, err = client.ExistObject(key)
	require.NoError(err)
	require.False(exists)

	// now let's set the object for real

	data := []byte("myData")
	refList = []string{"user1"}
	err = client.SetObject(datastor.Object{
		Key:           key,
		Data:          data,
		ReferenceList: refList,
	})
	require.NoError(err)

	// our getters from earlier should work now

	obj, err = client.GetObject(key)
	require.NoError(err)
	require.NotNil(obj)
	require.Equal(key, obj.Key)
	require.Equal(data, obj.Data)
	require.Equal(refList, obj.ReferenceList)

	status, err = client.GetObjectStatus(key)
	require.NoError(err)
	require.Equal(datastor.ObjectStatusOK, status)

	exists, err = client.ExistObject(key)
	require.NoError(err)
	require.True(exists)

	refCount, err = client.GetReferenceCount(key)
	require.NoError(err)
	require.Equal(int64(1), refCount)

	refList, err = client.GetReferenceList(key)
	require.NoError(err)
	require.Equal([]string{"user1"}, refList)

	// The iterator will work now as well

	ch, err = client.ListObjectKeyIterator(context.Background())
	require.NoError(err)
	select {
	case result, open := <-ch:
		require.True(open)
		require.NoError(result.Error)
		require.Equal(key, result.Key)
	case <-time.After(time.Millisecond * 500):
	}
	select {
	case _, open := <-ch:
		require.False(open)
	case <-time.After(time.Millisecond * 500):
	}

	// now let's play a bit with the reference lists

	err = client.AppendToReferenceList(key, nil)
	require.Error(rpctypes.ErrNilRefList, err)

	err = client.AppendToReferenceList(key, []string{"user2", "user1"})
	require.NoError(err)

	refList, err = client.GetReferenceList(key)
	require.NoError(err)
	require.Len(refList, 3)
	sort.Strings(refList)
	require.Subset([]string{"user1", "user1", "user2"}, refList)

	refCount, err = client.DeleteFromReferenceList(key, []string{"user3"})
	require.NoError(err)
	require.Equal(int64(3), refCount)

	refCount, err = client.DeleteFromReferenceList(key, []string{"user1", "user2", "user4"})
	require.NoError(err)
	require.Equal(int64(1), refCount)

	refList, err = client.GetReferenceList(key)
	require.NoError(err)
	require.Equal([]string{"user1"}, refList)

	err = client.DeleteReferenceList(key)
	require.NoError(err)

	refList, err = client.GetReferenceList(key)
	require.Equal(datastor.ErrKeyNotFound, err)
	require.Empty(refList)

	refCount, err = client.DeleteFromReferenceList(key, []string{"user1", "user2", "user4"})
	require.NoError(err, "deleting from a non-existent refList is fine")
	require.Equal(int64(0), refCount)

	err = client.AppendToReferenceList(key, []string{"user1", "user2", "user2"})
	require.NoError(err, "appending to a non-existent refList is fine as well")

	refList, err = client.GetReferenceList(key)
	require.NoError(err)
	require.Len(refList, 3)
	sort.Strings(refList)
	require.Subset([]string{"user1", "user2", "user2"}, refList)

	refCount, err = client.GetReferenceCount(key)
	require.NoError(err)
	require.Equal(int64(3), refCount)

	otherKey := []byte("myOtherKey")
	otherData := []byte("some other data")
	// now let's add one more object, this time non-referenced
	err = client.SetObject(datastor.Object{
		Key:  otherKey,
		Data: otherData,
	})
	require.NoError(err)

	// let's add another one, it's starting to get fun
	yetAnotherKey := []byte("myOtherKey")
	whyNotData := []byte("why not data")
	// now let's add one more object, this time non-referenced
	err = client.SetObject(datastor.Object{
		Key:  yetAnotherKey,
		Data: whyNotData,
	})
	require.NoError(err)

	// let's give this one some references
	err = client.SetReferenceList(yetAnotherKey, []string{"a", "b"})
	require.NoError(err)

	// now let's list them all, they should all appear!
	objects := map[string]datastor.Object{
		string(key): datastor.Object{
			Key:           key,
			Data:          data,
			ReferenceList: refList,
		},
		string(otherKey): datastor.Object{
			Key:           otherKey,
			Data:          otherData,
			ReferenceList: nil,
		},
		string(yetAnotherKey): datastor.Object{
			Key:           yetAnotherKey,
			Data:          whyNotData,
			ReferenceList: []string{"b", "a"},
		},
	}
	// start the iteration
	ch, err = client.ListObjectKeyIterator(context.Background())
	require.NoError(err)
	for resp := range ch {
		require.NoError(resp.Error)
		require.NotNil(resp.Key)

		expObject, ok := objects[string(resp.Key)]
		require.True(ok)
		require.Equal(expObject.Key, resp.Key)
		delete(objects, string(resp.Key))

		obj, err := client.GetObject(expObject.Key)
		require.NoError(err)
		require.NotNil(obj)
		require.Equal(expObject.Key, obj.Key)
		require.Equal(expObject.Data, obj.Data)

		if len(expObject.ReferenceList) == 0 {
			require.Empty(obj.ReferenceList)
		} else {
			sort.Strings(expObject.ReferenceList)
			sort.Strings(obj.ReferenceList)
			require.Equal(expObject.ReferenceList, obj.ReferenceList)
		}
	}
	require.Empty(objects, "all objects should have been listed")
}

// we'll append one ref at a time, one 255 different goroutines at once,
// as to ensure that conflicts are resolved correctly
func TestClientWithServer_AppendToReferenceListAsync(t *testing.T) {
	// first create our database and object
	require := require.New(t)

	client, _, clean, err := newServerClient()
	require.NoError(err)
	defer clean()
	require.NotNil(client)

	key := []byte("testkey1")
	value := []byte{1, 2, 3, 4}

	err = client.SetObject(datastor.Object{
		Key:  key,
		Data: value,
	})
	require.NoError(err)

	// now append our reference list
	group, _ := errgroup.WithContext(context.Background())
	var expectedList []string
	for i := 0; i < 256; i++ {
		userID := fmt.Sprintf("user%d", i)
		expectedList = append(expectedList, userID)
		group.Go(func() error {
			return client.AppendToReferenceList(key, []string{userID})
		})
	}
	require.NoError(group.Wait())

	// now ensure our ref list is idd correct, even though we don't know the order
	refList, err := client.GetReferenceList(key)
	require.NoError(err)

	sort.Strings(refList)
	sort.Strings(expectedList)
	require.Equal(expectedList, refList)
}

// we'll append one ref at a time, one 255 different goroutines at once,
// as to ensure that conflicts are resolved correctly
func TestClientWithServer_DeleteFromReferenceListAsync(t *testing.T) {
	// first create our database and object
	require := require.New(t)

	client, _, clean, err := newServerClient()
	require.NoError(err)
	defer clean()
	require.NotNil(client)

	key := []byte("testkey1")
	value := []byte{1, 2, 3, 4}

	const refCount = 256

	var startRefList []string
	for i := 0; i < refCount; i++ {
		startRefList = append(startRefList, fmt.Sprintf("user%d", i))
	}

	err = client.SetObject(datastor.Object{
		Key:           key,
		Data:          value,
		ReferenceList: startRefList,
	})
	require.NoError(err)

	// ensure we have our ref list
	refList, err := client.GetReferenceList(key)
	require.NoError(err)

	sort.Strings(refList)
	sort.Strings(startRefList)
	require.Equal(startRefList, refList)

	// now remove from our reference list, one by one
	group, _ := errgroup.WithContext(context.Background())
	for i := 0; i < refCount; i++ {
		userID := fmt.Sprintf("user%d", i)
		group.Go(func() error {
			_, err := client.DeleteFromReferenceList(key, []string{userID})
			return err
		})
	}
	require.NoError(group.Wait())

	// now ensure our ref list is now gone
	refList, err = client.GetReferenceList(key)
	require.Equal(datastor.ErrKeyNotFound, err)
	require.Empty(refList)

	referenceCount, err := client.GetReferenceCount(key)
	require.NoError(err)
	require.Equal(int64(0), referenceCount)
}
