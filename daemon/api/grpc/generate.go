package grpc

//go:generate protoc -I=. -I=../../../vendor -I=../../../vendor/github.com/gogo/protobuf/protobuf --gogoslick_out=plugins=grpc:. schema/daemon.proto
