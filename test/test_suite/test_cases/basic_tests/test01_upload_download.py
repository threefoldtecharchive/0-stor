from test_suite.test_cases.testcases_base import TestcasesBase
from termcolor import colored
from random import randint


class UploadDownload(TestcasesBase):
    def test001_upload_download_files(self):
        self.default_writer()
        self.get_uploaded_files_keys()
        self.default_reader()
        self.get_download_files_paths_from_threads()
        self.assertFalse(self.check_md5(self.downloaded_files_info))

    def test002_upload_file_with_small_chunck_size(self):
        random_file = self.return_random_dict_elemnt(source_dict=TestcasesBase.created_files_info)
        block_size = self.utiles.get_random_size(type='small')
        print(colored(' [*] block size %s ' % str(block_size), 'yellow'))
        new_config_file_path = self.create_new_config_file({'block_size': block_size})
        self.writer(created_files_info=random_file,
                    config_list=[new_config_file_path],
                    number_of_threads=10)
        self.get_uploaded_files_keys()
        self.reader(uploaded_files_info=self.uploaded_files_info, number_of_threads=10)
        self.get_download_files_paths_from_threads()
        self.assertFalse(self.check_md5(self.downloaded_files_info))

    def test003_upload_file_with_exact_chunck_size(self):
        pass

    def test004_upload_file_with_greater_chunck_size(self):
        pass

    def test005_upload_file_with_extremly_small_size(self):
        pass

    def test006_upload_non_exist_file(self):
        pass

    def test007_upload_file_with_write_permission(self):
        pass

    def test008_upload_file_with_read_permission(self):
        pass

    def test009_upload_file_with_read_write_permission(self):
        pass

    def test010_upload_file_with_admin_permission(self):
        pass

    def test011_download_non_exist_file(self):
        pass

    def test012_download_file_with_write_permission(self):
        pass

    def test013_download_file_with_read_permission(self):
        pass

    def test014_download_file_with_admin_permission(self):
        pass
