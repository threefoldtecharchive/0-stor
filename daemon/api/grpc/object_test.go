package grpc

import (
	"bytes"
	"crypto/rand"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/zero-os/0-stor/client"
	"github.com/zero-os/0-stor/client/components/storage"
	"github.com/zero-os/0-stor/client/datastor"
	"github.com/zero-os/0-stor/client/metastor"
	pb "github.com/zero-os/0-stor/daemon/api/grpc/schema"
)

func TestWrite(t *testing.T) {
	objSrv := newObjectSrv(newObjClientStub())

	req := &pb.WriteRequest{
		Key:   []byte("key"),
		Value: []byte("value"),
	}

	rep, err := objSrv.Write(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, rep.Meta)
}

func TestWriteError(t *testing.T) {
	objSrv := newObjectSrv(newObjClientStub())

	// nil key meta
	req := &pb.WriteRequest{
		Value: []byte("value"),
	}

	_, err := objSrv.Write(context.Background(), req)
	require.Equal(t, errNilKeyMeta, err)
}

func TestWriteStream(t *testing.T) {
	objSrv := newObjectSrv(newObjClientStub())

	// with key
	val := make([]byte, 4096)
	rand.Read(val)
	stream := &writeStreamServer{
		key:     []byte("myKey"),
		val:     val,
		refList: []string{"ref1"},
	}
	err := objSrv.WriteStream(stream)
	require.NoError(t, err)

	// with meta
	stream = &writeStreamServer{
		meta:    &pb.Meta{Key: []byte("keyMeta")},
		val:     val,
		refList: []string{"ref1"},
	}
	err = objSrv.WriteStream(stream)
	require.NoError(t, err)

}

type writeStreamServer struct {
	key     []byte
	meta    *pb.Meta
	val     []byte
	refList []string
	grpc.ServerStream
}

func (wss *writeStreamServer) SendAndClose(rep *pb.WriteStreamReply) error {
	return nil
}

func (wss *writeStreamServer) Recv() (*pb.WriteStreamRequest, error) {
	if len(wss.key) == 0 && wss.meta == nil {
		return nil, io.EOF
	}

	req := &pb.WriteStreamRequest{
		Key:           wss.key,
		Meta:          wss.meta,
		Value:         wss.val,
		ReferenceList: wss.refList,
	}
	wss.key = nil
	wss.meta = nil
	return req, nil
}

func TestWriteFile(t *testing.T) {
	objSrv := newObjectSrv(newObjClientStub())

	tmp, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	defer func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}()

	_, err = io.Copy(tmp, bytes.NewReader([]byte("dadadadadada")))
	require.NoError(t, err)

	req := &pb.WriteFileRequest{
		Key:      []byte("mykey"),
		FilePath: tmp.Name(),
	}
	rep, err := objSrv.WriteFile(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, rep.Meta)
}

func TestWriteFileNilKeyMeta(t *testing.T) {
	objSrv := newObjectSrv(newObjClientStub())

	tmp, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	defer func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}()

	req := &pb.WriteFileRequest{
		FilePath: tmp.Name(),
	}
	_, err = objSrv.WriteFile(context.Background(), req)
	require.Equal(t, errNilKeyMeta, err)
}

func TestWriteFileNilPath(t *testing.T) {
	objSrv := newObjectSrv(newObjClientStub())

	req := &pb.WriteFileRequest{
		Key: []byte("mykye"),
	}
	_, err := objSrv.WriteFile(context.Background(), req)
	require.Error(t, err)
}

func TestRead(t *testing.T) {
	cs := newObjClientStub()
	objSrv := newObjectSrv(cs)

	var (
		refList = []string{"myref"}
		key     = []byte("key")
		val     = []byte("val")
	)
	cs.set(newObjStub(key, val, refList))

	// read by key
	rep, err := objSrv.Read(context.Background(), &pb.ReadRequest{
		Key: key,
	})
	require.NoError(t, err)
	require.Equal(t, val, rep.Value)
	require.Equal(t, refList, rep.ReferenceList)

	// read by meta
	rep, err = objSrv.Read(context.Background(), &pb.ReadRequest{
		Meta: &pb.Meta{Key: key},
	})

	require.NoError(t, err)
	require.Equal(t, val, rep.Value)
	require.Equal(t, refList, rep.ReferenceList)

}

