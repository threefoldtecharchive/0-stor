package grpc

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/client/datastor"

	"github.com/zero-os/0-stor/server/api/grpc/rpctypes"
	pb "github.com/zero-os/0-stor/server/api/grpc/schema"
)

func TestNewClientPanics(t *testing.T) {
	require := require.New(t)

	require.Panics(func() {
		NewClient("", "", nil)
	}, "no address given")
	require.Panics(func() {
		NewClient("foo", "", nil)
	}, "no label given")
}

func TestClientSetObject(t *testing.T) {
	require := require.New(t)

	var os stubObjectService
	client := Client{objService: &os}
	client.contextConstructor = client.defaultContextConstructor

	err := client.SetObject(datastor.Object{})
	require.NoError(err, "server returns no error -> no error")

	os.err = rpctypes.ErrGRPCNilLabel
	err = client.SetObject(datastor.Object{})
	require.Equal(rpctypes.ErrNilLabel, err, "server returns GRPC error -> client error")

	errFoo := errors.New("fooErr")
	os.err = errFoo
	err = client.SetObject(datastor.Object{})
	require.Equal(errFoo, err, "any other error -> client error")
}

func TestClientGetObject(t *testing.T) {
	require := require.New(t)

	var os stubObjectService
	client := Client{objService: &os}
	client.contextConstructor = client.defaultContextConstructor

	obj, err := client.GetObject(nil)
	require.Equal(datastor.ErrMissingData, err)
	require.Nil(obj)

	os.data = []byte("foo")
	obj, err = client.GetObject(nil)
	require.NoError(err)
	require.NotNil(obj)
	require.Nil(obj.Key)
	require.Equal([]byte("foo"), obj.Data)
	require.Empty(obj.ReferenceList)

	os.err = rpctypes.ErrGRPCNilLabel
	obj, err = client.GetObject(nil)
	require.Equal(rpctypes.ErrNilLabel, err, "server returns GRPC error -> client error")
	require.Nil(obj)

	errFoo := errors.New("fooErr")
	os.err = errFoo
	obj, err = client.GetObject(nil)
	require.Equal(errFoo, err, "any other error -> client error")
	require.Nil(obj)
}

func TestClientDeleteObject(t *testing.T) {
	require := require.New(t)

	var os stubObjectService
	client := Client{objService: &os}
	client.contextConstructor = client.defaultContextConstructor

	err := client.DeleteObject(nil)
	require.NoError(err, "server returns no error -> no error")

	os.err = rpctypes.ErrGRPCNilLabel
	err = client.DeleteObject(nil)
	require.Equal(rpctypes.ErrNilLabel, err, "server returns GRPC error -> client error")

	errFoo := errors.New("fooErr")
	os.err = errFoo
	err = client.DeleteObject(nil)
	require.Equal(errFoo, err, "any other error -> client error")
}

func TestClientGetObjectStatus(t *testing.T) {
	require := require.New(t)

	var os stubObjectService
	client := Client{objService: &os}
	client.contextConstructor = client.defaultContextConstructor

	status, err := client.GetObjectStatus(nil)
	require.NoError(err)
	require.Equal(datastor.ObjectStatusMissing, status)

	os.status = math.MaxInt32
	status, err = client.GetObjectStatus(nil)
	require.Equal(datastor.ErrInvalidStatus, err)
	require.Equal(datastor.ObjectStatus(0), status)

	os.status = pb.ObjectStatusCorrupted
	status, err = client.GetObjectStatus(nil)
	require.NoError(err)
	require.Equal(datastor.ObjectStatusCorrupted, status)

	os.err = rpctypes.ErrGRPCNilLabel
	status, err = client.GetObjectStatus(nil)
	require.Equal(rpctypes.ErrNilLabel, err, "server returns GRPC error -> client error")
	require.Equal(datastor.ObjectStatus(0), status)

	errFoo := errors.New("fooErr")
	os.err = errFoo
	status, err = client.GetObjectStatus(nil)
	require.Equal(errFoo, err, "any other error -> client error")
	require.Equal(datastor.ObjectStatus(0), status)
}

func TestClientExistObject(t *testing.T) {
	require := require.New(t)

	var os stubObjectService
	client := Client{objService: &os}
	client.contextConstructor = client.defaultContextConstructor

	exists, err := client.ExistObject(nil)
	require.NoError(err)
	require.False(exists)

	os.status = math.MaxInt32
	exists, err = client.ExistObject(nil)
	require.Equal(datastor.ErrInvalidStatus, err)
	require.False(exists)

	os.status = pb.ObjectStatusCorrupted
	exists, err = client.ExistObject(nil)
	require.Equal(datastor.ErrObjectCorrupted, err)
	require.False(exists)

	os.status = pb.ObjectStatusOK
	exists, err = client.ExistObject(nil)
	require.NoError(err)
	require.True(exists)

	os.status = pb.ObjectStatusMissing
	exists, err = client.ExistObject(nil)
	require.NoError(err)
	require.False(exists)

	os.err = rpctypes.ErrGRPCNilLabel
	exists, err = client.ExistObject(nil)
	require.Equal(rpctypes.ErrNilLabel, err, "server returns GRPC error -> client error")
	require.False(exists)

	errFoo := errors.New("fooErr")
	os.err = errFoo
	exists, err = client.ExistObject(nil)
	require.Equal(errFoo, err, "any other error -> client error")
	require.False(exists)
}

