#!/bin/bash
set -ev

case $OSTYPE in
    darwin*) PLATFORM_ARCH="osx-x86_64" ;;
    *) PLATFORM_ARCH="linux-x86_64"     ;;
esac

VERSION="3.4.0"
PROTOC_DIR="protoc-$VERSION-$PLATFORM_ARCH"
PROTOC_ZIP="$PROTOC_DIR.zip"

# download and unzip binaries
curl -L -O "https://github.com/google/protobuf/releases/download/v$VERSION/$PROTOC_ZIP" ||\
    (echo "couldn't download $PROTOC_ZIP" && exit 1)
unzip "$PROTOC_ZIP" -d "$PROTOC_DIR" ||\
    (echo "couldn't unzip $PROTOC_ZIP" && exit 1)
rm -rf "$PROTOC_ZIP" ||\
    (echo "couldn't remove $PROTOC_ZIP" && exit 1)

# install binaries
sudo mv "$PROTOC_DIR/bin/protoc" "/usr/local/bin/protoc" ||\
    (echo "couldn't install $PROTOC_DIR/bin/protoc" && exit 1)

# remove unzipped and non-moved content
rm -rf "$PROTOC_DIR" || (echo "couldn't cleanup $PROTOC_DIR" && exit 1)

export GO111MODULE=off
go get -v -u github.com/golang/protobuf/protoc-gen-go
go get -v -u github.com/gogo/protobuf/protoc-gen-gofast
go install -v github.com/gogo/protobuf/protoc-gen-gogoslick
