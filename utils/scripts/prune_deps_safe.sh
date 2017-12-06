#!/bin/bash

# add all stuff to stage, so we don't lose stuff that was updated
git add -A

# prune deps using golang/dep
dep prune -v

# check which files/dirs are deleted, and ensure it's not one of ours
git diff --no-renames --name-only --diff-filter=D |
    grep -E 'gogoproto/gogo.proto|gogo/protobuf/plugin|protoc-gen-gogoslick|gogo/protobuf/vanity|gogo/protobuf/protobuf|protoc-gen-gogo/grpc|protoc-gen-gogo/generator|protoc-gen-gogo/plugin' |
    xargs git checkout

# add all files that should be deleted, if there are any, to stage as well
git add -A
