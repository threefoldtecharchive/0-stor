from test_suite.test_cases.testcases_base import TestcasesBase
from termcolor import colored
from random import randint
from parameterized import parameterized


class BasicTestCases(TestcasesBase):
    def test001_upload_download_files(self):
        self.default_writer()
        self.get_uploaded_files_keys()
        self.default_reader()
        self.get_download_files_paths_from_threads()
        self.assertFalse(self.check_md5(self.downloaded_files_info))

    @parameterized.expand([('small_block_size', 'small'),
                           ('medium_block_size', 'medium'),
                           ('large_block_size', 'large'),
                           ('Exact_size', 'exact_size')])
    def test002_upload_file_with(self, name, type):
        random_file = self.return_random_dict_elemnt(source_dict=TestcasesBase.created_files_info)
        if type == 'exact_size':
            block_size = list(random_file.values())[0]['size']
        else:
            block_size = self.utiles.get_random_size(type=type)
        self.upload_download_random_file_with_specific_config(random_file=random_file,
                                                              config_dict={'block_size': block_size})
        self.assertFalse(self.check_md5(self.downloaded_files_info))

    def test003_upload_non_existing_file(self):
        random_file = self.return_random_dict_elemnt(source_dict=TestcasesBase.created_files_info)
        old_path = random_file[list(random_file.keys())[0]]['path']
        random_file[list(random_file.keys())[0]]['path'] += str(randint(1000, 20000))
        self.writer(created_files_info=random_file, config_list=[self.default_config_path])
        self.get_uploaded_files_keys()
        random_file[list(random_file.keys())[0]]['path'] = old_path
        self.assertIn("can't read the file", self.uploaded_files_info[list(self.uploaded_files_info.keys())[0]]['key'])

    @parameterized.expand([('write', ['-w']),
                           ('read', ['-r']),
                           ('write_read', ['-r', '-w']),
                           ('delete', ['-d']),
                           ('delete_read', ['-d', '-r']),
                           ('delete_write', ['-d', '-w']),
                           ('delete_write_read', ['-d', '-w', '-r']),
                           ])
    def test004_upload_file_with_permission(self, name, permissions):
        random_file = self.return_random_dict_elemnt(source_dict=TestcasesBase.created_files_info)
        self.create_namespace_and_accept_invitation(permissions=permissions)

        self.writer(created_files_info=random_file,
                    config_list=[self.new_config_file_path],
                    number_of_threads=1)

        self.get_uploaded_files_keys()

        self.zero_store_cli.delete_namespace(namespace=self.new_namespace, config_path=self.default_config_path)

        if 'write' in name:
            self.assertNotIn("ERROR!", self.uploaded_files_info[list(self.uploaded_files_info.keys())[0]]['key'])
        else:
            self.assertIn("ERROR!", self.uploaded_files_info[list(self.uploaded_files_info.keys())[0]]['key'])

    def test005_delete_file_by_owner(self):
        random_file = self.return_random_dict_elemnt(source_dict=TestcasesBase.created_files_info)
        self.writer(created_files_info=random_file, config_list=[self.default_config_path], number_of_threads=1)
        self.get_uploaded_files_keys()
        self.assertNotIn("ERROR!", self.uploaded_files_info[list(self.uploaded_files_info.keys())[0]]['key'])

        self.deleter(uploaded_files_info=self.uploaded_files_info)
        self.assertNotIn("ERROR!", self.deleted_files_info[list(self.deleted_files_info.keys())[0]]['deleted_info'])

        self.reader(uploaded_files_info=self.uploaded_files_info, number_of_threads=1)
        self.get_download_files_paths_from_threads()
        self.assertEqual('There is no file!',
                         self.downloaded_files_info[list(self.downloaded_files_info.keys())[0]]['d_info']['md5'])

    @parameterized.expand([('write', ['-w']),
                           ('read', ['-r']),
                           ('write_read', ['-r', '-w']),
                           ('delete', ['-d']),
                           ('delete_read', ['-d', '-r']),
                           ('delete_write', ['-d', '-w']),
                           ('delete_write_read', ['-d', '-w', '-r']),
                           ])
    def test006_delete_file_by_iyo_slave_user(self, name, permissions):
        """ ZST-008

        **Test Scenario:**
        #. Create random file
        #. Create a new namespace as a master user.
        #. Send an invitation to the slave user and accept it.
        #. Upload the file as a master user.
        #. Try to delete the file as a slave user and check the result.
        """
        random_file = self.return_random_dict_elemnt(source_dict=TestcasesBase.created_files_info)
        self.create_namespace_and_accept_invitation(permissions=permissions, creat_config=False)

        config = {'namespace': self.new_namespace}
        new_config_file_path_upload = self.create_new_config_file(config)
        print(colored(' [*] uploading config path : %s' % new_config_file_path_upload, 'white'))
        self.writer(created_files_info=random_file,
                    config_list=[new_config_file_path_upload],
                    number_of_threads=1)
        self.get_uploaded_files_keys()

        config = {'iyo_app_id': self.iyo_slave_id,
                  'iyo_app_secret': self.iyo_slave_secret,
                  'namespace': self.new_namespace}
        new_config_file_path = self.create_new_config_file(config)
        print(colored(' [*] delete config path : %s' % new_config_file_path, 'white'))
        self.uploaded_files_info[list(self.uploaded_files_info.keys())[0]]['config_path'] = new_config_file_path
        self.deleter(uploaded_files_info=self.uploaded_files_info)

        if 'delete' in name:
            self.assertEqual("file deleted successfully",
                             self.deleted_files_info[list(self.deleted_files_info.keys())[0]]['deleted_info'])
        else:
            self.assertIn("JWT token doesn\'t contains required scopes", self.deleted_files_info[list(self.deleted_files_info.keys())[0]]['deleted_info'])

    @parameterized.expand([('write', ['-w']),
                           ('read', ['-r']),
                           ('write_read', ['-r', '-w']),
                           ('delete', ['-d']),
                           ('delete_read', ['-d', '-r']),
                           ('delete_write', ['-d', '-w']),
                           ('delete_write_read', ['-d', '-w', '-r']),
                           ])
    def test008_download_file_with_permission(self, name, permissions):
        """ ZST-008

        **Test Scenario:**
        #. Create random file
        #. Create a new namespace as a master user.
        #. Send an invitation to the slave user and accept it.
        #. Upload the file as a master user
        #. Try to download the file as a slave user and check the result.
        """
        random_file = self.return_random_dict_elemnt(source_dict=TestcasesBase.created_files_info)
        self.create_namespace_and_accept_invitation(permissions=permissions, creat_config=False)

        config = {'namespace': self.new_namespace}
        new_config_file_path_upload = self.create_new_config_file(config)
        print(colored(' [*] uploading config path : %s' % new_config_file_path_upload, 'white'))
        self.writer(created_files_info=random_file,
                    config_list=[new_config_file_path_upload],
                    number_of_threads=1)
        self.get_uploaded_files_keys()

        config = {'iyo_app_id': self.iyo_slave_id,
                  'iyo_app_secret': self.iyo_slave_secret,
                  'namespace': self.new_namespace}
        new_config_file_path = self.create_new_config_file(config)
        print(colored(' [*] downloading config path : %s' % new_config_file_path, 'white'))

        self.uploaded_files_info[list(self.uploaded_files_info.keys())[0]]['config_path'] = new_config_file_path
        self.reader(uploaded_files_info=self.uploaded_files_info, number_of_threads=1)
        self.get_download_files_paths_from_threads()

        self.zero_store_cli.delete_namespace(namespace=self.new_namespace, config_path=self.default_config_path)

        if 'read' in name:
            self.assertFalse(self.check_md5(self.downloaded_files_info))
        else:
            self.assertIn("JWT token doesn\'t contains required scopes",
                          self.downloaded_files_info[list(self.downloaded_files_info.keys())[0]]['d_info']['path'])

    def test009_create_delete_namespace(self):
        namespace = "xtremx_%d" % randint(1, 10000)
        self.assertTrue(self.zero_store_cli.create_namespace(namespace=namespace, config_path=self.default_config_path))
        self.assertTrue(self.zero_store_cli.delete_namespace(namespace=namespace, config_path=self.default_config_path))

    def test010_upload_file_to_non_existing_namespace(self):
        """ ZST-010

        **Test Scenario:**
        #. Create random file
        #. Update the config file with non existing namespace.
        #. Try to upload the file, should fail
        """
        random_file = self.return_random_dict_elemnt(source_dict=TestcasesBase.created_files_info)
        config = {'namespace': 'xTremx00xTremx00'}
        new_config_file_path_upload = self.create_new_config_file(config)
        print(colored(' [*] uploading config path : %s' % new_config_file_path_upload, 'white'))
        self.writer(created_files_info=random_file,
                    config_list=[new_config_file_path_upload],
                    number_of_threads=1)
        self.get_uploaded_files_keys()
        self.assertIn("JWT token doesn\'t contains required scopes", self.uploaded_files_info[list(self.uploaded_files_info.keys())[0]]['key'])
