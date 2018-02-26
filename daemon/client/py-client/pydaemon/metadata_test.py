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

import unittest
from unittest import mock

from .generated import daemon_pb2_grpc as stubs
from .generated import daemon_pb2 as model

from .metadata import Metadata


class TestMetadataClient(unittest.TestCase):
    def setUp(self):
        with mock.patch.object(stubs, 'MetadataServiceStub') as m:
            m.side_effect = mock.MagicMock()
            self.client = Metadata(None)

    def test_created(self):
        self.assertIsNotNone(self.client)
        self.assertIsInstance(self.client._stub, mock.MagicMock)

    def test_set(self):
        key, total_size, chunks, creation_epoch, last_write_epoch =\
            b'key', mock.MagicMock(), mock.MagicMock(), mock.MagicMock(), mock.MagicMock()

        self.client.set(key, total_size, chunks, creation_epoch, last_write_epoch)

        self.client._stub.SetMetadata.assert_called_once_with(
            model.SetMetadataRequest(
                metadata=model.Metadata(
                    key=key,
                    creationEpoch=creation_epoch,
                    lastWriteEpoch=last_write_epoch,
                    totalSize=total_size,
                    chunks=chunks,
                )
            )
        )

    def test_get(self):
        key = b'key'

        obj = mock.MagicMock()

        self.client._stub.GetMetadata.return_value = obj

        result = self.client.get(key)
        self.client._stub.GetMetadata.assert_called_once_with(
            model.GetMetadataRequest(
                key=key,
            )
        )

        self.assertEqual(obj.metadata, result)

    def test_delete(self):
        key = b'key'

        self.client.delete(key)
        self.client._stub.DeleteMetadata.assert_called_once_with(
            model.DeleteMetadataRequest(
                key=key,
            )
        )

    def test_list_keys(self):
        self.client.list_keys()

        self.client._stub.ListKeys.assert_called_once_with(
            model.ListMetadataKeysRequest()
        )