func TestReadNilKeyMeta(t *testing.T) {
	objSrv := newObjectSrv(newObjClientStub())
	_, err := objSrv.Read(context.Background(), &pb.ReadRequest{})
	require.Error(t, err)
	require.Equal(t, errNilKeyMeta, err)
}

func TestReadStream(t *testing.T) {
	cs := newObjClientStub()
	objSrv := newObjectSrv(cs)

	var (
		refList = []string{"myref"}
		key     = []byte("key")
		val     = []byte("val")
	)
	cs.set(newObjStub(key, val, refList))

	req := &pb.ReadStreamRequest{
		Key: key,
	}
	stream := &readStreamServer{}
	err := objSrv.ReadStream(req, stream)
	require.NoError(t, err)
}

type readStreamServer struct {
	grpc.ServerStream
	val []byte
}

func (rss *readStreamServer) Send(resp *pb.ReadStreamReply) error {
	return nil
}

func TestReadStreamError(t *testing.T) {
	cs := newObjClientStub()
	objSrv := newObjectSrv(cs)

	req := &pb.ReadStreamRequest{}

	stream := &readStreamServer{}
	err := objSrv.ReadStream(req, stream)
	require.Equal(t, errNilKey, err)
}

func TestReadFile(t *testing.T) {
	cs := newObjClientStub()
	objSrv := newObjectSrv(cs)

	var (
		refList = []string{"myref"}
		key     = []byte("key")
		val     = []byte("val")
	)
	cs.set(newObjStub(key, val, refList))

	// read by key
	tmpReadByKey, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	defer func() {
		tmpReadByKey.Close()
		os.Remove(tmpReadByKey.Name())
	}()

	rep, err := objSrv.ReadFile(context.Background(), &pb.ReadFileRequest{
		Key:      key,
		FilePath: tmpReadByKey.Name(),
	})
	require.NoError(t, err)
	require.Equal(t, refList, rep.ReferenceList)
	contentReadByKey, err := ioutil.ReadAll(tmpReadByKey)
	require.NoError(t, err)
	require.Equal(t, val, contentReadByKey)
}

func TestReadFileError(t *testing.T) {
	objSrv := newObjectSrv(newObjClientStub())

	// nil key
	_, err := objSrv.ReadFile(context.Background(), &pb.ReadFileRequest{
		FilePath: "/tmp/dada",
	})
	require.Error(t, err)
	require.Equal(t, errNilKey, err)

	// empty file path
	_, err = objSrv.ReadFile(context.Background(), &pb.ReadFileRequest{
		Key: []byte("mykey"),
	})
	require.Error(t, err)
	require.Equal(t, errNilFilePath, err)
}

func TestDelete(t *testing.T) {
	cs := newObjClientStub()
	objSrv := newObjectSrv(cs)

	var (
		refList = []string{"myref"}
		key     = []byte("key")
		val     = []byte("val")
	)
	cs.set(newObjStub(key, val, refList))

	// delete by key
	_, err := objSrv.Delete(context.Background(), &pb.DeleteRequest{
		Key: key,
	})

	require.NoError(t, err)

	cs.set(newObjStub(key, val, refList))

	// delete by meta
	_, err = objSrv.Delete(context.Background(), &pb.DeleteRequest{
		Meta: &pb.Meta{Key: key},
	})

	require.NoError(t, err)

	// delete non existed key
	_, err = objSrv.Delete(context.Background(), &pb.DeleteRequest{
		Key: key,
	})

	require.Equal(t, datastor.ErrKeyNotFound, err)
}

func TestDeleteNilKeyMeta(t *testing.T) {
	objSrv := newObjectSrv(newObjClientStub())

	_, err := objSrv.Delete(context.Background(), &pb.DeleteRequest{})

	require.Equal(t, errNilKeyMeta, err)
}

func TestAppendReferenceList(t *testing.T) {
	cs := newObjClientStub()
	objSrv := newObjectSrv(cs)

	var (
		refList = []string{"myref"}
		key     = []byte("key")
		val     = []byte("val")
	)
	cs.set(newObjStub(key, val, nil))

	// by key
	_, err := objSrv.AppendReferenceList(context.Background(), &pb.AppendReferenceListRequest{
		Key:           key,
		ReferenceList: refList,
	})

	require.NoError(t, err)

	// by meta
	_, err = objSrv.AppendReferenceList(context.Background(), &pb.AppendReferenceListRequest{
		Meta:          &pb.Meta{Key: key},
		ReferenceList: refList,
	})

	require.NoError(t, err)

}

