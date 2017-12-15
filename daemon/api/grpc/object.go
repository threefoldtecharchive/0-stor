package grpc

import (
	"errors"
	"io"
	"os"

	"github.com/zero-os/0-stor/client"
	"github.com/zero-os/0-stor/client/components/storage"
	"github.com/zero-os/0-stor/client/metastor"
	pb "github.com/zero-os/0-stor/daemon/api/grpc/schema"
	"golang.org/x/net/context"
)

var (
	errNilKeyMeta        = errors.New("both key and meta are nil")
	errNilFilePath       = errors.New("nil file path")
	errNilKey            = errors.New("nil key")
	errEmptyRefList      = errors.New("empty reference list")
	errInvalidCheckReply = errors.New("invalid check reply from 0-stor client")
)

type objectClient interface {
	// write
	WriteF(key []byte, r io.Reader, refList []string) (*metastor.Data, error)
	WriteWithMeta(key, val []byte, prevKey []byte, prevMeta, meta *metastor.Data, refList []string) (*metastor.Data, error)
	WriteFWithMeta(key []byte, r io.Reader, prevKey []byte, prevMeta, meta *metastor.Data, refList []string) (*metastor.Data, error)

	// read
	Read(key []byte) ([]byte, []string, error)
	ReadF(key []byte, w io.Writer) ([]string, error)
	ReadWithMeta(md *metastor.Data) ([]byte, []string, error)

	// delete
	Delete(key []byte) error
	DeleteWithMeta(md *metastor.Data) error

	// reference list
	AppendToReferenceList(key []byte, refList []string) error
	AppendToReferenceListWithMeta(md *metastor.Data, refList []string) error
	DeleteFromReferenceList(key []byte, refList []string) error
	DeleteFromReferenceListWithMeta(md *metastor.Data, refList []string) error

	Walk(startKey []byte, fromEpoch, toEpoch int64) <-chan *client.WalkResult
	Check(key []byte) (storage.ObjectCheckStatus, error)
	Repair(key []byte) error
}

type objectSrv struct {
	client objectClient
}

func newObjectSrv(client objectClient) *objectSrv {
	return &objectSrv{
		client: client,
	}
}

func (osrv *objectSrv) Write(ctx context.Context, req *pb.WriteRequest) (*pb.WriteReply, error) {
	if len(req.Key) == 0 && req.Meta == nil {
		return nil, errNilKeyMeta
	}

	meta, err := osrv.client.WriteWithMeta([]byte(req.Key),
		[]byte(req.Value),
		[]byte(req.PrevKey),
		pbMetaToStorMeta(req.PrevMeta),
		pbMetaToStorMeta(req.Meta),
		req.ReferenceList)

	if err != nil {
		return nil, err
	}

	return &pb.WriteReply{
		Meta: storMetaToPbMeta(meta),
	}, nil
}

