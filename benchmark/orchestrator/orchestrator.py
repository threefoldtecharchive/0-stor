#!/usr/bin/python3

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

"""
    Orchestrator controls the running of benchmarking process,
    aggregating results and producing report.
"""

import sys
import signal
from argparse import ArgumentParser
from lib import Config
from lib import Report


def handler(signum, frame):
    """ Handler for all SIGTSTP signals """
    raise KeyboardInterrupt

def main(argv):
    """ main function of the benchmarker """

    # parse arguments
    parser = ArgumentParser(epilog="""
        Orchestrator controls the benchmarking process,
        aggregating results and producing report.
    """, add_help=False)
    parser.add_argument('-h', '--help',
                        action='help',
                        help='help for orchestrator')
    parser.add_argument('-C',
                        '--conf',
                        metavar='string',
                        default='bench_config.yaml',
                        help='path to the config file (default bench_config.yaml)')
    parser.add_argument('--out',
                        metavar='string',
                        default='report',
                        help='directory where the benchmark report will be written (default ./report)')

    args = parser.parse_args()
    input_config = args.conf
    report_directory = args.out

    # path where config for scenarios is written
    output_config = "scenarios_config.yaml"
    # path to the benchmark results
    result_benchmark_file = "benchmark_result.yaml"

    print('********************')
    print('****Benchmarking****')
    print('********************')

    # Catch SIGTSTP signals
    signal.signal(signal.SIGTSTP, handler)

    # extract config information
    config = Config(input_config)

    # initialise report opject
    report = Report(report_directory)

    # loop over all given benchmarks
    try:
        while True:
            # switch to the next benchmark config
            benchmark = next(config.benchmark)

            # define a new data collection
            report.init_aggregator(benchmark)

            # loop over range of the secondary parameter
            for val_second in benchmark.second.range:

                report.aggregator.new()

                # alter the template config if secondary parameter is given
                if not benchmark.second.empty():
                    config.alter_template(benchmark.second.id, val_second)

                # loop over the prime parameter
                for val_prime in benchmark.prime.range:
                    # alter the template config if prime parameter is given
                    if not benchmark.prime.empty():
                        config.alter_template(benchmark.prime.id, val_prime)

                    try:
                        zstordb_prof_dir, client_prof_dir = config.new_profile_dir(report_directory)

                        # deploy zstor
                        config.deploy_zstor(profile_dir=zstordb_prof_dir)

                        # update config file
                        config.save(output_config)

                        # perform benchmarking
                        config.run_benchmark(config=output_config, 
                                                out=result_benchmark_file, 
                                                profile_dir=client_prof_dir)                        

                        # stop zstor
                        config.stop_zstor()
                    except:
                        config.stop_zstor()
                        raise
                    # fetch results of the benchmark
                    report.fetch_benchmark_output(result_benchmark_file)
                    
                    report.add_timeplot() # add timeplots to the report

            # add results of the benchmarking to the report
            report.add_aggregation()
            config.restore_template()
    except StopIteration:
        print("Benchmarking is done")

if __name__ == '__main__':
    main(sys.argv[1:])
