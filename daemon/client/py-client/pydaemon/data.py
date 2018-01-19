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


class Data:
    class FileMode:
        Truncate = model.FileModeTruncate
        Append = model.FileModeAppend
        Exclusive = model.FileModeExclusive

    def __init__(self, channel):
        self._stub = stubs.DataServiceStub(channel)

    def write(self, data):
        '''
        Write date to 0-store

        :param data: data (bytes)

        :return: chunks
        '''
        return self._stub.Write(
            model.DataWriteRequest(data=data)
        ).chunks

    def read(self, chunks):
        '''
        Read data from 0-stor

        :param chunk: chunks as returned by a write() call

        :return: data (bytes)
        '''
        return self._stub.Read(
            model.DataReadRequest(chunks=chunks)
        ).data

    def write_file(self, file_path):
        '''
        upload file to 0-stor

        :param file_path: path to local file to upload

        :return: file chunks
        '''

        return self._stub.WriteFile(
            model.DataWriteFileRequest(filePath=file_path)
        ).chunks

    def read_file(self, chunks, file_path, mode=FileMode.Truncate, sync_io=False):
        '''
        :param chunks: file chunks as returned from write_file
        :param file_path: local file path to download to
        :param mode: 0 = truncate, 1 = append, 2 = exclusive
        :param sync_io: use the O_SYNC on the file, forcing all write operation to be writen to the
                        underlying hardware before returning.
        '''

        return self._stub.ReadFile(
            model.DataReadFileRequest(chunks=chunks, filePath=file_path, fileMode=mode, synchronousIO=sync_io)
        )

    def write_stream(self, input, block_size=4096):
        '''
        Upload data from a file like object (input)

        :param input: file like object (implements a read function which return 'bytes')
        :param block_size: block size used to call input.read(block_size)

        :note: if input is an open file, make sure it's open in binary mode
        :return: metadata object
        '''
        def stream():
            while True:
                chunk = input.read(block_size)
                if len(chunk) == 0:
                    break
                yield model.DataWriteStreamRequest(
                    dataChunk=chunk
                )

        return self._stub.WriteStream(stream()).chunks

    def read_stream(self, chunks, output, chunk_size=4096):
        '''
        Download data to a file like object (output)

        :param chunks: chunks as returned by write_stream
        :param chunk_size: read chunk size in bytes

        :param output: file like object (implements a write function which takey 'bytes')
        '''

        response = self._stub.ReadStream(
            model.DataReadStreamRequest(chunks=chunks, chunkSize=chunk_size)
        )

        for data in response:
            output.write(data.dataChunk)

    def delete(self, chunks):
        '''
        Delete a data with chunks

        :param chunks: chunks
        '''

        return self._stub.Delete(
            model.DataDeleteRequest(chunks=chunks)
        )

    def check(self, chunks, fast=True):
        '''
        Checks data state with key

        :param chunks: data chunks
        :param fast: fast check (bool)

        :return: check state (0 = invalid, 1 = valie, 2 = optimal)
        '''
        return self._stub.Check(
            model.DataCheckRequest(chunks=chunks, fast=fast)
        ).status

    def repair(self, chunks):
        '''
        Reparis a file

        :param chunks: chunks
        '''

        return self._stub.Repair(
            model.DataRepairRequest(chunks=chunks)
        ).chunks
