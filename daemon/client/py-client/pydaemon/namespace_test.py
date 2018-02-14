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

from .namespace import Namespace


class TestNamespaceClient(unittest.TestCase):
    def setUp(self):
        with mock.patch.object(stubs, 'NamespaceServiceStub') as m:
            m.side_effect = mock.MagicMock()
            self.client = Namespace(None)

    def test_created(self):
        self.assertIsNotNone(self.client)
        self.assertIsInstance(self.client._stub, mock.MagicMock)

    def test_list(self):
        self.client.list()

        self.client._stub.ListNamespaces.assert_called_once_with(
            model.ListNamespacesRequest()
        )

    def test_create(self):
        namespace = 'namespace'

        self.client.create(namespace)

        self.client._stub.CreateNamespace.assert_called_once_with(
            model.CreateNamespaceRequest(
                namespace=namespace
            )
        )

    def test_delete(self):
        namespace = 'namespace'

        self.client.delete(namespace)

        self.client._stub.DeleteNamespace.assert_called_once_with(
            model.DeleteNamespaceRequest(
                namespace=namespace
            )
        )

    def test_get_permission(self):
        namespace = 'namespace'
        user = 'test'

        self.client.get_permission(namespace, user)

        self.client._stub.GetPermission.assert_called_once_with(
            model.GetPermissionRequest(
                namespace=namespace,
                userID=user
            )
        )

    def test_set_permission(self):
        namespace, user, admin, read, write, delete =\
            'namespace', 'test', True, False, True, False

        self.client.set_permission(namespace, user, admin, read, write, delete)

        self.client._stub.SetPermission.assert_called_once_with(
            model.SetPermissionRequest(
                namespace=namespace,
                userID=user,
                permission=model.Permission(
                    admin=admin, read=read, write=write, delete=delete,
                )
            )
        )
