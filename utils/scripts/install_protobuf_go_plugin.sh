#!/bin/bash

# install the vendored protobuf Go plugin
# (required by protoc exe for generating Go code)
VENDOR_DIR="github.com/zero-os/0-stor/vendor"
PLUGIN_DIR="github.com/golang/protobuf/protoc-gen-go"
echo "installing vendored protoc go codegen plugin..."
go install -v "$VENDOR_DIR/$PLUGIN_DIR"