func TestWalk(t *testing.T) {
	cs := newObjClientStub()
	objSrv := newObjectSrv(cs)

	err := objSrv.Walk(&pb.WalkRequest{
		StartKey:  []byte("dadad"),
		FromEpoch: 1,
		ToEpoch:   10,
	}, &walkStreamSterver{})
	require.NoError(t, err)
}

func TestWalkError(t *testing.T) {
	cs := newObjClientStub()
	objSrv := newObjectSrv(cs)

	err := objSrv.Walk(&pb.WalkRequest{
		FromEpoch: 1,
		ToEpoch:   10,
	}, &walkStreamSterver{})
	require.Equal(t, errNilKey, err)
}

type walkStreamSterver struct {
	grpc.ServerStream
}

func (wss *walkStreamSterver) Send(rep *pb.WalkReply) error {
	return nil
}

func TestAppendReferenceListError(t *testing.T) {
	objSrv := newObjectSrv(newObjClientStub())

	// without key and meta
	_, err := objSrv.AppendReferenceList(context.Background(), &pb.AppendReferenceListRequest{
		ReferenceList: []string{"aa"},
	})

	require.Equal(t, errNilKeyMeta, err)

	// without ref list
	_, err = objSrv.AppendReferenceList(context.Background(), &pb.AppendReferenceListRequest{
		Key: []byte("key"),
	})

	require.Equal(t, errEmptyRefList, err)

	// object not exist
	_, err = objSrv.AppendReferenceList(context.Background(), &pb.AppendReferenceListRequest{
		Key:           []byte("key"),
		ReferenceList: []string{"aa"},
	})

	require.Equal(t, datastor.ErrKeyNotFound, err)
}

func TestRemoveReferenceList(t *testing.T) {
	cs := newObjClientStub()
	objSrv := newObjectSrv(cs)

	var (
		refList = []string{"myref"}
		key     = []byte("key")
		val     = []byte("val")
	)
	cs.set(newObjStub(key, val, refList))

	// by key
	_, err := objSrv.RemoveReferenceList(context.Background(), &pb.RemoveReferenceListRequest{
		Key:           key,
		ReferenceList: refList,
	})

	require.NoError(t, err)

	cs.set(newObjStub(key, val, refList))
	// by meta
	_, err = objSrv.RemoveReferenceList(context.Background(), &pb.RemoveReferenceListRequest{
		Meta:          &pb.Meta{Key: key},
		ReferenceList: refList,
	})

	require.NoError(t, err)

}

func TestRemoveReferenceListError(t *testing.T) {
	objSrv := newObjectSrv(newObjClientStub())

	// without key and meta
	_, err := objSrv.RemoveReferenceList(context.Background(), &pb.RemoveReferenceListRequest{
		ReferenceList: []string{"aa"},
	})

	require.Equal(t, errNilKeyMeta, err)

	// without ref list
	_, err = objSrv.RemoveReferenceList(context.Background(), &pb.RemoveReferenceListRequest{
		Key: []byte("key"),
	})

	require.Equal(t, errEmptyRefList, err)

	// object not exist
	_, err = objSrv.RemoveReferenceList(context.Background(), &pb.RemoveReferenceListRequest{
		Key:           []byte("key"),
		ReferenceList: []string{"aa"},
	})

	require.Equal(t, datastor.ErrKeyNotFound, err)
}

func TestCheck(t *testing.T) {
	cs := newObjClientStub()
	objSrv := newObjectSrv(cs)

	var (
		refList = []string{"myref"}
		key     = []byte("key")
		val     = []byte("val")
	)
	cs.set(newObjStub(key, val, refList))

	_, err := objSrv.Check(context.Background(), &pb.CheckRequest{
		Key: key,
	})
	require.NoError(t, err)
}

func TestCheckError(t *testing.T) {
	cs := newObjClientStub()
	objSrv := newObjectSrv(cs)

	// nil key
	_, err := objSrv.Check(context.Background(), &pb.CheckRequest{})
	require.Equal(t, errNilKey, err)
}

