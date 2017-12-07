# 0-stor test suite

## Test suite architecture:
```bash
.
├── deploy_test_env
│   ├── js_docker_script.sh
│   ├── README.md
│   ├── requirements.txt
│   ├── run_tests.sh
│   ├── utils.py
│   └── ZeroHubClient.py
├── README.md
├── test_suite
│   ├── config.ini
│   ├── framework
│   │   ├── __init__.py
│   │   ├── iyo_client
│   │   │   ├── base.py
│   │   │   ├── client_jwt.py
│   │   │   ├── client.py
│   │   │   ├── client_utils.py
│   │   │   └── __init__.py
│   │   └── zero_store_cli.py
│   ├── __init__.py
│   ├── README.md
│   └── test_cases
│       ├── basic_tests
│       │   ├── __init__.py
│       │   ├── test01_basic_tests.py
│       │   └── test02_pipeplining.py
│       ├── extend_test
│       │   └── __init__.py
│       └── testcases_base.py
└── test_suite.log
```

## Run Test suite:

You have to install an environment to run this test suite against it. You can install it manually or trigger it from the Travis dashboard.

- Local Environment:

At least you should run 2 zstordb , then make the `zstor` binary available in your $PATH.

- Travis Environment:

Here is a full documentation about the deploying the environment 
https://github.com/zero-os/0-stor/blob/master/test/deploy_test_env/README.md



- Running the test suite manually in the local environment:
1- Install your local environment and ensure the `zstor` is installed and available in your $PATH

**Hint:**
You should have two IYO accounts, One as a master which will be used in the zerostor config file and the other as a slave which will be used in the test suite config file.

2- Configure the zerostor config file

```yaml
block_size: 4096
compress: true
data_shards: #<List of zstordb>
- '172.17.0.2:8080' 
- '172.17.0.3:8080'
distribution_data: 1
distribution_parity: 1
encrypt: true
encrypt_key: <Encryption key>
iyo_app_id: <IYO user1 clied ID>
iyo_app_secret: <IYO user1 clied secret>
meta_shards: #<ETCD cluster>
- http://172.17.0.3:2379
namespace: <IYO user1 namespace>
organization: <IYO user1 organization>
replication_max_size: 4096
replication_nr: 2
```

3- Configure the test suite config file, `cd` to the repo directory then `vim zero-os/0-stor/test/test_suite/config.ini`

```bash
[main]
number_of_servers = <number or running zstordb>
number_of_files = <any integer value>
default_config_path = <zerostor config file path>
iyo_url = https://itsyou.online/
iyo_user2_id = <IYO user2 client ID>
iyo_user2_secret = <IYO user2 client secret>
iyo_user2_username = <IYO user2 email>
```

 4- Install requirements, `cd` to the repo directory then,
```bash
cd 0-stor/test
pip3 install -r deploy_test_env/requirements.txt
```

5- Fire test suite
```bash
export PYTHONPATH='./'
nosetests-3.4 -vs test_suite/test_cases --tc-file test_suite/config.ini
```

Example: 
```bash
nosetests-3.4 -vs test_suite/test_cases/basic_tests/test01_upload_download.py:UploadDownload.test001_upload_file --tc-file test_suite/config.ini --tc=main.number_of_files:10
````
This command will execute the test001_update_file test case with number_of_files=10