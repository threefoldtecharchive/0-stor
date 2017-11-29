import packet, yaml, sys, random, json, time, os, requests, zerotier
from zeroos.core0.client import Client as core0_client

class PacketDotNet():
    def __init__(self, token):
        self.manager = packet.Manager(auth_token=token)
        self.project = self.manager.list_projects()[0]
    
    def get_available_facility(self, plan):
        facilities = [x.code for x in self.manager.list_facilities()]
        for facility in facilities:
            if self.manager.validate_capacity([(facility, plan, 1)]):
                return facility
        else:
            return None
    
    def create_machine(self, hostname, plan='baremetal_2'):
        ipxe_script_url = 'https://bootstrap.gig.tech/ipxe/{}/{}/organization={}'.format(branch, zerotier_network, iyo_organization)
        facility = self.get_available_facility(plan)
        device = self.manager.create_device(project_id=self.project.id,
                                            hostname=hostname,
                                            plan=plan,
                                            operating_system='custom_ipxe',
                                            ipxe_script_url=ipxe_script_url,
                                            facility=facility)
        return device
    
    def delete_machine(self, device_id):
        self.manager.call_api('devices/%s' % device_id, type='DELETE')

class Zerotier:
    def __init__(self, token):
        self.token = token
        self.client = zerotier.APIClient()
        self.client.set_auth_header('Bearer {}'.format(self.token))

    def create_network(self, name):
        data = {'config': {
            'v4AssignMode': {'zt': 'true'},
	        'ipAssignmentPools': [{'ipRangeStart': '10.147.17.1', 'ipRangeEnd': '10.147.17.254'}],
	        'name': name,
	        'routes': [{'target': '10.147.17.0/24','via': None}]}}
        
        response = self.client.network.createNetwork(data=data)
        return response.json()['id']

    def delete_network(self, networkid):
        self.client.network.deleteNetwork(networkid)


    def authorize_members_service(self, networkid):
        while True:
            members = self.client.network.listMembers(id=networkid).json()
            for member in members:
                if not member['config']['authorized']:
                    data = {"config": {"authorized": True}}
                    self.client.network.updateMember(id=networkid, address=member['nodeId'], data=data)
            else:
                time.sleep(10)


class Config:
    def __init__(self):
        self.path = '/tmp/config.json'

    def read(self):
        with open(self.path, 'r') as f:
            return json.load(f)

    def save(self, ** kwargs):
        config = self.read()

        for key, value in kwargs.items():
            config[key] = value

        with open(self.path, 'w') as f:
            json.dump(config, f)


if __name__  == '__main__':
    
    config = Config()
    packet_client = PacketDotNet(token=os.environ['packet_token'])
    zerotier_client = Zerotier(token=os.environ['zerotier_token'])

    action = sys.argv[1]
    build_id = os.environ['TRAVIS_BUILD_NUMBER']
    
    if action == 'create_nodes':
        branch = os.environ['core_0_branch']
        zerotier_network = os.environ['nodes_zerotier_network']
        iyo_organization = os.environ['iyo_organization']
        number_of_machines = os.environ['number_of_machines']
        
        nodes = []
        for i in range(int(number_of_machines)):
            hostname = '0-stor-node-{}-{}'.format(i, build_id)
            device = packet_client.create_machine(hostname=hostname)
            nodes.append(device.id)
        else:
            config.save(nodes=nodes)


    elif action == 'create_network':
        zerotier_network_name = sys.argv[2]
        zerotiers_list = []
        networkid = zerotier_client.create_network(name=zerotier_network_name)
        config_data = config.read()
        
        if 'zerotiers' in config_data:
            zerotiers_list = config_data['zerotiers']

        zerotiers_list.append(networkid)
        config.save(zerotiers=zerotiers_list)
        print(networkid)


    elif action == 'zerotier_auth_service':
        networkid = sys.argv[2]
        zerotier_client.authorize_members_service(networkid=networkid)


    elif action == 'teardown':
        config_data = config.read()
        for networkid in config_data['zerotiers']:
            print('[+] Deleting zerotier network: {}'.format(networkid))
            zerotier_client.delete_network(networkid)

        for nodeid in config_data['nodes']:
            print('[+] Deleting node: {}'.format(nodeid))
            packet_client.delete_machine(nodeid)

        jwt = os.environ['JWT']                
        ip = os.environ['server_ip']
        container_name = os.environ['container_name']
        
        print('[*] Deleting orchestrator container: {}'.format(container_name))
        client = core0_client(ip, password=jwt)
        container_id = list(client.container.find(container_name).keys())[-1]
        client.container.terminate(int(container_id))



    elif action == 'update_config':
        zerostor_servers = sys.argv[2]
        etcd_servers = sys.argv[3]

        config_path = '/gopath/src/github.com/zero-os/0-stor/cmd/zstor/config.yaml'
        with open(config_path, 'r') as f:
            config = yaml.load(f)

        number_of_servers = os.environ['number_of_servers']
        config['distribution_parity'] = 1
        config['distribution_data'] = int(number_of_servers) -1 
        config['replication_nr'] = int(number_of_servers)
        config['organization'] = os.environ['iyo_organization']
        config['namespace'] = os.environ['iyo_namespace']
        config['iyo_app_id'] = os.environ['iyo_client_id']
        config['iyo_app_secret'] = os.environ['iyo_secret']
        config['data_shards'] = zerostor_servers.strip().split(' ')
        config['meta_shards'] = ['http://{}'.format(x) for x in etcd_servers.strip().split(' ')]
        
        with open(config_path, 'w') as f:
            yaml.dump(config, f, default_flow_style=False)
