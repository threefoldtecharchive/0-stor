#!/bin/bash

# install the vendored protobuf Go plugin
# (required by protoc exe for generating Go code)
VENDOR_DIR="github.com/zero-os/0-stor/vendor"
PLUGIN_DIR="github.com/gogo/protobuf/protoc-gen-gogoslick"
echo "installing vendored protoc gogo slick codegen plugin..."
go install -v "$VENDOR_DIR/$PLUGIN_DIR"
