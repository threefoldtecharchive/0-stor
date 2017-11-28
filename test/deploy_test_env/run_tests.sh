action=$1

export container_name="0-stor-test-${TRAVIS_BUILD_NUMBER}"
export ays_repo="ays-repo-${TRAVIS_BUILD_NUMBER}"

generate_jwt(){
    export JWT=$(curl -X POST "https://itsyou.online/v1/oauth/access_token?grant_type=client_credentials&client_id=${iyo_client_id}&client_secret=${iyo_secret}&response_type=id_token&scope=user%3Amemberof%3A${iyo_organization},offline_access&validity=3600")
}

create_ays_repo_on_github(){
    curl -H "Content-Type: application/json" -H "Authorization: Bearer ${github_token}" -X POST -d '{"name":"'"${ays_repo}"'","description":"ays repo"}' https://api.github.com/user/repos
}

delete_ays_repo_on_github(){
    curl -H "Authorization: Bearer ${github_token}" -X DELETE https://api.github.com/repos/gigqa/${ays_repo}
}

add_sshkey_to_github(){
    curl -H "Content-Type: application/json" -H "Authorization: Bearer ${github_token}" \
    -X POST -d '{"title": "'${RANDOM}'", "key": "'"${1}"'"}' https://api.github.com/repos/gigqa/${ays_repo}/keys
}

join_zerotier_network(){
    sudo zerotier-cli join ${1}; sleep 5
    member_id=$(sudo zerotier-cli info | awk '{print $3}')
    curl -H "Content-Type: application/json" -H "Authorization: Bearer ${2}" \
    -X POST -d '{"config": {"authorized": true}}' https://my.zerotier.com/api/network/${1}/member/${member_id} >> /dev/null
}

