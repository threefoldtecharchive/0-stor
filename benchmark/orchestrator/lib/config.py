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

"""
    Package config includes functions to set up configuration for benchmarking scenarios
"""
import time
import os
from re import split
from copy import deepcopy
from subprocess import check_output
import yaml
from lib.zstor_local_setup import SetupZstor

class InvalidBenchmarkConfig(Exception):
    pass

# list of supported benchmark parameters
PARAMETERS = {'block_size',
              'key_size',
              'value_size',
              'clients',
              'method',
              'block_size',
              'data_shards',
              'parity_shards',
              'meta_shards_nr',
              'zstordb_jobs'}
PARAMETERS_DICT = {'encryption': {'type', 'private_key'},
                   'compression': {'type', 'mode'}}

PROFILES = {'cpu', 'mem', 'trace', 'block'}

class Config:
    """
    Class Config includes functions to set up environment for the benchmarking:
        - deploy zstor servers
        - config zstor benchmark client
        - iterate over range of benchmark parameters

    @template contains zerostor client config
    @benchmark defines iterator over provided benchmarks
    """
    def __init__(self, config_file):
        # read config yaml file
        with open(config_file, 'r') as stream:
            config = yaml.load(stream)
        # fetch template config for benchmarking
        self._template0 = config.get('template', None)
        self.restore_template()

        # fetch bench_config from template
        bench_config = self.template.get('benchmark', None)
        if not bench_config:
            raise InvalidBenchmarkConfig('no benchmark config given in the template')
        self.zstordb_jobs = bench_config.get('zstordb_jobs', 0)

        if not self.template:
            raise InvalidBenchmarkConfig('no zstor config given')

        # extract benchmarking parameters
        self.benchmark = iter(self.benchmark_generator(config.pop('benchmarks', None)))
        # extract profiling parameter
        self.profile = config.get('profile', None)

        if self.profile and (self.profile not in PROFILES):
            raise InvalidBenchmarkConfig("profile mode '%s' is not supported"%self.profile)

        self.count_profile = 0
        self.deploy = SetupZstor()
        self.meta_shards_nr = 1
        self.data_shards_nr = 1
        self.datastor = {}
        self.metastor = {}

    def new_profile_dir(self, path=""):
        """
        Create new directory for profile information in given path and dump current config
        Returns zstordb profile dir and client profile dir
        """

        if self.profile:
            directory = os.path.join(path, 'profile_information')
            if not os.path.exists(directory):
                os.makedirs(directory)
            directory = os.path.join(directory, 'profile_' + str(self.count_profile))
            if not os.path.exists(directory):
                os.makedirs(directory)

            file = os.path.join(directory, 'config.yaml')
            with open(file, 'w+') as outfile:
                yaml.dump({'scenarios': {'scenario': self.template}}, 
                            outfile, 
                            default_flow_style=False, 
                            default_style='')

            zstordb_dir = os.path.join(directory, 'zstordb')
            zstor_client_dir = os.path.join(directory, 'zstorclient')

            self.count_profile += 1
            return zstordb_dir, zstor_client_dir
        return "", "" 

    def benchmark_generator(self, benchmarks):
        """
        Iterate over list of benchmarks
        """

        if not benchmarks:
            yield BenchmarkPair()
        for bench in benchmarks:
            yield BenchmarkPair(bench)

    def alter_template(self, key_id, val): 
        """
        Recurcively search and ppdate @id config field with new value @val
        """

        def replace(d, key_id, val):
            for key in list(d.keys()):
                v = d[key]
                if not isinstance(v, dict):
                    if key != key_id:
                        continue
                    parameter_type = type(d[key])
                    try:
                        d[key] = parameter_type(val)
                    except:
                        raise InvalidBenchmarkConfig(
                            "for '{}' cannot convert val = {} to type {}".format(
                                key,val,parameter_type))
                    return True
                if isinstance(key_id, dict) and key == list(key_id.items())[0][0]:
                    return replace(v, key_id[key], val)
                if replace(v, key_id, val):
                    return True
            return False
        if not replace(self.template, key_id, val):
            raise InvalidBenchmarkConfig("parameter %s is not supported"%key_id)

    def restore_template(self):
        """ Restore initial zstor config """

        self.template = deepcopy(self._template0)

    def save(self, file_name):
        """ Save current config to file """

        # prepare config for output
        output = {'scenarios': {'scenario': self.template}}

        # write scenarios to a yaml file
        with open(file_name, 'w+') as outfile:
            yaml.dump(output, outfile, default_flow_style=False, default_style='')

    def update_deployment_config(self):
        """ 
        Fetch current zstor server deployment config
                ***specific for beta2***
        """

        # ensure that zstor config is given as dictionary
        if not self.template.get('zstor', None):
            self.template.update({'zstor': {}})
        zstor = self.template.get('zstor', {})

        # ensure that datastor config is given as dictionary
        if not zstor.get('datastor', None):
            zstor.update({'datastor': {}})
        self.datastor =  zstor['datastor']

        # ensure that pipeline config is given as dictionary
        if not self.datastor.get('pipeline', None):
            self.datastor.update({'pipeline': {}})
        pipeline = self.datastor['pipeline']
        distribution = pipeline.get('distribution', {})
        data_shards = distribution.get('data_shards', 1)
        parity_shards = distribution.get('parity_shards', 0)

        pipeline.update({'distribution': {
                            'data_shards': data_shards, 
                            'parity_shards':parity_shards}})

        self.data_shards_nr =  int(data_shards) + int(parity_shards)

        if 'metastor' not in zstor:
            zstor.update({'metastor': {'meta_shards_nr':1} })
        
        self.metastor = zstor['metastor']
        if 'meta_shards_nr' in self.metastor:
            self.meta_shards_nr = int(self.metastor['meta_shards_nr'])
      
        self.IYOtoken = self.template['zstor'].get('iyo', None)

    def deploy_zstor(self, profile_dir="profile_zstordb"):
        """
        Run zstordb and etcd servers
        Profile parameters will be used for zstordb
        """

        self.update_deployment_config()
        self.deploy.run_data_shards(servers=self.data_shards_nr,
                                        no_auth=(self.IYOtoken == None),
                                        jobs=self.zstordb_jobs,
                                        profile=self.profile,
                                        profile_dir=profile_dir)

        self.deploy.run_meta_shards(servers=self.meta_shards_nr)

        # wait for servers to start
        self.wait_local_servers_to_start()

        self.datastor.update({'shards': self.deploy.data_shards})
        self.metastor.update({'db':{'endpoints': self.deploy.meta_shards}})

    def wait_local_servers_to_start(self):
        """ Check whether ztror and etcd servers are listening on the ports """

        addrs = self.deploy.data_shards + self.deploy.meta_shards
        servers = 0
        timeout = time.time() + 20
        while servers < len(addrs):
            servers = 0
            for addr in addrs:
                port = ':%s'%split(':', addr)[-1]
                try:
                    responce = check_output(['lsof', '-i', port])
                except:
                    responce = 0
                if responce:
                    servers += 1
                if time.time() > timeout:
                    raise TimeoutError("couldn't run all required servers. Check that ports are free")

    def run_benchmark(self, config='config.yaml', out='result.yaml', profile_dir='./profile'):
        """ Runs benchmarking """

        self.deploy.run_benchmark(config=config,
                                    out=out,
                                    profile=self.profile,
                                    profile_dir=profile_dir)

    def stop_zstor(self):
        """ Stop zstordb and datastor servers """
        self.deploy.stop()
        self.datastor.update({'shards': []})
        self.metastor.update({'db':{'endpoints': []}})

