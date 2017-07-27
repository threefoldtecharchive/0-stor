//go:generate protoc -I ../specs/protobuf/  ../specs/protobuf/store.proto --go_out=plugins=grpc:store

package grpc
