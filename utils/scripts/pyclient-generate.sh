#!env sh
set -ex

MODULE=pydaemon

ROOT=$(git rev-parse --show-toplevel)
CLIENT=${ROOT}/daemon/client/${MODULE}
GENERATED=${CLIENT}/generated

mkdir -p ${GENERATED}
rm -rf ${GENERATED}/*.py

python -m grpc_tools.protoc -I${ROOT}/daemon/api/grpc/schema --python_out=${GENERATED} --grpc_python_out=${GENERATED} daemon.proto
touch ${GENERATED}/__init__.py
2to3 -w ${GENERATED}
rm -rf ${GENERATED}/*.bak
