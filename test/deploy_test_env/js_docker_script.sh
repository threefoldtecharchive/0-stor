client_id=${1}
client_secret=${2}
organization=${3}
namespace=${4}
machines=${5}
servers=${6}
zero_stor_flist=${7}

export_jwt(){
    jwt=$(ays generatetoken --clientid ${client_id} --clientsecret ${client_secret} --organization ${organization} --validity 3600)
    eval $jwt
}

create_object_cluster_blueprint(){
cat > /optvar/cockpit_repos/orchestrator-server/blueprints/object_cluster.bp << EOL
storage_cluster__myobjectcluster:
  label: myobjectcluster
  nrServer: ${servers}
  diskType: ssd
  metadiskType: ssd
  clusterType: object
  nodes:
EOL

for node in ${1//" "/}
do
    echo "  - '${node}'" >> /optvar/cockpit_repos/orchestrator-server/blueprints/object_cluster.bp
done

cat >> /optvar/cockpit_repos/orchestrator-server/blueprints/object_cluster.bp << EOL
  dataShards: 1
  parityShards: 1

actions:
  - action: install
EOL

}

create_etcd_cluster_blueprint(){
cat > /optvar/cockpit_repos/orchestrator-server/blueprints/etcd_cluster.bp << EOL
etcd_cluster__myetcdcluster:
  nodes:
EOL

for node in ${1//" "/}
do
    echo "  - '${node}'" >> /optvar/cockpit_repos/orchestrator-server/blueprints/etcd_cluster.bp
done

cat >> /optvar/cockpit_repos/orchestrator-server/blueprints/etcd_cluster.bp << EOL
actions:
  - action: install
    actor: etcd_cluster
EOL
}

update_configuration_bp(){
 #sed -i -e "s,https://hub.gig.tech/gig-official-apps/0-stor-master.flist,${zero_stor_flist},g" /optvar/cockpit_repos/orchestrator-server/blueprints/configuration.bp
 echo "  - {key: 0-stor-flist, value: '${zero_stor_flist}'}" >> /optvar/cockpit_repos/orchestrator-server/blueprints/configuration.bp
 cat /optvar/cockpit_repos/orchestrator-server/blueprints/configuration.bp
}

cd /optvar/cockpit_repos/orchestrator-server

echo "[*] Update configuration blueprint."
update_configuration_bp

while [ $(ls services | grep node! | wc -l) != ${machines} ]
do
  echo "Watting for nodes to be ready"
  sleep 5
done
sleep 60

export HOME=/root
export LC_LANG=UTF-8
export LC_ALL=C.UTF-8
export PATH=/opt/jumpscale9/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
export PYTHONPATH=/opt/jumpscale9/lib/:/opt/code/github/jumpscale/core9/:/opt/code/github/jumpscale/prefab9/:/opt/code/github/jumpscale/ays9:/opt/code/github/jumpscale/lib9:/opt/code/github/jumpscale/portal9

export_jwt

nodes=$(ls services | grep node! | cut -d! -f2)

ays blueprint configuration.bp

create_etcd_cluster_blueprint "${nodes}"
create_object_cluster_blueprint "${nodes}"

ays blueprint etcd_cluster.bp
ays blueprint object_cluster.bp
ays run create -fy

etcd_servers_ips=$(ays service show -r etcd | grep clientBind | cut -d: -f2-)
echo ${etcd_servers_ips} > /tmp/etcd_servers_ips

zerostor_servers_ips=$(ays service show -r zerostor | grep bind | cut -d: -f2-)
echo ${zerostor_servers_ips} > /tmp/zerostor_servers_ips

echo "zerostor servers : "
cat /tmp/zerostor_servers_ips
echo "zerostor servers : "
cat /tmp/etcd_servers_ips