func TestRepair(t *testing.T) {
	cs := newObjClientStub()
	objSrv := newObjectSrv(cs)

	var (
		refList = []string{"myref"}
		key     = []byte("key")
		val     = []byte("val")
	)
	cs.set(newObjStub(key, val, refList))

	_, err := objSrv.Repair(context.Background(), &pb.RepairRequest{
		Key: key,
	})
	require.NoError(t, err)

}

func TestRepairError(t *testing.T) {
	cs := newObjClientStub()
	objSrv := newObjectSrv(cs)

	// nil key
	_, err := objSrv.Repair(context.Background(), &pb.RepairRequest{})
	require.Equal(t, errNilKey, err)
}

func TestPbChunkToStorChunk(t *testing.T) {
	var (
		key1   = []byte("key_1")
		key2   = []byte("key_2")
		shards = []string{"shards"}
	)
	pbChunks := []*pb.Chunk{
		&pb.Chunk{
			Size_:  1,
			Key:    key1,
			Shards: shards,
		},
		&pb.Chunk{
			Size_:  2,
			Key:    key2,
			Shards: shards,
		},
	}
	storChunks := pbChunksToStorChunks(pbChunks)
	require.Equal(t, 2, len(storChunks))
	require.Equal(t, &metastor.Chunk{
		Size:   1,
		Key:    key1,
		Shards: shards,
	}, storChunks[0])
	require.Equal(t, &metastor.Chunk{
		Size:   2,
		Key:    key2,
		Shards: shards,
	}, storChunks[1])

}

func TestStorChunkToPbChunk(t *testing.T) {
	var (
		key1   = []byte("key_1")
		key2   = []byte("key_2")
		shards = []string{"shards"}
	)
	storChunks := []*metastor.Chunk{
		&metastor.Chunk{
			Size:   1,
			Key:    key1,
			Shards: shards,
		},
		&metastor.Chunk{
			Size:   2,
			Key:    key2,
			Shards: shards,
		},
	}

	pbChunks := storChunksToPbChunks(storChunks)
	require.Equal(t, 2, len(pbChunks))
	require.Equal(t, &pb.Chunk{
		Size_:  1,
		Key:    key1,
		Shards: shards,
	}, pbChunks[0])

	require.Equal(t, &pb.Chunk{
		Size_:  2,
		Key:    key2,
		Shards: shards,
	}, pbChunks[1])

}

type objStub struct {
	key     []byte
	val     []byte
	refList []string
	md      *metastor.Data
	status  storage.ObjectCheckStatus
}

type objClientStub struct {
	kv  map[string]objStub
	mux sync.Mutex
}

func newObjClientStub() *objClientStub {
	return &objClientStub{
		kv: make(map[string]objStub),
	}
}

func (c *objClientStub) get(key []byte) (objStub, error) {
	c.mux.Lock()
	defer c.mux.Unlock()
	obj, ok := c.kv[string(key)]
	if !ok {
		return obj, datastor.ErrKeyNotFound
	}
	return obj, nil
}

func (c *objClientStub) set(obj objStub) {
	c.mux.Lock()
	c.kv[string(obj.key)] = obj
	c.mux.Unlock()
}

func (c *objClientStub) delete(key []byte) {
	c.mux.Lock()
	delete(c.kv, string(key))
	c.mux.Unlock()
}

func (c *objClientStub) setObjStatus(key []byte, status storage.ObjectCheckStatus) error {
	obj, err := c.get(key)
	if err != nil {
		return err
	}
	obj.status = status
	c.set(obj)
	return nil
}

func (c *objClientStub) populateObj(numObj int, refList []string) (kvs map[string]objStub) {
	for i := 0; i < numObj; i++ {
		key := make([]byte, 10)
		val := make([]byte, 30)
		rand.Read(key)
		rand.Read(val)
		obj := newObjStub(key, val, refList)
		c.set(obj)
		kvs[string(key)] = obj
	}
	return
}

func newObjStub(key []byte, val []byte, refList []string) objStub {
	md := &metastor.Data{
		Key: key,
	}
	return objStub{
		key:     key,
		val:     val,
		refList: refList,
		md:      md,
		status:  storage.ObjectCheckStatusOptimal,
	}
}

