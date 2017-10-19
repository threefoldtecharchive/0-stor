#!/bin/bash
set -ex

apt-get update
apt-get install wget make -y

# install go
wget https://storage.googleapis.com/golang/go1.9.linux-amd64.tar.gz -O /tmp/go1.9.linux-amd64.tar.gz
tar -C /usr/local -xzf /tmp/go1.9.linux-amd64.tar.gz
export GOPATH=/gopath
mkdir -p $GOPATH
export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin
STOR0=$GOPATH/src/github.com/zero-os/0-stor/
mkdir -p $GOPATH/src/github.com/zero-os/

# move code into GOPATH
mv /0-stor $STOR0

pushd $STOR0
make
popd

mkdir -p /tmp/archives/
tar -czf "/tmp/archives/0-stor.tar.gz" -C $STOR0/ bin
