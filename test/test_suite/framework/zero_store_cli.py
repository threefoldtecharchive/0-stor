from termcolor import colored
import subprocess, time


class ZeroStoreCLI:
    def download_file(self, job, queue):
        job_id = job['id']
        key = job['key']
        config_path = job['config_path']

        result = job['result']
        command = 'zerostorcli --conf %s file download %s %s' % (config_path, key, result)
        #command = 'cd /gopath/src/github.com/zero-os/0-stor/cmd/zstor/ && zerostorcli file download %s %s' % (key, result)
        for _ in range(10):
            out, error = self.execute_shell_commands(command)
            if error:
                time.sleep(1)
            else:
                break

        if error:
            queue.put({'job_id': job_id,
                       'downloaded_path': 'ERROR! %s ' % error})
        else:
            result = out.split(' ')
            if '/tmp/download/' in result[3]:
                path = result[3][:-1]
                queue.put({'job_id': job_id,
                           'downloaded_path': path})
            else:
                queue.put({'job_id': job_id,
                           'downloaded_path': 'ERROR! %s ' % str(out)})

    def upload_file(self, job, queue):
        file_path = job['file_path']
        config_path = job['config_path']
        job_id = job['id']
        command = 'zerostorcli --conf %s file upload %s' % (config_path, file_path)
        #command = 'cd /gopath/src/github.com/zero-os/0-stor/cmd/zstor && zerostorcli file upload %s' % file_path
        out, error = self.execute_shell_commands(command)
        if error:
            queue.put({'job_id': job_id,
                       'uploaded_key': 'ERROR! %s ' % error})
        else:
            result = out.split('\n')[-2]
            if 'file uploaded' in result:
                key = result.split(' ')[-1]
                queue.put({'job_id': job_id,
                           'uploaded_key': key})
            else:
                queue.put({'job_id': job_id,
                           'uploaded_key': 'ERROR!'})

    def create_namespace(self, namespace, config_path):
        print(colored(" [*] Create namespace : %s" % namespace, 'white'))
        command = 'zerostorcli --conf %s namespace create %s' % (config_path, namespace)
        out, error = self.execute_shell_commands(command)
        if error:
            print(colored(" ERROR : %s " % str(error), 'red'))
            return False
        else:
            print(colored(" [*] %s " % out, 'green'))
            return True

    def delete_namespace(self, namespace, config_path):
        command = 'zerostorcli --conf %s namespace delete %s' % (config_path, namespace)
        out, error = self.execute_shell_commands(command)
        if error:
            print(colored(" ERROR : %s " % str(error), 'red'))
            # TO DO : Verify that this namespace has been deleted
            return False
        else:
            return True

    def get_user_acl(self, namespace, username, config_path):
        command = "zerostorcli %s conf namespace get-acl --namespace %s --user %s" % (config_path, namespace, username)
        out, error = self.execute_shell_commands(command)
        if error:
            print(colored(" ERROR : %s " % str(error), 'red'))
            return False
        else:
            return True

    def set_user_acl(self, namespace, username, permission_list, config_path):
        permissions = ''
        for permission in permission_list:
            permissions += permission + " "
        permissions = permissions[:-1]

        command = "zerostorcli --conf %s namespace set-acl --namespace %s --user %s %s" % (config_path, namespace, username, permissions)
        out, error = self.execute_shell_commands(command)
        if error:
            print(colored(" ERROR : %s " % str(error), 'red'))
            return False
        else:
            return True

    def delete_file(self, uploaded_key, config_path):
        command = 'zerostorcli --conf %s file delete %s' % (config_path, uploaded_key)
        for _ in range(10):
            out, error = self.execute_shell_commands(command)
            if error:
                time.sleep(1)
            else:
                break

        if error:
            return 'delete_result: ERROR! %s ' % error
        else:
            if 'file deleted successfully' in out:
                return 'delete_result: %s ' % out
            else:
                return 'delete_result: ERROR! %s ' % out

    def execute_shell_commands(self, cmd):
        #print(colored(" [*] Execute: %s" % cmd, 'white'))
        process = subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        out, error = process.communicate()
        # if error:
        #     print(colored(' [*] Error!! %s' % error.decode('utf-8'), 'red'))
        # else:
        #     print(colored(" [*] OK.", 'green'))
        return out.decode('utf-8'), error.decode('utf-8')
