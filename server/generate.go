//go:generate protoc -I schema schema/ztor.proto --go_out=plugins=grpc,import_path=schema:schema

package server
