/*
 * Copyright (C) 2017-2018 GIG Technology NV and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rpctypes

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// (daemon) server-side error
var (
	ErrGRPCNilKey           = grpc.Errorf(codes.InvalidArgument, "daemon: key is not provided")
	ErrGRPCNilData          = grpc.Errorf(codes.InvalidArgument, "daemon: data is not provided")
	ErrGRPCNilFilePath      = grpc.Errorf(codes.InvalidArgument, "daemon: file path is not provided")
	ErrGRPCNilMetadata      = grpc.Errorf(codes.InvalidArgument, "daemon: metadata is not provided")
	ErrGRPCNilNamespace     = grpc.Errorf(codes.InvalidArgument, "daemon: namespace is not provided")
	ErrGRPCNilUserID        = grpc.Errorf(codes.InvalidArgument, "daemon: userID is not provided")
	ErrGRPCNilPermissions   = grpc.Errorf(codes.InvalidArgument, "daemon: permissions are not provided")
	ErrGRPCNilChunks        = grpc.Errorf(codes.InvalidArgument, "daemon: (meta) chunks are not provided")
	ErrGRPCInvalidChunkSize = grpc.Errorf(codes.InvalidArgument, "daemon: invalid chunksize (has to be 1 or higher)")
	ErrGRPCKeyNotFound      = grpc.Errorf(codes.NotFound, "daemon: key is no found")
	ErrGRPCDataNotRead      = grpc.Errorf(codes.NotFound, "daemon: no data could be read")
	ErrGRPCDataCorrupted    = grpc.Errorf(codes.DataLoss, "daemon: data is corrupted")
	ErrGRPCNotSupported     = grpc.Errorf(codes.Unimplemented, "daemon: method not supported")
	ErrGRPCInvalidFileMode  = grpc.Errorf(codes.Unimplemented, "daemon: file mode not supported")
	ErrGRPCNoLocalFS        = grpc.Errorf(codes.PermissionDenied, "daemon: local filesystem access not allowed")
	ErrGRPCPermissionDenied = grpc.Errorf(codes.PermissionDenied, "daemon: JWT token does not permit requested action")
)

// string to (daemon) server error mapping
var errStringToError = map[string]error{
	grpc.ErrorDesc(ErrGRPCNilKey):           ErrGRPCNilKey,
	grpc.ErrorDesc(ErrGRPCNilData):          ErrGRPCNilData,
	grpc.ErrorDesc(ErrGRPCNilFilePath):      ErrGRPCNilFilePath,
	grpc.ErrorDesc(ErrGRPCNilMetadata):      ErrGRPCNilMetadata,
	grpc.ErrorDesc(ErrGRPCNilNamespace):     ErrGRPCNilNamespace,
	grpc.ErrorDesc(ErrGRPCNilUserID):        ErrGRPCNilUserID,
	grpc.ErrorDesc(ErrGRPCNilPermissions):   ErrGRPCNilPermissions,
	grpc.ErrorDesc(ErrGRPCNilChunks):        ErrGRPCNilChunks,
	grpc.ErrorDesc(ErrGRPCInvalidChunkSize): ErrGRPCInvalidChunkSize,
	grpc.ErrorDesc(ErrGRPCKeyNotFound):      ErrGRPCKeyNotFound,
	grpc.ErrorDesc(ErrGRPCDataNotRead):      ErrGRPCDataNotRead,
	grpc.ErrorDesc(ErrGRPCDataCorrupted):    ErrGRPCDataCorrupted,
	grpc.ErrorDesc(ErrGRPCNotSupported):     ErrGRPCNotSupported,
	grpc.ErrorDesc(ErrGRPCInvalidFileMode):  ErrGRPCInvalidFileMode,
	grpc.ErrorDesc(ErrGRPCNoLocalFS):        ErrGRPCNoLocalFS,
	grpc.ErrorDesc(ErrGRPCPermissionDenied): ErrGRPCPermissionDenied,
}

// (daemon) client-side error
var (
	ErrNilKey           = Error(ErrGRPCNilKey)
	ErrNilData          = Error(ErrGRPCNilData)
	ErrNilFilePath      = Error(ErrGRPCNilFilePath)
	ErrNilMetadata      = Error(ErrGRPCNilMetadata)
	ErrNilNamespace     = Error(ErrGRPCNilNamespace)
	ErrNilUserID        = Error(ErrGRPCNilUserID)
	ErrNilPermissions   = Error(ErrGRPCNilPermissions)
	ErrNilChunks        = Error(ErrGRPCNilChunks)
	ErrInvalidChunkSize = Error(ErrGRPCInvalidChunkSize)
	ErrKeyNotFound      = Error(ErrGRPCKeyNotFound)
	ErrDataNotRead      = Error(ErrGRPCDataNotRead)
	ErrDataCorrupted    = Error(ErrGRPCDataCorrupted)
	ErrNotSupported     = Error(ErrGRPCNotSupported)
	ErrInvalidFileMode  = Error(ErrGRPCInvalidFileMode)
	ErrNoLocalFS        = Error(ErrGRPCNoLocalFS)
	ErrPermissionDenied = Error(ErrGRPCPermissionDenied)
)

// DaemonError defines gRPC server errors.
// (https://github.com/grpc/grpc-go/blob/master/rpc_util.go#L319-L323)
type DaemonError struct {
	code codes.Code
	desc string
}

// Code returns grpc/codes.Code.
func (e DaemonError) Code() codes.Code {
	return e.code
}

func (e DaemonError) Error() string {
	return e.desc
}

// Error transforms a GRPC error into a DaemonError Error,
// so we can have nice client-side errors.
func Error(err error) error {
	if err == nil {
		return nil
	}
	verr, ok := errStringToError[grpc.ErrorDesc(err)]
	if !ok { // not gRPC error
		return err
	}
	return DaemonError{code: grpc.Code(verr), desc: grpc.ErrorDesc(verr)}
}
