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

import functools
import unittest
from unittest import mock

from .generated import daemon_pb2_grpc as stubs
from .generated import daemon_pb2 as model

from .data import Data


class TestDataClient(unittest.TestCase):
    def setUp(self):
        with mock.patch.object(stubs, 'DataServiceStub') as m:
            m.side_effect = mock.MagicMock()
            self.client = Data(None)

    def test_created(self):
        self.assertIsNotNone(self.client)
        self.assertIsInstance(self.client._stub, mock.MagicMock)

    def test_write(self):
        obj = mock.MagicMock()
        self.client._stub.Write.return_value = obj

        data = b'value'

        result = self.client.write(data)

        self.client._stub.Write.assert_called_once_with(
            model.DataWriteRequest(data=data)
        )
        self.assertEqual(obj.chunks, result)

    def test_read(self):
        obj = mock.MagicMock()
        obj.data = b'data'
        self.client._stub.Read.return_value = obj

        chunks = mock.MagicMock()

        result = self.client.read(chunks)

        self.client._stub.Read.assert_called_once_with(
            model.DataReadRequest(chunks=chunks)
        )
        self.assertEqual(obj.data, result)

    def test_write_file(self):
        obj = mock.MagicMock()
        self.client._stub.WriteFile.return_value = obj

        file_path = b'file path'

        result = self.client.write_file(file_path)

        self.client._stub.WriteFile.assert_called_once_with(
            model.DataWriteFileRequest(filePath=file_path)
        )

        self.assertEqual(obj.chunks, result)

    def test_read_file(self):
        chunks = mock.MagicMock()
        file_path = b'file path'
        mode = Data.FileMode.Append | Data.FileMode.Exclusive
        sync_io = True

        self.client.read_file(chunks, file_path, mode=mode, sync_io=sync_io)

        self.client._stub.ReadFile.assert_called_once_with(
            model.DataReadFileRequest(chunks=chunks, filePath=file_path, fileMode=mode, synchronousIO=sync_io)
        )

    def test_write_stream(self):
        bs = 1000
        input = mock.MagicMock()

        chunks = 10
        state = {
            # read will return 3 chunks
            'chunk': chunks
        }

        def read(state, bs):
            if state['chunk'] > 0:
                state['chunk'] -= 1
                return b'Chunk: %d' % (chunks - state['chunk'])
            return ''

        input.read.side_effect = functools.partial(read, state)

        def effect(iter):
            i = 1
            for obj in iter:
                self.assertEqual(obj, model.DataWriteStreamRequest(
                        dataChunk=b'Chunk: %d' % i
                    )
                )
                i += 1
            return mock.MagicMock()

        self.client._stub.WriteStream.side_effect = effect
        self.client.write_stream(input, block_size=bs)

        calls = []
        for i in range(chunks + 1):
            calls.append(mock.call(bs))

        input.read.assert_has_calls(calls)
        self.assertEqual(input.read.call_count, chunks + 1)

    def test_read_stream(self):
        chunks = mock.MagicMock()
        bs = 1000
        output = mock.MagicMock()

        chunks_no = 10
        obj = []
        calls = []
        for i in range(chunks_no):
            m = mock.MagicMock()
            m.dataChunk = 'data chunk: %d' % i
            calls.append(mock.call(m.dataChunk))
            obj.append(m)

        self.client._stub.ReadStream.return_value = obj

        self.client.read_stream(chunks, output, bs)
        self.client._stub.ReadStream.assert_called_once_with(
            model.DataReadStreamRequest(chunks=chunks, chunkSize=bs)
        )

        self.assertEqual(output.write.call_count, chunks_no)
        output.write.assert_has_calls(calls)

    def test_delete(self):
        chunks = mock.MagicMock()

        self.client.delete(chunks)
        self.client._stub.Delete.assert_called_once_with(
            model.DataDeleteRequest(chunks=chunks)
        )

    def test_check(self):

        self.client._stub.Check.return_value.status = True

        chunks = mock.MagicMock()
        fast = True

        result = self.client.check(chunks, fast)
        self.client._stub.Check.assert_called_once_with(
            model.DataCheckRequest(chunks=chunks, fast=fast)
        )

        self.assertTrue(result)

    def test_repair(self):
        obj = mock.MagicMock()
        self.client._stub.Repair.return_value = obj

        chunks = mock.MagicMock()

        result = self.client.repair(chunks)
        self.client._stub.Repair.assert_called_once_with(
            model.DataRepairRequest(chunks=chunks)
        )

        self.assertEqual(obj.chunks, result)
