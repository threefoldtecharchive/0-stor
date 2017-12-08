package grpc

import (
	"io"

	"github.com/zero-os/0-stor/server"
	pb "github.com/zero-os/0-stor/server/api/grpc/schema"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type stubObjectService struct {
	key, data []byte
	refList   []string
	status    pb.ObjectStatus
	err       error
	streamErr error
}

// SetObject implements pb.ObjectService.SetObject
func (os *stubObjectService) SetObject(ctx context.Context, in *pb.SetObjectRequest, opts ...grpc.CallOption) (*pb.SetObjectResponse, error) {
	if os.err != nil {
		return nil, os.err
	}
	return &pb.SetObjectResponse{}, nil
}

// GetObject implements pb.ObjectService.GetObject
func (os *stubObjectService) GetObject(ctx context.Context, in *pb.GetObjectRequest, opts ...grpc.CallOption) (*pb.GetObjectResponse, error) {
	if os.err != nil {
		return nil, os.err
	}
	return &pb.GetObjectResponse{
		Data:          os.data,
		ReferenceList: os.refList,
	}, nil
}

// DeleteObject implements pb.ObjectService.DeleteObject
func (os *stubObjectService) DeleteObject(ctx context.Context, in *pb.DeleteObjectRequest, opts ...grpc.CallOption) (*pb.DeleteObjectResponse, error) {
	if os.err != nil {
		return nil, os.err
	}
	return &pb.DeleteObjectResponse{}, nil
}

// GetObjectStatus implements pb.ObjectService.GetObjectStatus
func (os *stubObjectService) GetObjectStatus(ctx context.Context, in *pb.GetObjectStatusRequest, opts ...grpc.CallOption) (*pb.GetObjectStatusResponse, error) {
	if os.err != nil {
		return nil, os.err
	}
	return &pb.GetObjectStatusResponse{
		Status: os.status,
	}, nil
}

// ListObjectKeys implements pb.ObjectService.ListObjectKeys
func (os *stubObjectService) ListObjectKeys(ctx context.Context, in *pb.ListObjectKeysRequest, opts ...grpc.CallOption) (pb.ObjectManager_ListObjectKeysClient, error) {
	if os.err != nil {
		return nil, os.err
	}
	return &stubListObjectKeysClient{
		ClientStream: nil,
		eof:          os.key != nil,
		key:          os.key,
		err:          os.streamErr,
	}, nil
}

type stubListObjectKeysClient struct {
	grpc.ClientStream
	key []byte
	err error
	eof bool
}

// Recv implements pb.ObjectManager_ListObjectKeysClient.Recv
func (stream *stubListObjectKeysClient) Recv() (*pb.ListObjectKeysResponse, error) {
	if stream.err != nil {
		return nil, stream.err
	}
	if stream.key == nil {
		if stream.eof {
			return nil, io.EOF
		}
		return &pb.ListObjectKeysResponse{}, nil
	}
	resp := &pb.ListObjectKeysResponse{Key: stream.key}
	stream.key = nil
	return resp, nil
}

// SetReferenceList implements pb.ObjectService.SetReferenceList
func (os *stubObjectService) SetReferenceList(ctx context.Context, in *pb.SetReferenceListRequest, opts ...grpc.CallOption) (*pb.SetReferenceListResponse, error) {
	if os.err != nil {
		return nil, os.err
	}
	return &pb.SetReferenceListResponse{}, nil
}

// GetReferenceList implements pb.ObjectService.GetReferenceList
func (os *stubObjectService) GetReferenceList(ctx context.Context, in *pb.GetReferenceListRequest, opts ...grpc.CallOption) (*pb.GetReferenceListResponse, error) {
	if os.err != nil {
		return nil, os.err
	}
	return &pb.GetReferenceListResponse{
		ReferenceList: os.refList,
	}, nil
}

// GetReferenceCount implements pb.ObjectService.GetReferenceCount
func (os *stubObjectService) GetReferenceCount(ctx context.Context, in *pb.GetReferenceCountRequest, opts ...grpc.CallOption) (*pb.GetReferenceCountResponse, error) {
	if os.err != nil {
		return nil, os.err
	}
	return &pb.GetReferenceCountResponse{
		Count: int64(len(os.refList)),
	}, nil
}

// AppendToReferenceList implements pb.ObjectService.AppendToReferenceList
func (os *stubObjectService) AppendToReferenceList(ctx context.Context, in *pb.AppendToReferenceListRequest, opts ...grpc.CallOption) (*pb.AppendToReferenceListResponse, error) {
	if os.err != nil {
		return nil, os.err
	}
	return &pb.AppendToReferenceListResponse{}, nil
}

// DeleteFromReferenceList implements pb.ObjectService.DeleteFromReferenceList
func (os *stubObjectService) DeleteFromReferenceList(ctx context.Context, in *pb.DeleteFromReferenceListRequest, opts ...grpc.CallOption) (*pb.DeleteFromReferenceListResponse, error) {
	if os.err != nil {
		return nil, os.err
	}
	refList := server.ReferenceList(os.refList)
	refList.RemoveReferences(server.ReferenceList(in.ReferenceList))
	os.refList = []string(refList)
	return &pb.DeleteFromReferenceListResponse{
		Count: int64(len(os.refList)),
	}, nil
}

// DeleteReferenceList implements pb.ObjectService.DeleteReferenceList
func (os *stubObjectService) DeleteReferenceList(ctx context.Context, in *pb.DeleteReferenceListRequest, opts ...grpc.CallOption) (*pb.DeleteReferenceListResponse, error) {
	if os.err != nil {
		return nil, os.err
	}
	return &pb.DeleteReferenceListResponse{}, nil
}

type stubNamespaceService struct {
	label                          string
	readRPH, writeRPH, nrOfObjects int64
	err                            error
}

// GetNamespace implements pb.NamespaceService.GetNamespace
func (ns *stubNamespaceService) GetNamespace(ctx context.Context, in *pb.GetNamespaceRequest, opts ...grpc.CallOption) (*pb.GetNamespaceResponse, error) {
	if ns.err != nil {
		return nil, ns.err
	}
	return &pb.GetNamespaceResponse{
		Label:               ns.label,
		ReadRequestPerHour:  ns.readRPH,
		WriteRequestPerHour: ns.writeRPH,
		NrObjects:           ns.nrOfObjects,
	}, nil
}
