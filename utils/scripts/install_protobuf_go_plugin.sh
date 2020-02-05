#!/bin/bash

# install the vendored protobuf Go plugin
# (required by protoc exe for generating Go code)
PLUGIN="github.com/gogo/protobuf/protoc-gen-gogoslick"
echo "installing protoc gogo slick codegen plugin..."
env GO111MODULE=on go install -v "$PLUGIN"
