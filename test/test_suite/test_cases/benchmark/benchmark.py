from test_suite.test_cases.testcases_base import TestcasesBase
from termcolor import colored
import yaml
from parameterized import parameterized
import time, statistics


class Benchmark(TestcasesBase):
    @classmethod
    def setUpClass(cls):
        pass

    def setUp(self):
        self._testID = self._testMethodName
        self._startTime = time.time()

        with open(self.default_config_path, 'r') as stream:
            self.config_zerostor = yaml.load(stream)
        self.iyo_master_client.oauth.login_via_client_credentials(client_id=self.config_zerostor['iyo_app_id'],
                                                                  client_secret=self.config_zerostor['iyo_app_secret'])
        print('\n')
    @parameterized.expand((['64K_100', 100],
                           ['64K_200', 200],
                           ['3M_100', 100],
                           ['3M_200', 200],
                           ))
    def test_benchmark_upload(self, mode, datasets):
        print(colored(' [*] Uploading Performance Test', 'green'))
        print(colored(' [*] Mode : %s ' % mode, 'yellow'))
        self.time = {mode: []}

        if '64K' in mode:
            file_size = 64 * 1024
        else:
            file_size = 3 * 1024 * 1024

        for dataset in range(1, datasets):
            self.create_files(number_of_files=1, file_size=file_size)
            start = time.time()
            self.default_writer()
            self.get_uploaded_files_keys()
            end = time.time()
            self.time[mode].append(end - start)
            self.assertNotIn('ERROR', self.uploaded_files_info[list(self.uploaded_files_info.keys())[0]]['key'])
        print(colored(' Mode : %s \n  Uploading avg. time : %f' % (mode, statistics.mean(self.time[mode])), 'green'))

    @parameterized.expand((['64K_100', 100],
                           ['64K_200', 200],
                           ['3M_100', 100],
                           ['3M_200', 200],
                           ))
    def test_benchmark_download(self, mode, datasets):
        print(colored(' [*] Downloading Performance Test', 'green'))
        print(colored(' [*] Mode : %s ' % mode, 'yellow'))
        self.time = {mode: []}
        if 'small' in mode:
            file_size = 64 * 1024
        else:
            file_size = 3 * 1024 * 1024
        for dataset in range(1, datasets):
            self.create_files(number_of_files=1, file_size=file_size)
            self.default_writer()
            self.get_uploaded_files_keys()
            self.assertNotIn('ERROR', self.uploaded_files_info[list(self.uploaded_files_info.keys())[0]]['key'])
            start = time.time()
            self.default_reader()
            self.get_download_files_paths_from_threads()
            end = time.time()
            self.time[mode].append(end - start)

        print(colored(' Mode : %s \n  Uploading avg. time : %f' % (mode, statistics.mean(self.time[mode])), 'green'))
