#!/bin/bash

case $OSTYPE in
    darwin*) PLATFORM_ARCH="osx-x86_64" ;;
    *) PLATFORM_ARCH="linux-x86_64"     ;;
esac

VERSION="3.5.0"
PROTOC_DIR="protoc-$VERSION-$PLATFORM_ARCH"
PROTOC_ZIP="$PROTOC_DIR.zip"

# download and unzip binaries
curl -L -O "https://github.com/google/protobuf/releases/download/v$VERSION/$PROTOC_ZIP" ||\
    (echo "couldn't download $PROTOC_ZIP" && exit 1)
unzip "$PROTOC_ZIP" -d "$PROTOC_DIR" ||\
    (echo "couldn't unzip $PROTOC_ZIP" && exit 1)
rm -rf "$PROTOC_ZIP" ||\
    (echo "couldn't remove $PROTOC_ZIP" && exit 1)

# clean up old includes or make sure include namespace exists
if [ -d "/usr/local/include/google/protobuf" ]
then
	sudo rm -rf "/usr/local/include/google/protobuf" ||\
        (echo "couldn't delete /usr/local/include/google/protobuf" && exit 1)
else
    # make sure that at least the namespace exists
	sudo mkdir -p "/user/local/include/google" ||\
        (echo "couldn't create /user/local/include/google" && exit 1)
fi

# install binaries
sudo mv "$PROTOC_DIR/bin/protoc" "/usr/local/bin/protoc" ||\
    (echo "couldn't install $PROTOC_DIR/bin/protoc" && exit 1)
sudo mv "$PROTOC_DIR/include/google/protobuf" "/usr/local/include/google/protobuf" ||\
    (echo "couldn't copy $PROTOC_DIR/include/google/protobuf proto std files" && exit 1)

# remove unzipped and non-moved content
rm -rf "$PROTOC_DIR" || (echo "couldn't cleanup $PROTOC_DIR" && exit 1)

# install vendored go plugin
BASEDIR=$(dirname "$0")
"$BASEDIR"/install_protobuf_go_plugin.sh
