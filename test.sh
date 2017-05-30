#!/bin/bash
set -e
pushd store
    echo "Build store"
    go get ./...
    go test ./...
popd
echo "Generate docs"
pushd specs/raml
    raml2html sdstor.raml > sdstor.html
popd
echo "Install go-raml"
pushd $GOPATH/src/github.com/Jumpscale/go-raml
    bash install.sh
popd
echo "Validate raml server generation"
go-raml server -l go --api-file-per-method --dir servertmp --ramlfile specs/raml/sdstor.raml --lib-root-urls https://raw.githubusercontent.com/Jumpscale-Cockpit/raml-definitions/master/
echo "Validate raml client generation"
go-raml client -l python --ramlfile specs/raml/sdstor.raml --dir clienttmp
