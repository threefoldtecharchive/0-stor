import uuid, random, time, signal, logging, yaml, queue, json
from unittest import TestCase
from nose.tools import TimeExpired
from termcolor import colored
from test_suite.framework.zero_store_cli import ZeroStoreCLI
from test_suite.framework.iyo_client.base import Client
import threading
from hashlib import md5
import subprocess
from testconfig import config
import os


class TestcasesBase(TestCase):
    created_files_info = {}
    iyo_slave_client = Client(config['main']['iyo_url'])

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.utiles = Utiles()
        self.lg = self.utiles.logging

        self.zero_store_cli = ZeroStoreCLI()
        self.default_config_path = config['main']['default_config_path']
        # if not self.default_config_path:
        #     gopath = os.environ.get('GOPATH', '/gopath')
        #     self.default_config_path = '{gopath}/src/github.com/zero-os/0-stor/cmd/zstor/config.yaml'.format(gopath=gopath)
        self.number_of_servers = int(config['main']['number_of_servers'])
        self.number_of_files = int(config['main']['number_of_files'])

        self.uploader_job_ids = []
        self.downloader_job_ids = []

        self.writer_jobs = queue.Queue()
        self.reader_jobs = queue.Queue()

        self.upload_queue_result = queue.Queue()
        self.download_queue_result = queue.Queue()

        self.uploaded_files_info = {}
        self.downloaded_files_info = {}
        self.deleted_files_info = {}

        self.iyo_master_client = Client(config['main']['iyo_url'])
        self.iyo_slave_id = config['main']['iyo_user2_id']
        self.iyo_slave_secret = config['main']['iyo_user2_secret']
        self.iyo_slave_username = config['main']['iyo_user2_username']
        self.iyo_slave_client = TestcasesBase.iyo_slave_client

    @classmethod
    def setUpClass(cls):
        self = cls()
        self.create_files(self.number_of_files)
        iyo_user2_id = config['main']['iyo_user2_id']
        iyo_user2_secret = config['main']['iyo_user2_secret']
        cls.iyo_slave_client.oauth.login_via_client_credentials(client_id=iyo_user2_id,
                                                                client_secret=iyo_user2_secret)

    def setUp(self):
        self._testID = self._testMethodName
        self._startTime = time.time()

        def timeout_handler(signum, frame):
            raise TimeExpired('Timeout expired before end of test %s' % self._testID)

        signal.signal(signal.SIGALRM, timeout_handler)
        signal.alarm(900)

        with open(self.default_config_path, 'r') as stream:
            self.config_zerostor = yaml.load(stream)
        self.iyo_master_client.oauth.login_via_client_credentials(client_id=self.config_zerostor['iyo_app_id'],
                                                                  client_secret=self.config_zerostor['iyo_app_secret'])
        print('\n')

    def tearDown(self):
        pass

    def create_files(self, number_of_files, file_size=None):
        self.execute_shell_commands(cmd='rm -rf /tmp/upload; rm -rf /tmp/download')
        self.execute_shell_commands(cmd='mkdir -p /tmp/upload; mkdir -p /tmp/download')
        for i in range(0, number_of_files):
            file_size = file_size or random.randint(1024, 10 * 1024)
            file_path = '/tmp/upload/upload_file_%d' % random.randint(1, 1000)

            with open(file_path, 'wb') as file:
                file.write(os.urandom(file_size))
            file.close()

            # file_md5 = self.md5sum(file_path=file_path)
            TestcasesBase.created_files_info['id_%d' % i] = {'path': file_path,
                                                             'size': file_size,
                                                             'md5': ''}
        #print(colored(' [*] Create : %d files under /tmp/upload ' % number_of_files, 'white'))

    def md5sum(self, file_path):
        hash = md5()
        try:
            with open(file_path, "rb") as file:
                for chunk in iter(lambda: file.read(128 * hash.block_size), b""):
                    hash.update(chunk)
            return hash.hexdigest()
        except FileNotFoundError:
            return 0

    def create_new_config_file(self, new_config):
        """
            Logic:
                - It takes new config dict, read the default config file as dict, update the default config file dict,
                create a new config file with the updated dict.
        """
        self.config_zerostor.update(new_config)
        new_config_file_path = '/tmp/config_%s.yaml' % self.random_string()
        with open(new_config_file_path, 'w') as new_config_file:
            yaml.dump(self.config_zerostor, new_config_file, default_flow_style=False)
        return new_config_file_path

    def get_piplining_compination(self):
        data, parity = self.get_data_parity()
        pipes_options = []
        for chunk in ['0', 'chunk']:
            for compress in ['0', 'snappy', 'lz', 'gzip']:
                for encrypt in ['0', 'sha256', 'blake2', 'md5']:
                    for distribution in ['0', 'distribution']:
                        pipes_options.append({'chunk': chunk,
                                              'compress': compress,
                                              'encrypt': encrypt,
                                              'distribution': distribution})
        pipes_compination = []
        for option in pipes_options:
            data = {'pipes': []}
            if option['chunk'] != '0':
                chunker = {'name': self.random_string(),
                           'config': {'chunkSize': random.randint(1, 1024000000)},
                           'type': 'chunker'}
                data['pipes'].append(chunker)

            if option['compress'] != '0':
                compress = {'name': self.random_string(),
                            'config': {'type': option['compress']},
                            'type': 'compress'}
                data['pipes'].append(compress)

            if option['encrypt'] != '0':
                encrypt = {'name': self.random_string(),
                           'config': {'nonce': 123456789012,
                                      'privKey': 'ab345678901234567890123456789012',
                                      'type': option['encrypt']},
                           'type': 'encrypt'}
                data['pipes'].append(encrypt)

            if option['distribution'] != '0':
                distribution = {'name': self.random_string(),
                                'config': {'data': data, 'parity': parity},
                                'type': 'distribution'}
                data['pipes'].append(distribution)
            pipes_compination.append(data)
        return pipes_compination

    def get_data_parity(self):
        while True:
            data = random.randint(1, self.number_of_servers)
            parity = random.randint(1, self.number_of_servers)
            if data + parity == self.number_of_servers:
                print(colored(' [*] data: %d, parity: %d' % (data, parity), 'white'))
                break
        return data, parity

    def random_string(self, size=10):
        return str(uuid.uuid4()).replace('-', '')[:size]

    def default_writer(self):
        return self.writer(created_files_info=TestcasesBase.created_files_info, config_list=[self.default_config_path])

    def writer(self, created_files_info, config_list, number_of_threads=1):
        """
            This writer takes:
                - dict of create_files {id_xx:{'path':'x', 'size':'xx', 'md5':'xx', 'thread':''} .. }
                - list of config, each item represent a dict of config
                - number of threads
            Logic:
                - If its only one file and its only one config, you can send any number of threads.
                else, it will create threads equal to #files * #configs and each thread will send each file
                with different config.
                - It will update the updated_files_info by adding the thread to each file
                - It will return threads too
        """
        if len(created_files_info) == 1 and len(config_list) == 1 and number_of_threads != 1:
            key = list(created_files_info.keys())[0]
            for i in range(0, number_of_threads):
                job_id = self.create_job_id(job_ids_list=self.uploader_job_ids)
                self.create_writer_jobs(job_id=job_id, file_info=created_files_info[key],
                                        config_path=config_list[0])
        else:
            number_of_threads = len(created_files_info) * len(config_list)
            for file_key in created_files_info:
                for config in config_list:
                    job_id = self.create_job_id(job_ids_list=self.uploader_job_ids)
                    self.uploader_job_ids.append(job_id)
                    self.create_writer_jobs(job_id=job_id, file_info=created_files_info[file_key],
                                            config_path=config)

        jobs_queue = list(self.writer_jobs.queue)
        #print(colored(' [*] Number of writer threads : %i' % number_of_threads, 'white'))
        for i in range(0, number_of_threads):
            thread = threading.Thread(target=self.writer_work)
            thread.start()
            self.uploaded_files_info[jobs_queue[i]['id']]['thread'] = thread
        return self.uploaded_files_info

    def writer_work(self):
        while not self.writer_jobs.empty():
            job = self.writer_jobs.get()
            self.zero_store_cli.upload_file(job=job, queue=self.upload_queue_result)
            self.writer_jobs.task_done()

    def create_writer_jobs(self, job_id, file_info, config_path):
        self.writer_jobs.put({'id': job_id,
                              'file_path': file_info['path'],
                              'config_path': config_path})
        self.uploaded_files_info.update({job_id: {'path': file_info['path'],
                                                  'config_path': config_path,
                                                  'size': file_info['size'],
                                                  'md5': self.md5sum(file_path=file_info['path'])}})

    def create_job_id(self, job_ids_list):
        while True:
            job_id = random.getrandbits(128)
            if job_id not in job_ids_list:
                return job_id

    def get_uploaded_files_keys(self):
        """
            This method block into threads and update uploaded_file_info with key
        """
        for job_id in self.uploaded_files_info:
            thread = self.uploaded_files_info[job_id]['thread']
            thread.join()
            upload_queue_list = list(self.upload_queue_result.queue)
            for data in upload_queue_list:
                if data['job_id'] == job_id:
                    self.uploaded_files_info[job_id]['key'] = data['uploaded_key']
                    break
            else:
                self.uploaded_files_info[job_id]['key'] = 'ERROR! : upload_queue does not have this key'

    def default_reader(self):
        return self.reader(uploaded_files_info=self.uploaded_files_info)

    def reader(self, uploaded_files_info, number_of_threads=1):
        """
            uploaded_files_info = {job_id: {file_paht:'', file_config:'', file_size:'', md5:'', key:''} }
        """
        if number_of_threads < len(uploaded_files_info):
            number_of_threads = len(uploaded_files_info)

        if number_of_threads == len(uploaded_files_info):
            for uploader_job_id in uploaded_files_info:
                job_id = self.create_job_id(job_ids_list=self.downloader_job_ids)

                self.create_reader_jobs(job_id=job_id, uploader_job_id=uploader_job_id,
                                        uploader_file_info=uploaded_files_info[uploader_job_id])
        else:
            for i in range(0, number_of_threads):
                job_id = self.create_job_id(job_ids_list=self.downloader_job_ids)
                uploader_job_id = random.choice(self.uploader_job_ids)
                self.create_reader_jobs(job_id=job_id, uploader_job_id=uploader_job_id,
                                        uploader_file_info=uploaded_files_info[uploader_job_id])

        jobs_queue = list(self.reader_jobs.queue)
        for i in range(0, number_of_threads):
            thread = threading.Thread(target=self.reader_work)
            thread.start()
            self.downloaded_files_info[jobs_queue[i]['id']]['d_info']['thread'] = thread
        return self.downloaded_files_info

    def reader_work(self):
        while not self.reader_jobs.empty():
            job = self.reader_jobs.get()
            self.zero_store_cli.download_file(job=job, queue=self.download_queue_result)
            self.reader_jobs.task_done()

    def create_reader_jobs(self, job_id, uploader_job_id, uploader_file_info):
        self.reader_jobs.put({'id': job_id,
                              'key': uploader_file_info['key'],
                              'config_path': uploader_file_info['config_path'],
                              'result': '/tmp/download/' + str(job_id)})
        self.downloaded_files_info.update({job_id: {'u_info': {'uploaded_job_id': uploader_job_id,
                                                               'key': uploader_file_info['key'],
                                                               'md5': uploader_file_info['md5'],
                                                               'size': uploader_file_info['size']},
                                                    'd_info': {'md5': ''}
                                                    }
                                           })

    def get_download_files_paths_from_threads(self):
        """
            This method block into threads and update downloaded_files_info with result
        """
        for job_id in self.downloaded_files_info:
            thread = self.downloaded_files_info[job_id]['d_info']['thread']
            thread.join()
            download_queue_list = list(self.download_queue_result.queue)
            for data in download_queue_list:
                if data['job_id'] == job_id:
                    if "ERROR" in data['downloaded_path']:
                        self.downloaded_files_info[job_id]['d_info']['path'] = data['downloaded_path']
                        self.downloaded_files_info[job_id]['d_info']['md5'] = 'There is no file!'
                    else:
                        self.downloaded_files_info[job_id]['d_info']['path'] = data['downloaded_path']
                        self.downloaded_files_info[job_id]['d_info']['md5'] = self.md5sum(
                            file_path=data['downloaded_path'])
                    break
            else:
                self.downloaded_files_info[job_id]['d_info'][
                    'path'] = 'ERROR! : download queue does not have this key'
                self.downloaded_files_info[job_id]['md5'] = 'There is no file!'

    def deleter(self, uploaded_files_info):
        """
            uploaded_files_info = {job_id: {file_paht:'', file_config:'', file_size:'', md5:'', key:''} }
        """
        for job_id in uploaded_files_info:
            uploaded_key = uploaded_files_info[job_id]['key']
            config_path = uploaded_files_info[job_id]['config_path']
            result = self.zero_store_cli.delete_file(uploaded_key=uploaded_key, config_path=config_path)
            self.deleted_files_info[job_id] = {'u_info': uploaded_files_info[job_id],
                                               'deleted_info': result}

    def check_md5(self, downloaded_files_info):
        print(colored(' [*] Compare downloaded files with uploaded files.', 'white'))
        corrupted_files = {}
        for id in downloaded_files_info:
            if downloaded_files_info[id]['d_info']['md5'] != downloaded_files_info[id]['u_info']['md5']:
                corrupted_files[id] = {'d_info': {'path': downloaded_files_info[id]['d_info']['path'],
                                                  'md5': downloaded_files_info[id]['d_info']['md5']
                                                  },
                                       'u_info': {'key': downloaded_files_info[id]['u_info']['key'],
                                                  'md5': downloaded_files_info[id]['u_info']['md5']
                                                  }
                                       }
        if corrupted_files:
            self.utiles.print_pretty_dict(dict=corrupted_files, color='red')
        return corrupted_files

    def get_random_file_to_upload(self):
        file_id = 'id_%d' % random.randint(1, 10)
        return TestcasesBase.created_files_info[file_id]

    def execute_shell_commands(self, cmd):
        # print(colored(" [*] Execute: %s" % cmd, 'white'))
        process = subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        out, error = process.communicate()
        # if error:
        #     print(colored(' [*] Error!! %s' % error.decode('utf-8'), 'red'))
        # else:
        #     print(colored(" [*] OK.", 'green'))
        return out.decode('utf-8'), error.decode('utf-8')

    def return_random_dict_elemnt(self, source_dict):
        result = {}
        random_key = random.choice(list(source_dict.keys()))
        result[random_key] = source_dict[random_key]
        return result

    def upload_download_random_file_with_specific_config(self, random_file, config_dict):
        print(colored(' [*] config %s ' % str(config_dict), 'yellow'))
        new_config_file_path = self.create_new_config_file(config_dict)
        print(colored(' [*] new config path : %s' % new_config_file_path, 'white'))
        number_of_threads = 1
        self.writer(created_files_info=random_file,
                    config_list=[new_config_file_path],
                    number_of_threads=number_of_threads)
        self.get_uploaded_files_keys()
        self.reader(uploaded_files_info=self.uploaded_files_info, number_of_threads=number_of_threads)
        self.get_download_files_paths_from_threads()

    def iyo_slave_accept_invitation(self, namespace, permissions_list):
        results = []
        for permission in permissions_list:
            if permission == '-w':
                suborg = 'write'
            elif permission == '-r':
                suborg = 'read'
            elif permission == '-d':
                suborg = 'delete'
            elif permission == '-a':
                suborg = 'admin'
            namespace_value = "%s.0stor.%s.%s" % (self.config_zerostor['organization'], namespace, suborg)
            status_code = self.iyo_slave_client.api.AcceptMembership(namespace_value, 'member',
                                                                     self.iyo_slave_username).status_code
            print(colored(' [*] namespace: %s, response status code: %d' % (namespace_value, status_code), 'yellow'))
            results.append(status_code)
        return results

    def create_namespace_and_accept_invitation(self, permissions, creat_config=True):
        self.new_namespace = "xtremx_%d_%d" % (randint(1, 10000), randint(1, 10000))
        self.assertTrue(
            self.zero_store_cli.create_namespace(namespace=self.new_namespace, config_path=self.default_config_path))
        self.assertTrue(self.zero_store_cli.set_user_acl(namespace=self.new_namespace, username=self.iyo_slave_username,
                                                         permission_list=permissions,
                                                         config_path=self.default_config_path))
        results = self.iyo_slave_accept_invitation(namespace=self.new_namespace, permissions_list=permissions)

        for status_code in results:
            self.assertEqual(status_code, 201)
        if creat_config:
            config = {'iyo_app_id': self.iyo_slave_id,
                      'iyo_app_secret': self.iyo_slave_secret,
                      'namespace': self.new_namespace}
            self.new_config_file_path = self.create_new_config_file(config)
            print(colored(' [*] new config path : %s' % self.new_config_file_path, 'white'))    

class Utiles:
    def __init__(self):
        self.config = {}
        self.logging = logging
        self.log('test_suite.log')

    def log(self, log_file_name='log.log'):
        log = self.logging.getLogger()
        fileHandler = self.logging.FileHandler(log_file_name)
        log.addHandler(fileHandler)
        self.logging.basicConfig(filename=log_file_name, filemode='rw', level=logging.INFO,
                                 format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')

    def print_pretty_dict(self, dict, color):
        print('\n')
        print(colored(json.dumps(dict, sort_keys=True, indent=4), color))

    def get_random_size(self, type):
        if type == 'small':
            return 2 ** random.randint(4, 11)
        elif type == 'medium':
            return 2 ** random.randint(10, 14)
        elif type == 'large':
            return 2 ** random.randint(14, 20)
