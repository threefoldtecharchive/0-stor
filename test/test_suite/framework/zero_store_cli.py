from termcolor import colored
import subprocess, time


class ZeroStoreCLI:
    def download_file(self, job, queue):
        job_id = job['id']
        key = job['key']
        config_path = job['config_path']

        result = job['result']
        command = 'zstor --config %s file download %s -o %s' % (config_path, key, result)
        for _ in range(10):
            out, return_code = self.execute_shell_commands(command)
            if return_code:
                time.sleep(1)
            else:
                break

        if return_code:
            queue.put({'job_id': job_id,
                       'downloaded_path': 'ERROR! %s ' % out})
        else:
            if 'downloaded' in out:
                queue.put({'job_id': job_id,
                           'downloaded_path': result})
            else:
                queue.put({'job_id': job_id,
                           'downloaded_path': 'ERROR! %s ' % str(out)})

    def upload_file(self, job, queue):
        file_path = job['file_path']
        config_path = job['config_path']
        job_id = job['id']
        command = 'zstor --config %s file upload %s' % (config_path, file_path)
        out, return_code = self.execute_shell_commands(command)
        if return_code:
            queue.put({'job_id': job_id,
                       'uploaded_key': 'ERROR! %s ' % out})
        else:
            result = out.split('\n')[-2]
            if 'uploaded as key =' in result:
                key = file_path.split('/')[-1]
                queue.put({'job_id': job_id,
                           'uploaded_key': key})
            else:
                queue.put({'job_id': job_id,
                           'uploaded_key': 'ERROR!'})

    def create_namespace(self, namespace, config_path):
        print(colored(" [*] Create namespace : %s" % namespace, 'white'))
        command = 'zstor --config %s namespace create %s' % (config_path, namespace)
        out, return_code = self.execute_shell_commands(command)
        if return_code:
            print(colored(" ERROR : %s " % str(out), 'red'))
            return False
        else:
            print(colored(" [*] %s " % out, 'green'))
            return True

    def delete_namespace(self, namespace, config_path):
        command = 'zstor --config %s namespace delete %s' % (config_path, namespace)
        out, return_code = self.execute_shell_commands(command)
        if return_code:
            print(colored(" ERROR : %s " % str(out), 'red'))
            # TO DO : Verify that this namespace has been deleted
            return False
        else:
            return True

    def get_user_acl(self, namespace, usermail, config_path):
        command = "zstor --config %s namespace permission get %s %s" % (config_path, usermail, namespace)
        out, return_code = self.execute_shell_commands(command)
        if return_code:
            print(colored(" ERROR : %s " % str(out), 'red'))
            return False
        else:
            return True

    def set_user_acl(self, namespace, usermail, permission_list, config_path):
        permissions = ''
        for permission in permission_list:
            permissions += permission + " "
        permissions = permissions[:-1]
        command = "zstor --config %s namespace permission set %s %s %s" % (config_path,  usermail, namespace, permissions)
        out, return_code = self.execute_shell_commands(command)
        if return_code:
            print(colored(" ERROR : %s " % str(out), 'red'))
            return False
        else:
            return True

    def delete_file(self, uploaded_key, config_path):
        command = 'zstor --config %s file delete %s' % (config_path, uploaded_key)
        for _ in range(10):
            out, return_code = self.execute_shell_commands(command)
            if return_code:
                time.sleep(1)
            else:
                break
        if return_code:
            return 'delete_result: ERROR! %s ' % out
        else:
            if 'file deleted successfully' in out:
                return 'delete_result: %s ' % out
            else:
                return 'delete_result: ERROR! %s ' % out

    def execute_shell_commands(self, cmd):
        #print(colored(" [*] Execute: %s" % cmd, 'white'))
        process = subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        out, error = process.communicate()
        out += error
        # if error:
        #     print(colored(' [*] Error!! %s' % error.decode('utf-8'), 'red'))
        # else:
        #     print(colored(" [*] OK.", 'green'))
        return out.decode('utf-8'), process.returncode
