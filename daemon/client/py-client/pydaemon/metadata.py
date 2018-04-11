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


class Metadata:
    def __init__(self, channel):
        self._stub = stubs.MetadataServiceStub(channel)

    def set(self, key, total_size, chunks, creation_epoch=0, last_write_epoch=0):
        '''
        Sets metadata object give, a key, total size, chunks (as returned from the client.data stub)

        :param key: key (bytes)
        :param total_size: total size of all chunks
        :param creation_epoch: defines the time this data was initially created,
                               in the Unix epoch format, in nano seconds.
        :param last_write_epoch: defines the time this data was last modified (e.g. repaired),
                                 in the Unix epoch format, in nano seconds.
        '''

        return self._stub.SetMetadata(
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

    def get(self, key):
        '''
        Gets a metadata object

        :param key: key (bytes)
        :return: metadata
        '''
        return self._stub.GetMetadata(
            model.GetMetadataRequest(key=key)
        ).metadata

    def delete(self, key):
        '''
        Deletes metadata object

        :param key: key (bytes)
        '''
        return self._stub.DeleteMetadata(
            model.DeleteMetadataRequest(key=key)
        )

    def list_keys(self):
        '''
        List all keys in the namespace
        '''
        return self._stub.ListKeys(model.ListMetadataKeysRequest())
