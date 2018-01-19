# Copyright (C) 2017-2018 GIG Technology NV and Contributors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from .generated import daemon_pb2_grpc as stubs
from .generated import daemon_pb2 as model


class File:
    class FileMode:
        Truncate = model.FileModeTruncate
        Append = model.FileModeAppend
        Exclusive = model.FileModeExclusive

    def __init__(self, channel):
        self._stub = stubs.FileServiceStub(channel)

    def write(self, key, data):
        '''
        Write date to 0-store

        :param key: key (bytes)
        :param data: data (bytes)

        :return: metadata
        '''
        return self._stub.Write(
            model.WriteRequest(key=key, data=data)
        ).metadata

    def read(self, key):
        '''
        Read data from 0-stor

        :param key: key (bytes)

        :return: data (bytes)
        '''
        return self._stub.Read(
            model.ReadRequest(key=key)
        ).data

    def write_file(self, key, file_path):
        '''
        upload file to 0-stor

        :param key: file key (bytes)
        :param file_path: path to local file to upload

        '''
        return self._stub.WriteFile(
            model.WriteFileRequest(key=key, filePath=file_path)
        ).metadata

    def read_file(self, key, file_path, mode=FileMode.Truncate, sync_io=False):
        '''
        :param key: file key
        :param file_path: local file path to download to
        :param mode: 0 = truncate, 1 = append, 2 = exclusive
        :param sync_io: use the O_SYNC on the file, forcing all write operation to be writen to the
                        underlying hardware before returning.
        '''

        return self._stub.ReadFile(
            model.ReadFileRequest(key=key, filePath=file_path, fileMode=mode, synchronousIO=sync_io)
        )

    def write_stream(self, key, input, block_size=4096):
        '''
        Upload data from a file like object (input)

        :param key: key (bytes)
        :param input: file like object (implements a read function which return 'bytes')

        :note: if input is an open file, make sure it's open in binary mode
        :return: metadata object
        '''
        def stream():
            yield model.WriteStreamRequest(
                metadata=model.WriteStreamRequest.Metadata(key=key)
            )
            while True:
                chunk = input.read(block_size)
                if len(chunk) == 0:
                    break
                yield model.WriteStreamRequest(
                    data=model.WriteStreamRequest.Data(dataChunk=chunk)
                )

        return self._stub.WriteStream(stream()).metadata

    def read_stream(self, key, output, chunk_size=4096):
        '''
        Download data to a file like object (output)

        :param key: key (bytes)
        :param output: file like object (implements a write function which takey 'bytes')
        '''

        response = self._stub.ReadStream(
            model.ReadStreamRequest(key=key, chunkSize=chunk_size)
        )

        for data in response:
            output.write(data.dataChunk)

    def delete(self, key):
        '''
        Delete a file with key

        :param key: key (bytes)
        '''

        return self._stub.Delete(
            model.DeleteRequest(key=key)
        )

    def check(self, key, fast=True):
        '''
        Checks file state with key

        :param key: key (bytes)
        :param fast: fast check (bool)

        :return: check state (0 = invalid, 1 = valie, 2 = optimal)
        '''
        return self._stub.Check(
            model.CheckRequest(key=key, fast=fast)
        ).status

    def repair(self, key):
        '''
        Reparis a file

        :param key: key (bytes)
        '''

        return self._stub.Repair(
            model.RepairRequest(key=key)
        ).metadata