func TestClientGetNamespace(t *testing.T) {
	require := require.New(t)

	var nss stubNamespaceService
	client := Client{namespaceService: &nss, label: "myLabel"}
	client.contextConstructor = client.defaultContextConstructor

	ns, err := client.GetNamespace()
	require.Equal(datastor.ErrInvalidLabel, err, "returned label should be equal to client's label")
	require.Nil(ns)

	nss.label = "myLabel"
	ns, err = client.GetNamespace()
	require.NoError(err, "returned label should be equal to client's label")
	require.NotNil(ns)
	require.Equal("myLabel", ns.Label)
	require.Equal(int64(0), ns.ReadRequestPerHour)
	require.Equal(int64(0), ns.WriteRequestPerHour)
	require.Equal(int64(0), ns.NrObjects)

	nss.err = rpctypes.ErrGRPCNilLabel
	ns, err = client.GetNamespace()
	require.Equal(rpctypes.ErrNilLabel, err, "server errors should be caught by client")
	require.Nil(ns)

	nss.nrOfObjects = 42
	nss.readRPH = 1023
	nss.writeRPH = math.MaxInt64
	nss.err = nil
	ns, err = client.GetNamespace()
	require.NoError(err)
	require.NotNil(ns)
	require.Equal("myLabel", ns.Label)
	require.Equal(int64(1023), ns.ReadRequestPerHour)
	require.Equal(int64(math.MaxInt64), ns.WriteRequestPerHour)
	require.Equal(int64(42), ns.NrObjects)

	client.label = "foo"
	ns, err = client.GetNamespace()
	require.Equal(datastor.ErrInvalidLabel, err, "returned label should be equal to client's label")
	require.Nil(ns)
}

func TestClientListObjectKeys(t *testing.T) {
	require := require.New(t)

	var os stubObjectService
	client := Client{objService: &os}
	client.contextConstructor = client.defaultContextConstructor

	require.Panics(func() {
		client.ListObjectKeyIterator(nil)
	}, "no context given")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := client.ListObjectKeyIterator(ctx)
	require.NoError(err, "server returns no error -> no error")
	require.NotNil(ch)
	select {
	case resp, open := <-ch:
		require.True(open)
		require.Equal(datastor.ErrMissingKey, resp.Error)
		require.Nil(resp.Key)
	case <-time.After(time.Millisecond * 500):
	}

	os.err = rpctypes.ErrGRPCNilLabel
	ch, err = client.ListObjectKeyIterator(context.Background())
	require.Equal(rpctypes.ErrNilLabel, err)
	require.Nil(ch)

	os.err = nil
	os.streamErr = datastor.ErrMissingKey
	ch, err = client.ListObjectKeyIterator(context.Background())
	require.NoError(err, "channel should be created")
	require.NotNil(ch)
	select {
	case resp, open := <-ch:
		require.True(open)
		require.Equal(datastor.ErrMissingKey, resp.Error)
		require.Nil(resp.Key)
	case <-time.After(time.Millisecond * 500):
		t.Fatal("nothing received from channel while it was expected")
	}
	select {
	case _, open := <-ch:
		require.False(open)
	case <-time.After(time.Millisecond * 500):
	}

	os.streamErr = nil
	os.key = []byte("foo")
	ch, err = client.ListObjectKeyIterator(context.Background())
	require.NoError(err, "channel should be created")
	require.NotNil(ch)
	select {
	case resp, open := <-ch:
		require.True(open)
		require.NoError(resp.Error)
		require.Equal([]byte("foo"), resp.Key)
	case <-time.After(time.Millisecond * 500):
		t.Fatal("nothing received from channel while it was expected")
	}
	select {
	case _, open := <-ch:
		require.False(open)
	case <-time.After(time.Millisecond * 500):
	}
}

func TestClientSetReferenceList(t *testing.T) {
	require := require.New(t)

	var os stubObjectService
	client := Client{objService: &os}
	client.contextConstructor = client.defaultContextConstructor

	err := client.SetReferenceList(nil, nil)
	require.NoError(err, "server returns no error -> no error")

	os.err = rpctypes.ErrGRPCNilLabel
	err = client.SetReferenceList(nil, nil)
	require.Equal(rpctypes.ErrNilLabel, err, "server returns GRPC error -> client error")

	errFoo := errors.New("fooErr")
	os.err = errFoo
	err = client.SetReferenceList(nil, nil)
	require.Equal(errFoo, err, "any other error -> client error")
}

