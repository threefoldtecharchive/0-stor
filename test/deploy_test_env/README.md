## Prerequisites 
### Create Zero-os node
- Go to packet.net
- Deploy a server with operating system ``custom_ipxe`` and use this ipxe link:
``
https://bootstrap.gig.tech/ipxe/zero-os-<core0_branch>/<zero_os_zerotier_network>/organization=<iyo_organization>
``
    - ``core_0_branch``: [0-core](https://github.com/zero-os/0-core) branch
    - ``zero_os_zerotier_network``: the zerotier network of the Zero-os node.
    - ``iyo_organization``: itsyou.online organization

- install 0-core python client using this command:
``bash
pip3 install -U "git+https://github.com/zero-os/0-core.git@${core_0_branch}#subdirectory=client/py-client"
``
- Connect to the node using the python client and mount any disk to store container data, for example to mount disk ``sda`` :

```python
from zeroos.core0.client import Client

client = Client('<zero-os_zerotier_ip>', password='<jwt>')
client.bash('mkfs.btrfs /dev/sda').get()
client.bash('mount /dev/sda /var/cache').get()
```


## What this script does
Automate the installation of the 0-stor testing environment.

- Create an orchestrator container on the provided Zero-os node using [autosetup script](https://github.com/zero-os/0-orchestrator/tree/master/autosetup).
- Create 0-stor nodes on packet.net.
- Deploy etcd cluster and object cluster on the 0-stor nodes.
> Note: the etcd cluster will be deployed on the maximum odd number of the 0-stor nodes
- Execute 0-stor testsuite.

## Install testing environment

#### Set environment variables:
- ``TRAVIS_BUILD_NUMBER``: any random number (you don't have to set it if you trigger the build from travis)
- ``TRAVIS_EVENT_TYPE``: ``api`` (you don't have to set it if you trigger the build from travis)

- ``packet_token`` : packet.net access token to be able to create 0-stor nodes. ( create your own from [here](https://app.packet.net/portal#/api-keys) )

- ``github_token``: github token to create/delete ays repo ( create you own from [here](https://github.com/settings/tokens) )
- ``zerotier_token``: zerotier account token. ( create your own form [here](https://my.zerotier.com/) )
- ``zero_os_zerotier_token``: zerotier account token of the zero-os node (default: value of ```zerotier_token``` )
- ``zero_os_zerotier_network``: zerotier network id of the Zero-os node.

- ``server_ip`` : the zerotier ip address of the Zero-os node.

- ``ZERO_STOR_CLIENT_BRANCH``: 0-stor client branch branch.
- ``ZERO_STORE_SERVER_BRANCH``: 0-stor server branch.

- ``iyo_organization``: itsyou.online organization.
- ``iyo_client_id`` : itsyou.online user client id.
- ``iyo_secret`` : itsyou.online user client secret.
- ``iyo_namespace`` : itsyou.online organization namespace.

- ``orchestrator_branch``: [0-core](https://github.com/zero-os/0-core) branch to be used to build the 0-stor flist.
- ``core_0_branch``: [0-orchestrator](https://github.com/zero-os/0-orchestrator) branch to be used in the creation of 0-stor nodes.
- ``number_of_servers``: number of 0-stor servers te be created in the object cluster.
- ``number_of_machines`` : number of 0-stor nodes to be created on packet.net.

- ``TEST_CASE``: testcases directory (for example: ``test_suite/test_cases/basic_tests``).
- ``default_config_path``: default configration file path (default: ``/gopath/src/github.com/zero-os/0-stor/cmd/zstor/config.yaml``)
- ``number_of_files``: number of random files to be used in uploading/downloading test cases.
- ``iyo_user2_id``: itsyou.online user_2 client id
- ``iyo_user2_secret``: itsyou.online user_2 secret
- ``iyo_user2_username``: itsyou.online user_2 username

### Trigger build from travis
- You can check the default values of the enviroment variables [here](https://travis-ci.org/zero-os/0-stor/settings)
- Go to https://travis-ci.org/zero-os/0-stor and select **more options** then trigger build.
or go to https://travis-dash.gig.tech and trigger from the ci-dashboard

### Trigger build manually
- Export the above environment variables in the shell.
- Execute the following commands:
```bash

## install requirements
sudo apt install git -y
pip3 install -U "git+https://github.com/zero-os/0-core.git@${core_0_branch}#subdirectory=client/py-client"
pip3 install -U "git+https://github.com/zero-os/0-orchestrator.git@${orchestrator_branch}#subdirectory=pyclient"
curl -s https://install.zerotier.com/ | sudo bash
pip3 install -r test/deploy_test_env/requirements.txt

git clone https://github.com/zero-os/0-stor; cd 0-stor

## Install the environment
bash test/deploy_test_env/run_tests.sh before 
## Run tests
bash test/deploy_test_env/run_tests.sh test 
## Teardown
bash test/deploy_test_env/run_tests.sh after 
```