class Benchmark():
    """ Benchmark class is used defines and validates benchmark parameter """

    def __init__(self, parameter={}):
        if not parameter:
             # return empty Benchmark
            self.range = [' ']
            self.id = ''
            return
    
        self.id = parameter.get('id')
        self.range = parameter.get('range', [])

        # check if parameter id or range are missing
        if not self.id or not self.range:
            raise InvalidBenchmarkConfig("parameter id or range is missing")
        
        # check if given parameter id is present in list of supported parameters
        if not isinstance(self.id, dict):
             # if parameter id is given as string check if included in PARAMETERS
            if self.id not in PARAMETERS:
                raise InvalidBenchmarkConfig("parameter {0} is not supported".format(self.id))
            return
    
        # if parameter id is given as dictionary, check if included in PARAMETERS_DICT
        def contain(d, id):
            if not isinstance(d, dict) or not isinstance(id, dict):
                return id in d
            for key in list(d.keys()):
                if id.get(key) and contain(d[key], id[key]):
                    return True
            return False
        if not contain(PARAMETERS_DICT, self.id):
            raise InvalidBenchmarkConfig("parameter {0} is not supported".format(self.id))

    def empty(self):
        """ Return True if benchmark is empty """

        return len(self.range) == 1 and not self.id

class BenchmarkPair():
    """
    BenchmarkPair defines primary and secondary parameter for benchmarking
    """

    def __init__(self, bench_pair={}):
        if not bench_pair:
            # define empty benchmark
            self.prime = Benchmark()
            self.second = Benchmark()
            return

        # extract parameters from a dictionary
        self.prime = Benchmark(bench_pair.pop('prime_parameter', None))
        self.second = Benchmark(bench_pair.pop('second_parameter', None))

        if not self.prime.empty() and self.prime.id == self.second.id:
            raise InvalidBenchmarkConfig("primary and secondary parameters should be different")

        if self.prime.empty() and not self.second.empty():
            raise InvalidBenchmarkConfig("if secondary parameter is given, primary parameter has to be given")
