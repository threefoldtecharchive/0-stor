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


class Namespace:
    def __init__(self, channel):
        self._stub = stubs.NamespaceServiceStub(channel)

    def create(self, namespace):
        return self._stub.CreateNamespace(
            model.CreateNamespaceRequest(namespace=namespace)
        )

    def delete(self, namespace):
        return self._stub.DeleteNamespace(
            model.DeleteNamespaceRequest(namespace=namespace)
        )

    def get_permission(self, namespace, user):
        return self._stub.GetPermission(
            model.GetPermissionRequest(namespace=namespace, userID=user)
        )

    def set_permission(self, namespace, user, admin=False, read=False, write=False, delete=False):
        perm = model.Permission(admin=admin, read=read, write=write, delete=delete)
        return self._stub.SetPermission(
            model.SetPermissionRequest(namespace=namespace, userID=user, permission=perm)
        )