func TestClientGetReferenceList(t *testing.T) {
	require := require.New(t)

	var os stubObjectService
	client := Client{objService: &os}
	client.contextConstructor = client.defaultContextConstructor

	refList, err := client.GetReferenceList(nil)
	require.Equal(datastor.ErrMissingRefList, err)
	require.Nil(refList)

	os.refList = []string{"user1"}
	refList, err = client.GetReferenceList(nil)
	require.NoError(err, "server returns no error -> no error")
	require.Equal([]string{"user1"}, refList)

	os.err = rpctypes.ErrGRPCNilLabel
	refList, err = client.GetReferenceList(nil)
	require.Equal(rpctypes.ErrNilLabel, err, "server returns GRPC error -> client error")
	require.Nil(refList)

	errFoo := errors.New("fooErr")
	os.err = errFoo
	refList, err = client.GetReferenceList(nil)
	require.Equal(errFoo, err, "any other error -> client error")
	require.Nil(refList)
}

func TestClientGetReferenceCount(t *testing.T) {
	require := require.New(t)

	var os stubObjectService
	client := Client{objService: &os}
	client.contextConstructor = client.defaultContextConstructor

	count, err := client.GetReferenceCount(nil)
	require.NoError(err, "server returns no error -> no error")
	require.Equal(int64(0), count)

	os.refList = []string{"user1", "user2"}
	count, err = client.GetReferenceCount(nil)
	require.NoError(err, "server returns no error -> no error")
	require.Equal(int64(2), count)

	os.err = rpctypes.ErrGRPCNilLabel
	count, err = client.GetReferenceCount(nil)
	require.Equal(rpctypes.ErrNilLabel, err, "server returns GRPC error -> client error")
	require.Equal(int64(0), count)

	errFoo := errors.New("fooErr")
	os.err = errFoo
	count, err = client.GetReferenceCount(nil)
	require.Equal(errFoo, err, "any other error -> client error")
	require.Equal(int64(0), count)
}

func TestClientAppendToReferenceList(t *testing.T) {
	require := require.New(t)

	var os stubObjectService
	client := Client{objService: &os}
	client.contextConstructor = client.defaultContextConstructor

	err := client.AppendToReferenceList(nil, nil)
	require.NoError(err, "server returns no error -> no error")

	os.err = rpctypes.ErrGRPCNilLabel
	err = client.AppendToReferenceList(nil, nil)
	require.Equal(rpctypes.ErrNilLabel, err, "server returns GRPC error -> client error")

	errFoo := errors.New("fooErr")
	os.err = errFoo
	err = client.AppendToReferenceList(nil, nil)
	require.Equal(errFoo, err, "any other error -> client error")
}

func TestClientDeleteFromReferenceList(t *testing.T) {
	require := require.New(t)

	var os stubObjectService
	client := Client{objService: &os}
	client.contextConstructor = client.defaultContextConstructor

	count, err := client.DeleteFromReferenceList(nil, nil)
	require.NoError(err, "server returns no error -> no error")
	require.Equal(int64(0), count)

	os.refList = []string{"user1"}
	count, err = client.DeleteFromReferenceList(nil, nil)
	require.NoError(err, "server returns no error -> no error")
	require.Equal(int64(1), count)

	count, err = client.DeleteFromReferenceList(nil, []string{"user2", "user1"})
	require.NoError(err, "server returns no error -> no error")
	require.Equal(int64(0), count)

	os.err = rpctypes.ErrGRPCNilLabel
	count, err = client.DeleteFromReferenceList(nil, nil)
	require.Equal(rpctypes.ErrNilLabel, err, "server returns GRPC error -> client error")
	require.Equal(int64(0), count)

	errFoo := errors.New("fooErr")
	os.err = errFoo
	count, err = client.DeleteFromReferenceList(nil, nil)
	require.Equal(errFoo, err, "any other error -> client error")
	require.Equal(int64(0), count)
}

func TestClientDeleteReferenceList(t *testing.T) {
	require := require.New(t)

	var os stubObjectService
	client := Client{objService: &os}
	client.contextConstructor = client.defaultContextConstructor

	err := client.DeleteReferenceList(nil)
	require.NoError(err, "server returns no error -> no error")

	os.err = rpctypes.ErrGRPCNilLabel
	err = client.DeleteReferenceList(nil)
	require.Equal(rpctypes.ErrNilLabel, err, "server returns GRPC error -> client error")

	errFoo := errors.New("fooErr")
	os.err = errFoo
	err = client.DeleteReferenceList(nil)
	require.Equal(errFoo, err, "any other error -> client error")
}

func TestClientClosePanic(t *testing.T) {
	client := new(Client)
	require.Panics(t, func() {
		client.Close()
	}, "nil pointer dereference, because connection property is not set")
}
