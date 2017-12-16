#!/usr/bin/env bash
action=$1
if [ "$TRAVIS_EVENT_TYPE" == "cron" ] || [ "$TRAVIS_EVENT_TYPE" == "api" ]; then
    if [ "$action" == "before" ]; then
        echo "[+] Deploy zstordb servers"
        sudo bash test/deploy_zstor_in_travis/manual_deployment.sh ${number_of_servers} ${iyo_client_id} ${iyo_secret} ${iyo_organization} ${iyo_namespace} ${ZSTORDB_BRANCH} ${TESTCASE_BRANCH}
    elif [ "$action" == "test" ]; then
        echo " [*] Execute test case"
        cd test && export PYTHONPATH='./' && nosetests-3.4 -vs --logging-level=WARNING --progressive-with-bar --rednose $TEST_CASE --tc-file test_suite/config.ini --tc=main.number_of_servers:${number_of_servers} --tc=main.number_of_files:${number_of_files} --tc=main.default_config_path:${default_config_path} --tc=main.iyo_user2_id:${iyo_user2_id} --tc=main.iyo_user2_secret:${iyo_user2_secret} --tc=main.iyo_user2_username:${iyo_user2_username}
    fi
fi