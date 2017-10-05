import requests
from bs4 import BeautifulSoup
import argparse


class ZeroHubClient:
    def __init__(self, jwt):
        self.baseurl = 'https://hub.gig.tech'
        self.cookies = dict(caddyoauth=jwt)

    def upload(self, filename):
        files = {'file': open(filename, 'rb')}
        r = requests.post('%s/upload' % self.baseurl, files=files, cookies=self.cookies)
        soup = BeautifulSoup(r.text, 'lxml')
        source = str(soup.findAll('code')[1])
        return source[source.index('https'):source.index('</')]

    def merge(self, sources, target):
        arguments = []

        for source in sources:
            arguments.append(('flists[]', source))

        arguments.append(('name', target))
        r = requests.post('%s/merge' % self.baseurl, data=arguments, cookies=self.cookies)
        print(r.text)

        return True


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='ZeroHub Client')
    parser.add_argument('-f', '--file', help='local file path', required=True)
    parser.add_argument('-i', '--client_id', help='Client ID', required=True)
    parser.add_argument('-s', '--client_secret', help='Client secret', required=True)
    args = vars(parser.parse_args())

    localfile = args['file']
    clientid = args['client_id']
    clientsecret = args['client_secret']
    response = requests.post(
        'https://itsyou.online/v1/oauth/access_token?grant_type=client_credentials&client_id=%s&client_secret=%s&response_type=id_token' % (
            clientid, clientsecret))

    if response.status_code != 200:
        raise RuntimeError("Authentification failed")

    jwt = response.content.decode('utf-8')

    client = ZeroHubClient(jwt)

    print(" [+] uploading source file")
    source_link = client.upload(localfile)
    print(' [+] %s ' % source_link)

    with open('0_stor_flist', 'w') as file:
        file.write(source_link)
