//go:generate  go-bindata -debug -o routes/bindata.go -pkg routes -prefix ../specs/raml/ ../specs/raml/sdstor.html
//go:generate protoc -I specs/protobuf  specs/protobuf/store.proto --go_out=plugins=grpc:grpc_store

package main
