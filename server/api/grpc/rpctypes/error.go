package rpctypes

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// server-side error
var (
	ErrGRPCNilKey                 = grpc.Errorf(codes.InvalidArgument, "zstordb: key is not provided")
	ErrGRPCNilData                = grpc.Errorf(codes.InvalidArgument, "zstordb: data is not provided")
	ErrGRPCNilRefList             = grpc.Errorf(codes.InvalidArgument, "zstordb: reference list is not provided")
	ErrGRPCKeyNotFound            = grpc.Errorf(codes.NotFound, "zstordb: key is no found")
	ErrGRPCDatabase               = grpc.Errorf(codes.Internal, "zstordb: database operation failed")
	ErrGRPCObjectDataCorrupted    = grpc.Errorf(codes.DataLoss, "zstordb: object data is corrupted")
	ErrGRPCObjectRefListCorrupted = grpc.Errorf(codes.DataLoss, "zstordb: object reflist is corrupted")
	ErrGRPCNilLabel               = grpc.Errorf(codes.Unauthenticated, "zstordb: no label given")
	ErrGRPCNilToken               = grpc.Errorf(codes.Unauthenticated, "zstordb: no JWT token given")
	ErrGRPCUnimplemented          = grpc.Errorf(codes.Unimplemented, "zstordb: method support not implemented")
	ErrGRPCPermissionDenied       = grpc.Errorf(codes.PermissionDenied, "zstordb: JWT token does not permit requested action")
)

// string to server error mapping
var errStringToError = map[string]error{
	grpc.ErrorDesc(ErrGRPCNilKey):                 ErrGRPCNilKey,
	grpc.ErrorDesc(ErrGRPCNilData):                ErrGRPCNilData,
	grpc.ErrorDesc(ErrGRPCNilRefList):             ErrGRPCNilRefList,
	grpc.ErrorDesc(ErrGRPCKeyNotFound):            ErrGRPCKeyNotFound,
	grpc.ErrorDesc(ErrGRPCDatabase):               ErrGRPCDatabase,
	grpc.ErrorDesc(ErrGRPCObjectDataCorrupted):    ErrGRPCObjectDataCorrupted,
	grpc.ErrorDesc(ErrGRPCObjectRefListCorrupted): ErrGRPCObjectRefListCorrupted,
	grpc.ErrorDesc(ErrGRPCNilLabel):               ErrGRPCNilLabel,
	grpc.ErrorDesc(ErrGRPCNilToken):               ErrGRPCNilToken,
	grpc.ErrorDesc(ErrGRPCUnimplemented):          ErrGRPCUnimplemented,
	grpc.ErrorDesc(ErrGRPCPermissionDenied):       ErrGRPCPermissionDenied,
}

// client-side error
var (
	ErrNilKey                 = Error(ErrGRPCNilKey)
	ErrNilData                = Error(ErrGRPCNilData)
	ErrNilRefList             = Error(ErrGRPCNilRefList)
	ErrKeyNotFound            = Error(ErrGRPCKeyNotFound)
	ErrDatabase               = Error(ErrGRPCDatabase)
	ErrObjectDataCorrupted    = Error(ErrGRPCObjectDataCorrupted)
	ErrObjectRefListCorrupted = Error(ErrGRPCObjectRefListCorrupted)
	ErrNilLabel               = Error(ErrGRPCNilLabel)
	ErrNilToken               = Error(ErrGRPCNilToken)
	ErrUnimplemented          = Error(ErrGRPCUnimplemented)
	ErrPermissionDenied       = Error(ErrGRPCPermissionDenied)
)

// ZStorError defines gRPC server errors.
// (https://github.com/grpc/grpc-go/blob/master/rpc_util.go#L319-L323)
type ZStorError struct {
	code codes.Code
	desc string
}

// Code returns grpc/codes.Code.
func (e ZStorError) Code() codes.Code {
	return e.code
}

func (e ZStorError) Error() string {
	return e.desc
}

// Error transforms a GRPC error into a ZStor Error,
// so we can have nice client-side errors.
func Error(err error) error {
	if err == nil {
		return nil
	}
	verr, ok := errStringToError[grpc.ErrorDesc(err)]
	if !ok { // not gRPC error
		return err
	}
	return ZStorError{code: grpc.Code(verr), desc: grpc.ErrorDesc(verr)}
}
