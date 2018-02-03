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

import os
import shutil
import subprocess
import tempfile
import signal
from random import randint
import socket

Base = '127.0.0.1' # base of addresses at the local host

class SetupZstor:
    """ SetupZstor is responsible for managing a zstor setup """

    def __init__(self, ):
        self.zstor_nodes = []
        self.etcd_nodes = []
        self.cleanup_dirs = []
        self.data_shards = []
        self.meta_shards = []

    def run_data_shards(self,
                        servers=2,
                        no_auth=True,
                        jobs=0,
                        profile=None,
                        profile_dir="profile",
                        data_dir=None,
                        meta_dir=None):
        """  Start zstordb servers """

        for i in range(0, servers):
            if not data_dir:
                db_dir = tempfile.mkdtemp()
            else:
                db_dir = os.path.join(data_dir, str(i))
                os.makedirs(data_dir)

            if not meta_dir:
                md_dir = tempfile.mkdtemp()
            else:
                md_dir = os.path.join(meta_dir, str(i))
                os.makedirs(meta_dir)

            self.cleanup_dirs.extend((db_dir, md_dir))

            port = str(pick_free_port())
            self.data_shards.append('%s:%s'%(Base,port))

            args = ["zstordb",
                    "--listen", ":" + port,
                    "--data-dir", db_dir,
                    "--meta-dir", md_dir,
                    "--jobs", str(jobs),
                    ]

            if profile and is_profile_flag(profile):
                args.extend(("--profile-mode", profile))

                profile_dir_zstordb = profile_dir + "_" + str(i)

                if not os.path.exists(profile_dir_zstordb):
                    os.makedirs(profile_dir_zstordb)

                args.extend(("--profile-output", profile_dir_zstordb))

            if no_auth:
                args.append("--no-auth")

            self.zstor_nodes.append(subprocess.Popen(args, stderr=subprocess.PIPE))

    # stop data shards
    def stop_data_shards(self):
        for node in self.zstor_nodes:
            node.send_signal(signal.SIGINT)
            try:
                node.communicate(timeout=5)
            except subprocess.TimeoutExpired:
                print("Timed out waiting for zstordb to close, killing it")
                node.kill()

        self.zstor_nodes = []
        self.data_shards = []

    def run_meta_shards(self, servers=2, data_dir=""):
        """ Start etcd servers on random free ports """

        cluster_token = "etcd-cluster-" + str(randint(0, 99))
        names = []
        peer_addresses = []
        client_addresses = []
        init_cluster = ""
        base = "http://127.0.0.1:"

        for i in range(0, servers):
            name = "node" + str(i)
            port = str(pick_free_port())
            self.meta_shards.append('%s:%s'%(Base, port))

            client_port = base + port
            peer_port = base + str(pick_free_port())
            init_cluster += name + "=" + peer_port + ","

            names.append(name)
            peer_addresses.append(peer_port)
            client_addresses.append(client_port)

        for i in range(0, servers):
            name = names[i]
            client_address = client_addresses[i]
            peer_address = peer_addresses[i]

            if data_dir == "":
                db_dir = tempfile.mkdtemp()
            else:
                db_dir = data_dir + "/etcd" + str(i)

            self.cleanup_dirs.append(db_dir)

            args = ["etcd",
                    "--name", name,
                    "--initial-advertise-peer-urls", peer_address,
                    "--listen-peer-urls", peer_address,
                    "--listen-client-urls", client_address,
                    "--advertise-client-urls", client_address,
                    "--initial-cluster-token", cluster_token,
                    "--initial-cluster", init_cluster,
                    "--data-dir", db_dir,
                   ]
            self.etcd_nodes.append(subprocess.Popen(args,
                                                    stdout=subprocess.PIPE,
                                                    stderr=subprocess.PIPE))

    def stop_meta_shards(self):
        """ Stop etcd servers """

        for node in self.etcd_nodes:
            node.terminate()

        self.etcd_nodes = []
        self.meta_shards = []

    def cleanup(self):
        """ Delete all directories in cleanup """
        while self.cleanup_dirs:
            shutil.rmtree(self.cleanup_dirs.pop(), ignore_errors=True)

    def stop(self):
        """ Stop zstordb and etcd servers """        

        self.stop_meta_shards()
        self.stop_data_shards()
        self.cleanup()

    @staticmethod
    def run_benchmark(profile=None,
                        profile_dir="profile_client",
                        config="client_config.yaml",
                        out="bench_result.yaml"):
        """ Run benchmark client"""

        args = ["zstorbench",
                "--conf", config,
                "--out-benchmark",
                out,
                ]

        if profile and is_profile_flag(profile):
            args.extend(("--profile-mode", profile))
            args.extend(("--out-profile", profile_dir))

        # run benchmark client
        subprocess.run(args, )

def is_profile_flag(flag):
    """ return true if provided profile flag is valid """
    return flag in ('cpu', 'mem', 'block', 'trace')

def pick_free_port():
    """ Pick free port using socket """

    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.bind(('localhost', 0))
    addr, port = s.getsockname()
    s.close()
    return port

if __name__ == '__main__':
    z = SetupZstor()
    from IPython import embed
    embed()
