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

// server-side error
var (
	ErrGRPCNilKey              = grpc.Errorf(codes.InvalidArgument, "zstordb: key is not provided")
	ErrGRPCNilData             = grpc.Errorf(codes.InvalidArgument, "zstordb: data is not provided")
	ErrGRPCKeyNotFound         = grpc.Errorf(codes.NotFound, "zstordb: key is no found")
	ErrGRPCDatabase            = grpc.Errorf(codes.Internal, "zstordb: database operation failed")
	ErrGRPCObjectDataCorrupted = grpc.Errorf(codes.DataLoss, "zstordb: object data is corrupted")
	ErrGRPCNilLabel            = grpc.Errorf(codes.Unauthenticated, "zstordb: no label given")
	ErrGRPCNilToken            = grpc.Errorf(codes.Unauthenticated, "zstordb: no JWT token given")
	ErrGRPCUnimplemented       = grpc.Errorf(codes.Unimplemented, "zstordb: method support not implemented")
	ErrGRPCPermissionDenied    = grpc.Errorf(codes.PermissionDenied, "zstordb: JWT token does not permit requested action")
)

// string to server error mapping
var errStringToError = map[string]error{
	grpc.ErrorDesc(ErrGRPCNilKey):              ErrGRPCNilKey,
	grpc.ErrorDesc(ErrGRPCNilData):             ErrGRPCNilData,
	grpc.ErrorDesc(ErrGRPCKeyNotFound):         ErrGRPCKeyNotFound,
	grpc.ErrorDesc(ErrGRPCDatabase):            ErrGRPCDatabase,
	grpc.ErrorDesc(ErrGRPCObjectDataCorrupted): ErrGRPCObjectDataCorrupted,
	grpc.ErrorDesc(ErrGRPCNilLabel):            ErrGRPCNilLabel,
	grpc.ErrorDesc(ErrGRPCNilToken):            ErrGRPCNilToken,
	grpc.ErrorDesc(ErrGRPCUnimplemented):       ErrGRPCUnimplemented,
	grpc.ErrorDesc(ErrGRPCPermissionDenied):    ErrGRPCPermissionDenied,
}

// client-side error
var (
	ErrNilKey              = Error(ErrGRPCNilKey)
	ErrNilData             = Error(ErrGRPCNilData)
	ErrKeyNotFound         = Error(ErrGRPCKeyNotFound)
	ErrDatabase            = Error(ErrGRPCDatabase)
	ErrObjectDataCorrupted = Error(ErrGRPCObjectDataCorrupted)
	ErrNilLabel            = Error(ErrGRPCNilLabel)
	ErrNilToken            = Error(ErrGRPCNilToken)
	ErrUnimplemented       = Error(ErrGRPCUnimplemented)
	ErrPermissionDenied    = Error(ErrGRPCPermissionDenied)
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
