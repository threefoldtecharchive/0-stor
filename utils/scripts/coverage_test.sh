#!/usr/bin/env bash

set -e
set -x

echo "" > coverage.txt

test_package_coverage() {
for d in $(go list "$1"); do
    go test -coverprofile=profile.out -covermode=atomic "$d"
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done
}

test_package_coverage "./server/..."
test_package_coverage "./client/..."
test_package_coverage "./daemon/..."
test_package_coverage "./cmd"