install_client(){
    export GOPATH=/gopath
    export GOROOT=/usr/local/go
    export PATH=/usr/local/go/bin:$PATH
    mkdir -p /gopath/src/github.com
    cp -ra /home/travis/build/zero-os /gopath/src/github.com
    cd /gopath/src/github.com/zero-os/0-stor
    git config remote.origin.fetch +refs/heads/*:refs/remotes/origin/*
    git fetch
    git checkout -f ${ZERO_STOR_CLIENT_BRANCH}
    git branch
    echo " [*] Install zerostor client from branch : ${ZERO_STOR_CLIENT_BRANCH}"
    make cli
    chmod 777 /gopath/src/github.com/zero-os/0-stor/cmd/zstor/config.yaml
    ln -sf /gopath/src/github.com/zero-os/0-stor/bin/zerostorcli /bin/zerostorcli
    hash -r
    zerostorcli --version
    cd /home/travis/build/zero-os/0-stor
}

install_go_1_9(){
    rm -rf /usr/local/go
    curl https://storage.googleapis.com/golang/go1.9.linux-amd64.tar.gz > go1.9.linux-amd64.tar.gz
    tar -C /usr/local -xzf go1.9.linux-amd64.tar.gz
    mkdir /gopath
    echo "export GOPATH=/gopath" >> ~/.bash_profile
    echo "export GOROOT=/usr/local/go" >> ~/.bash_profile
    echo "export PATH=/usr/local/go/bin:$PATH" >> ~/.bash_profile
    source ~/.bash_profile
}

create_0_stor_flist(){
    git clone https://github.com/zero-os/0-orchestrator.git /home/0-orchestrator
    cd /home/0-orchestrator
    git checkout ${orchestrator_branch}
    cd -
    export GOPATH=/gopath
    export GOROOT=/usr/local/go
    export PATH=/usr/local/go/bin:$PATH
    go version
    echo "[*] Execute builder-0-stor.sh"
    bash /home/0-orchestrator/buildscripts/builder-0-stor.sh ${ZERO_STORE_SERVER_BRANCH}
    ls /tmp/archives/
}

if [ "$TRAVIS_EVENT_TYPE" == "cron" ] || [ "$TRAVIS_EVENT_TYPE" == "api" ]; then

    if [ "$action" == "before" ]; then

        echo "[*] Install go"
        sudo bash -c "$(declare -f install_go_1_9); install_go_1_9"

        echo "[+] Create 0-stor flist from ${ZERO_STORE_SERVER_BRANCH} branch"
        sudo bash -c "$(declare -f create_0_stor_flist); create_0_stor_flist"
        python3 test/deploy_test_env/ZeroHubClient.py -f /tmp/archives/0-stor-${ZERO_STORE_SERVER_BRANCH}.tar.gz -i ${iyo_client_id} -s ${iyo_secret}
        zero_stor_flist=$(cat 0_stor_flist)
        echo "Zerostor flist : ${zero_stor_flist}"


        echo '{}' > /tmp/config.json

        echo "=================================================="
        echo "[+] Create ays repo on github"
        create_ays_repo_on_github >> /dev/null

        echo "[+] Generate ssh key"
        ssh-keygen -f ~/.ssh/id_rsa -t rsa -N ''
        eval `ssh-agent` && ssh-add
        public_ssh_key=$(cat ~/.ssh/id_rsa.pub)
        private_ssh_key=$(cat ~/.ssh/id_rsa)
        add_sshkey_to_github "${public_ssh_key}" >> /dev/null

        echo "[+] Installing 0-stor client ..."
        travis_dir=$(pwd)
        sudo bash -c "$(declare -f install_client); install_client"
        pwd

        echo "[+] Create zerotier networks"
        export container_zerotier_network=$(python3 test/deploy_test_env/utils.py create_network 'container-zerotier-network')
        echo "[*] Container Zerotier Network ID : ${container_zerotier_network}"
        export nodes_zerotier_network=$(python3 test/deploy_test_env/utils.py create_network 'nodes-zerotier-network')
        echo "[*] Nodes Zerotier Network ID : ${nodes_zerotier_network}"

        ### temporary fix
        export nodes_zerotier_network=${container_zerotier_network}

        if [[ "${zero_os_zerotier_token}" == "" ]]; then
            zero_os_zerotier_token=${zerotier_token}
        fi

        join_zerotier_network ${zero_os_zerotier_network} ${zero_os_zerotier_token}; sleep 3
        join_zerotier_network ${container_zerotier_network} ${zerotier_token}; sleep 3

        echo "[+] Creating Zero-OS nodes on packet.net"
        python3 test/deploy_test_env/utils.py create_nodes

        echo "[+] Start zerotier auto authorization service"
        python3 test/deploy_test_env/utils.py zerotier_auth_service ${container_zerotier_network} &

        generate_jwt

        echo "[+] Deploying orchestrator from flist"
        curl -o /tmp/autobootstrap.py https://raw.githubusercontent.com/zero-os/0-orchestrator/${orchestrator_branch}/autosetup/autobootstrap.py
        python3 /tmp/autobootstrap.py --server ${server_ip} --password "${JWT}" --container ${container_name} --zt-net ${container_zerotier_network} \
        --upstream "git@github.com:gigqa/${ays_repo}.git" --organization ${iyo_organization} --client-id ${iyo_client_id} --client-secret ${iyo_secret} \
        --network "packet" --nodes-zt-net ${nodes_zerotier_network} --nodes-zt-token ${zerotier_token} --stor-namespace ${iyo_namespace} --ssh-key "$HOME/.ssh/id_rsa" 2>&1 | tee /tmp/autosetup.log

        echo "[+] Stop zerotier auto authorization service"
        pkill -9 python3 >> /dev/null

        echo "[+] Deploying etcd cluster and object cluster"
        container_ip=$(cat /tmp/autosetup.log | grep "container address:" | cut -d: -f2 )
        scp -o StrictHostKeyChecking=no test/deploy_test_env/js_docker_script.sh root@"${container_ip:1}":
        ssh -t -o StrictHostKeyChecking=no root@"${container_ip:1}" bash js_docker_script.sh ${iyo_client_id} ${iyo_secret} ${iyo_organization} ${iyo_namespace} ${number_of_machines} ${number_of_servers} ${zero_stor_flist}

        echo "[+] Update 0-stor client config file"
        zerostor_servers_ips=$(ssh root@"${container_ip:1}" cat /tmp/zerostor_servers_ips)
        etcd_servers_ips=$(ssh root@"${container_ip:1}" cat /tmp/etcd_servers_ips)
        python3 test/deploy_test_env/utils.py update_config "${zerostor_servers_ips}" "${etcd_servers_ips}"
        cat /gopath/src/github.com/zero-os/0-stor/cmd/zstor/config.yaml


    elif [ "$action" == "test" ]; then
        echo " [*] Execute test case"
        cat /gopath/src/github.com/zero-os/0-stor/cmd/zstor/config.yaml
        cd test && git branch && export PYTHONPATH='./' && nosetests-3.4 -vs --logging-level=WARNING --progressive-with-bar --rednose $TEST_CASE --tc-file test_suite/config.ini --tc=main.number_of_servers:${number_of_servers} --tc=main.number_of_files:${number_of_files} --tc=main.default_config_path:${default_config_path} --tc=main.iyo_user2_id:${iyo_user2_id} --tc=main.iyo_user2_secret:${iyo_user2_secret} --tc=main.iyo_user2_username:${iyo_user2_username}

    elif [ "$action" == "after" ]; then
        generate_jwt
        python3 test/deploy_test_env/utils.py teardown
        echo "[*] Deleting ays repo: ${ays_repo}"
        delete_ays_repo_on_github    
    fi
fi