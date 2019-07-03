# Copyright (C) 2017-2018 GIG Technology NV and Contributors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from setuptools import setup, find_packages
# To use a consistent encoding
from codecs import open
from os import path

here = path.abspath(path.dirname(__file__))

# Get the long description from the README file
with open(path.join(here, 'README.md'), encoding='utf-8') as f:
    long_description = f.read()

setup (
    name='pydaemon',
    version='1.0',
    description='Python client for 0-stor client daemon',
    long_description=long_description,
    url='https://github.com/threefoldtech/0-stor',
    author='Muhamad Azmy',
    author_email='muhamada@greenitglobe.com',
    license='Apache 2.0',
    packages=find_packages(),
    install_requires=[
        'grpcio>=1.8.3',
        'grpcio-tools>=1.8.3',
    ],
)
