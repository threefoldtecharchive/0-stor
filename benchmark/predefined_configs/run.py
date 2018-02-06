#!/usr/local/bin/python3

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

'''
This script will loop all subdirectories and look for benchmark config files (ending with 'bench.yaml')
It will put the results in a directory with the file name of the config appended with '_result'
If the orchestrator.py script returns a permission denied error, try add execution rights to the orchestrator script.
'''
import sys, os

_ORCH_LOCATION = os.path.normpath(
    os.path.join(sys.path[0], "..", "orchestrator", "orchestrator.py"))
_BENCH_FILE_SUFFIX = "bench.yaml"

def main(argv):
    configdir = sys.argv[1] if len(sys.argv) > 1 else "."
    dirlist = os.listdir(configdir)

    for d in dirlist:
        if not os.path.isdir(d):
            continue

        # list subdir and check for config files
        subdirlist =  os.listdir(d)
        for f in subdirlist:
            conf_file = os.path.join(d, f)
            # check file suffix
            if not f.endswith(_BENCH_FILE_SUFFIX):
                continue
            
            # remove '.yaml' to get result dir
            result_dir = conf_file[:-5] + "_result"

            # run benchmark
            print("running benchmark using config %s"%conf_file)
            os.system("%s -C %s --out %s" % (_ORCH_LOCATION, conf_file, result_dir))

if __name__ == '__main__':
    main(sys.argv[1:])
