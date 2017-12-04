# 0-stor test suite

## Test suite architecture:
```bash
.
├── deploy_zstor_in_travis
│   ├── manual_deployment.sh
│   ├── requirements.txt
│   └── run_tests.sh
├── README.md
└── test_suite
    ├── config.ini
    ├── framework
    │   ├── __init__.py
    │   ├── iyo_client
    │   │   ├── base.py
    │   │   ├── client_jwt.py
    │   │   ├── client.py
    │   │   ├── client_utils.py
    │   │   └── __init__.py
    │   └── zero_store_cli.py
    ├── __init__.py
    ├── README.md
    └── test_cases
        ├── basic_tests
        │   ├── __init__.py
        │   ├── test01_basic_tests.py
        │   └── test02_pipeplining.py
        ├── extend_test
        │   └── __init__.py
        └── testcases_base.py

```

## Run Test suite:

You have to install an environment to run this test suite against it. You can install it manually or trigger it from the Travis dashboard.

## 1- Travis Environment:
**Hint:**
Triggering it with the default prameters values will use $TRAVIS_BRANCH to install zstor and run all test cases agains it.

Here is the travis pramaters description to fire the test suite:
```yaml
TEST_CASE=test_suite/test_cases
iyo_organization= <IYO user1 organization>
iyo_namespace= <IYO user1 namespace>
iyo_client_id= <IYO user1 client ID>
iyo_secret= <IYO user1 client secret>
number_of_files= <Any integer value>
iyo_user2_username= <IYO user1 mail>
iyo_user2_id= <IYO user2 client ID>
iyo_user2_secret= <IYO user2 client secret>
number_of_servers= <Any integer value>
default_config_path=/gopath/src/github.com/zero-os/0-stor/cmd/zstor/config.yaml
ZSTORDB_BRANCH= <Branch to make zstordb and zsotr binaries>
TESTCASE_BRANCH= <Branch to excute test cases>
```



## 2- Local Environment:
**Hint:**
You should have two IYO accounts, One as a master which will be used in the zerostor config file and the other as a slave which will be used in the test suite config file.

**1- Installing the local environment.** There is a bash script to automate the installing process. It will install all zstordb dependencies, run one etcd, run <NUMBER_OF_SERVERS> zstordb and edit the zstor config file. To run it `cd` to the repo directory then:

```bash
➜  0-stor git:(master) ✗ bash test/deploy_zstor_in_travis/manual_deployment.sh -h
This script will to automate the installing process.
It will install all zstordb dependencies, run one etcd,
run <NUMBER_OF_SERVERS> zstordb and edit the zstor config file.

Usage:
    bash manual_deployment.sh <NUMBER_OF_SERVERS> <IYO_APP_ID> <IYO_APP_SECRET> <ORGANIZATION> <NAMESPACE> <ZSTORDB_BRANCH> <TESTCASE_BRANCH>

Parameters:
    NUMBER_OF_SERVERS = Number of running zstordb
    IYO_APP_ID = IYO user1 client ID
    IYO_APP_SECRET = IYO user1 client secret
    ORGANIZATION = IYO user1 organization
    NAMESPACE = IYO user1 namespace
    ZSTORDB_BRANCH = Branch to make zstordb and zsotr binaries
    TESTCASE_BRANCH = Branch to excute test cases           

```

**2- Configure the test suite config file**, `cd` to the repo directory then `vim zero-os/0-stor/test/test_suite/config.ini`

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

 **3- Install the testsuite requirements,** `cd` to the repo directory then,
```bash
pip3 install -r test/deploy_zstor_in_travis/requirements.txt
```

**4- Fire test suite**
```bash
export PYTHONPATH='./'
nosetests-3.4 -vs test_suite/test_cases --tc-file test_suite/config.ini
```

Example: 
```bash
nosetests-3.4 -vs test_suite/test_cases/basic_tests/test01_basic_tests.py:BasicTestCases.test001_upload_download_files --tc-file test_suite/config.ini --tc=main.number_of_files:10
````
This command will execute the test001_update_file test case with number_of_files=10