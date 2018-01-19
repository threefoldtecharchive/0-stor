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

from pydaemon import client


def main():
    cl = client.Client('localhost:8080')

    # write some data.
    # NOTE: key and data must be (bytes)
    meta = cl.file.write(b'my-key', b'hello 0-stor')

    # read the data back to a file. (download)
    cl.file.read_file(b'my-key', '/tmp/my-key')

    # get the data back using the `file` service
    d1 = cl.file.read(b'my-key')

    # or using the metadata
    d2 = cl.data.read(meta.chunks)

    assert(d1 == d2)


if __name__ == '__main__':
    main()