func (osrv *objectSrv) WriteFile(ctx context.Context, req *pb.WriteFileRequest) (*pb.WriteFileReply, error) {
	// check input
	if len(req.Key) == 0 && req.Meta == nil {
		return nil, errNilKeyMeta
	}

	if len(req.FilePath) == 0 {
		return nil, errNilFilePath
	}

	// open the given file path
	f, err := os.Open(req.FilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	meta, err := osrv.client.WriteFWithMeta([]byte(req.Key),
		f,
		[]byte(req.PrevKey),
		pbMetaToStorMeta(req.PrevMeta),
		pbMetaToStorMeta(req.Meta),
		req.ReferenceList)

	if err != nil {
		return nil, err
	}

	return &pb.WriteFileReply{
		Meta: storMetaToPbMeta(meta),
	}, nil

}

func (osrv *objectSrv) WriteStream(stream pb.ObjectService_WriteStreamServer) error {
	// creates the reader
	sr, req, err := newWriteStreamReader(stream)
	if err != nil {
		return err
	}

	var meta *metastor.Data

	if req.Meta != nil {
		meta, err = osrv.client.WriteFWithMeta(req.Key, sr, req.PrevKey,
			pbMetaToStorMeta(req.PrevMeta),
			pbMetaToStorMeta(req.Meta),
			req.ReferenceList)
	} else {
		meta, err = osrv.client.WriteF(req.Key, sr, req.ReferenceList)
	}
	if err != nil {
		return err
	}
	return stream.SendAndClose(&pb.WriteStreamReply{
		Meta: storMetaToPbMeta(meta),
	})
}

func (osrv *objectSrv) Read(ctx context.Context, req *pb.ReadRequest) (*pb.ReadReply, error) {
	// check input
	if len(req.Key) == 0 && req.Meta == nil {
		return nil, errNilKeyMeta
	}

	var (
		value   []byte
		refList []string
		err     error
	)
	if req.Meta != nil {
		value, refList, err = osrv.client.ReadWithMeta(pbMetaToStorMeta(req.Meta))
	} else {
		value, refList, err = osrv.client.Read([]byte(req.Key))
	}

	if err != nil {
		return nil, err
	}

	return &pb.ReadReply{
		Value:         value,
		ReferenceList: refList,
	}, nil
}

func (osrv *objectSrv) ReadFile(ctx context.Context, req *pb.ReadFileRequest) (*pb.ReadFileReply, error) {
	// check input
	if len(req.Key) == 0 {
		return nil, errNilKey
	}

	if len(req.FilePath) == 0 {
		return nil, errNilFilePath
	}

	// open the given file path
	f, err := os.Create(req.FilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	refList, err := osrv.client.ReadF(req.Key, f)

	if err != nil {
		return nil, err
	}

	return &pb.ReadFileReply{
		ReferenceList: refList,
	}, nil
}

func (osrv *objectSrv) ReadStream(req *pb.ReadStreamRequest, stream pb.ObjectService_ReadStreamServer) error {
	if len(req.Key) == 0 {
		return errNilKey
	}
	writer := newReadStreamWriter(stream)

	refList, err := osrv.client.ReadF(req.Key, writer)
	if err != nil {
		return err
	}

	return stream.Send(&pb.ReadStreamReply{
		ReferenceList: refList,
	})
}

func (osrv *objectSrv) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteReply, error) {
	// check input
	if len(req.Key) == 0 && req.Meta == nil {
		return nil, errNilKeyMeta
	}

	var err error

	if req.Meta != nil {
		err = osrv.client.DeleteWithMeta(pbMetaToStorMeta(req.Meta))
	} else {
		err = osrv.client.Delete(req.Key)
	}
	if err != nil {
		return nil, err
	}
	return &pb.DeleteReply{}, nil
}

func (osrv *objectSrv) Walk(req *pb.WalkRequest, stream pb.ObjectService_WalkServer) error {
	if len(req.StartKey) == 0 {
		return errNilKey
	}

	ch := osrv.client.Walk(req.StartKey, req.FromEpoch, req.ToEpoch)

	for wr := range ch {
		if wr.Error != nil {
			return wr.Error
		}

		stream.Send(&pb.WalkReply{
			Key:           wr.Key,
			Meta:          storMetaToPbMeta(wr.Meta),
			Value:         wr.Data,
			ReferenceList: wr.RefList,
		})
	}
	return nil
}

func (osrv *objectSrv) AppendReferenceList(ctx context.Context, req *pb.AppendReferenceListRequest) (*pb.AppendReferenceListReply, error) {
	err := checkAppendReferenceListReq(req)
	if err != nil {
		return nil, err
	}

	if req.Meta != nil {
		err = osrv.client.AppendToReferenceListWithMeta(pbMetaToStorMeta(req.Meta), req.ReferenceList)
	} else {
		err = osrv.client.AppendToReferenceList(req.Key, req.ReferenceList)
	}

	if err != nil {
		return nil, err
	}

	return &pb.AppendReferenceListReply{}, nil
}

func (osrv *objectSrv) RemoveReferenceList(ctx context.Context, req *pb.RemoveReferenceListRequest) (*pb.RemoveReferenceListReply, error) {
	err := checkRemoveReferenceListReq(req)
	if err != nil {
		return nil, err
	}

	if req.Meta != nil {
		err = osrv.client.DeleteFromReferenceListWithMeta(pbMetaToStorMeta(req.Meta), req.ReferenceList)
	} else {
		err = osrv.client.DeleteFromReferenceList(req.Key, req.ReferenceList)
	}

	if err != nil {
		return nil, err
	}

	return &pb.RemoveReferenceListReply{}, nil
}

func (osrv *objectSrv) Check(ctx context.Context, req *pb.CheckRequest) (*pb.CheckReply, error) {
	if req.Key == nil {
		return nil, errNilKey
	}

	status, err := osrv.client.Check(req.Key)
	if err != nil {
		return nil, err
	}

	pbStatus, err := storCheckStatusToPb(status)
	if err != nil {
		return nil, err
	}

	return &pb.CheckReply{
		Status: pbStatus,
	}, nil
}

func (osrv *objectSrv) Repair(ctx context.Context, req *pb.RepairRequest) (*pb.RepairReply, error) {
	if req.Key == nil {
		return nil, errNilKey
	}

	if err := osrv.client.Repair(req.Key); err != nil {
		return nil, err
	}

	return &pb.RepairReply{}, nil
}

func storCheckStatusToPb(cs storage.ObjectCheckStatus) (pb.CheckReply_Status, error) {
	switch cs {
	case storage.ObjectCheckStatusValid:
		return pb.CheckReplyStatusValid, nil
	case storage.ObjectCheckStatusOptimal:
		return pb.CheckReplyStatusOptimal, nil
	case storage.ObjectCheckStatusInvalid:
		return pb.CheckReplyStatusInvalid, nil
	}
	return pb.CheckReply_Status(0), errInvalidCheckReply
}

func checkAppendReferenceListReq(req *pb.AppendReferenceListRequest) error {
	if req.Key == nil && req.Meta == nil {
		return errNilKeyMeta
	}
	if len(req.ReferenceList) == 0 {
		return errEmptyRefList
	}
	return nil
}

func checkRemoveReferenceListReq(req *pb.RemoveReferenceListRequest) error {
	if req.Key == nil && req.Meta == nil {
		return errNilKeyMeta
	}
	if len(req.ReferenceList) == 0 {
		return errEmptyRefList
	}
	return nil
}

// convert protobuf chunk to 0-stor native chunk data type
func pbChunksToStorChunks(pbChunks []*pb.Chunk) []*metastor.Chunk {
	if len(pbChunks) == 0 {
		return nil
	}

	chunks := make([]*metastor.Chunk, 0, len(pbChunks))

	for _, c := range pbChunks {
		chunks = append(chunks, &metastor.Chunk{
			Size:   c.Size_,
			Key:    []byte(c.Key),
			Shards: c.Shards,
		})
	}

	return chunks
}

// convert from protobuf meta to 0-stor native meta data type
func pbMetaToStorMeta(pbMeta *pb.Meta) *metastor.Data {
	if pbMeta == nil {
		return nil
	}

	return &metastor.Data{
		Epoch:    pbMeta.Epoch,
		Key:      []byte(pbMeta.Key),
		Chunks:   pbChunksToStorChunks(pbMeta.Chunks),
		Previous: []byte(pbMeta.Previous),
		Next:     []byte(pbMeta.Next),
	}
}

// convert from 0-stor chunk to protobuf chunk data type
func storChunksToPbChunks(chunks []*metastor.Chunk) []*pb.Chunk {
	if len(chunks) == 0 {
		return nil
	}

	pbChunks := make([]*pb.Chunk, 0, len(chunks))

	for _, c := range chunks {
		pbChunks = append(pbChunks, &pb.Chunk{
			Size_:  c.Size,
			Key:    c.Key,
			Shards: c.Shards,
		})
	}

	return pbChunks
}

// convert from 0-stor native meta to protobuf meta data type
func storMetaToPbMeta(storMeta *metastor.Data) *pb.Meta {
	if storMeta == nil {
		return nil
	}
	return &pb.Meta{
		Epoch:    storMeta.Epoch,
		Key:      storMeta.Key,
		Chunks:   storChunksToPbChunks(storMeta.Chunks),
		Previous: storMeta.Previous,
		Next:     storMeta.Next,
	}
}

// writeStreamReader is io.Reader implementation
// that is needed by WriteStream API
type writeStreamReader struct {
	stream pb.ObjectService_WriteStreamServer
	buff   []byte
}

func newWriteStreamReader(stream pb.ObjectService_WriteStreamServer) (*writeStreamReader, *pb.WriteStreamRequest, error) {
	req, err := stream.Recv()
	if err != nil {
		return nil, nil, err
	}
	return &writeStreamReader{
		stream: stream,
		buff:   req.Value,
	}, req, nil
}

// Read implements io.Reader.Read interface
func (sr *writeStreamReader) Read(dest []byte) (int, error) {
	wantLen := len(dest)
	if len(sr.buff) >= wantLen {
		return sr.getValFromBuf(dest, wantLen), nil
	}

	for len(sr.buff) < wantLen {
		req, err := sr.stream.Recv()
		if err != nil {
			return sr.getValFromBuf(dest, wantLen), err
		}
		sr.buff = append(sr.buff, req.Value...)
	}

	return sr.getValFromBuf(dest, wantLen), nil
}

// read value from our internal buffer
func (sr *writeStreamReader) getValFromBuf(dest []byte, wantLen int) int {
	if len(sr.buff) == 0 {
		return 0
	}

	readLen := wantLen
	if len(sr.buff) < wantLen {
		readLen = len(sr.buff)
	}

	copy(dest, sr.buff[:readLen])
	sr.buff = sr.buff[readLen:]
	return readLen
}

// readStreamWriter is io.Writer implementations
// that is needed by ReadStream API
type readStreamWriter struct {
	stream pb.ObjectService_ReadStreamServer
}

func newReadStreamWriter(stream pb.ObjectService_ReadStreamServer) *readStreamWriter {
	return &readStreamWriter{
		stream: stream,
	}
}

func (rsw *readStreamWriter) Write(p []byte) (int, error) {
	err := rsw.stream.Send(&pb.ReadStreamReply{
		Value: p,
	})
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