func (c *objClientStub) WriteF(key []byte, r io.Reader, refList []string) (*metastor.Data, error) {
	val, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	obj := newObjStub(key, val, refList)
	c.set(obj)
	return obj.md, nil
}

func (c *objClientStub) WriteWithMeta(key, val []byte, prevKey []byte, prevMeta, meta *metastor.Data, refList []string) (*metastor.Data, error) {
	obj := newObjStub(key, val, refList)
	obj.md.Previous = prevKey
	if prevMeta != nil {
		obj.md.Previous = prevMeta.Previous
	}

	c.set(obj)
	return obj.md, nil
}

func (c *objClientStub) WriteFWithMeta(key []byte, r io.Reader, prevKey []byte, prevMeta, meta *metastor.Data, refList []string) (*metastor.Data, error) {
	val, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	obj := newObjStub(key, val, refList)
	obj.md.Previous = prevKey
	if prevMeta != nil {
		obj.md.Previous = prevMeta.Previous
	}

	c.set(obj)
	return obj.md, nil
}

// read
func (c *objClientStub) Read(key []byte) ([]byte, []string, error) {
	obj, err := c.get(key)
	if err != nil {
		return nil, nil, err
	}
	return obj.val, obj.refList, nil
}

func (c *objClientStub) ReadF(key []byte, w io.Writer) ([]string, error) {
	obj, err := c.get(key)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(w, bytes.NewReader(obj.val)); err != nil {
		return nil, err
	}
	return obj.refList, nil
}
func (c *objClientStub) ReadWithMeta(md *metastor.Data) ([]byte, []string, error) {
	return c.Read(md.Key)
}

// delete
func (c *objClientStub) Delete(key []byte) error {
	if _, err := c.get(key); err != nil {
		return err
	}
	c.delete(key)
	return nil
}

func (c *objClientStub) DeleteWithMeta(md *metastor.Data) error {
	return c.Delete(md.Key)
}

func (c *objClientStub) appendToReferenceList(key []byte, refList []string) error {
	obj, err := c.get(key)
	if err != nil {
		return err
	}
	obj.refList = append(obj.refList, refList...)
	c.set(obj)
	return nil
}

// reference list
func (c *objClientStub) AppendToReferenceList(key []byte, refList []string) error {
	return c.appendToReferenceList(key, refList)
}

func (c *objClientStub) AppendToReferenceListWithMeta(md *metastor.Data, refList []string) error {
	return c.appendToReferenceList(md.Key, refList)
}
func (c *objClientStub) deleteFromReferenceList(key []byte, refList []string) error {
	obj, err := c.get(key)
	if err != nil {
		return err
	}

	var newRefList []string
	for _, objRl := range obj.refList {
		toBeDeleted := func() bool {
			for _, rl := range refList {
				if rl == objRl {
					return true
				}
			}
			return false
		}()
		if !toBeDeleted {
			newRefList = append(newRefList, objRl)
		}
	}
	obj.refList = newRefList
	c.set(obj)
	return nil
}

func (c *objClientStub) DeleteFromReferenceList(key []byte, refList []string) error {
	return c.deleteFromReferenceList(key, refList)
}
func (c *objClientStub) DeleteFromReferenceListWithMeta(md *metastor.Data, refList []string) error {
	return c.deleteFromReferenceList(md.Key, refList)
}

func (c *objClientStub) Walk(startKey []byte, fromEpoch, toEpoch int64) <-chan *client.WalkResult {
	ch := make(chan *client.WalkResult)
	go func() {
		defer close(ch)
		ch <- &client.WalkResult{
			Key: startKey,
			Meta: &metastor.Data{
				Key:   startKey,
				Epoch: fromEpoch,
			},
			Data: startKey,
		}
	}()
	return ch
}

func (c *objClientStub) Check(key []byte) (storage.ObjectCheckStatus, error) {
	obj, err := c.get(key)
	if err != nil {
		return storage.ObjectCheckStatus(0), err
	}
	return obj.status, nil
}

func (c *objClientStub) Repair(key []byte) error {
	obj, err := c.get(key)
	if err != nil {
		return err
	}
	obj.status = storage.ObjectCheckStatusOptimal
	c.set(obj)
	return nil
}
