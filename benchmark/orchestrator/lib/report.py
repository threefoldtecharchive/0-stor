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
    Package report contains tools for collecting results of the benchmarking 
    and generating final report. 

    Output files show benchmarking results by means of tables and figures.
    Report for each benchmark is added to the output files as soon as the 
    benchmark is finished.

    Two types of output files are avaliable: main report file and a timeplot
    collection. Main report file consists of performance measures and allows 
    for performance comparision among various scenarios.

    The performance metric, reflected in the report is throughput. 
    Throughput is an average data rate observed throughout the benchmark.

    Timeplot collection contains the scope of timeplots collected during 
    the benchmarking. Timeplots show number of operations such as reads 
    or writes per time unit, observed during the benchmark.
"""
import os
import matplotlib
# fixes: _tkinter.TclError: no display name and no $DISPLAY environment variable
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import yaml
import re
import humanize
from lib.scenario import Scenario
from lib.scenario import InvalidBenchmarkResult
from lib.scenario import TIME_UNITS
from lib.scenario import filter_dict

FILTER_KEYS = {'organization',
                'namespace',
                'iyo',
                'shards',
                'db',
                'hashing',
                'metastor',
                'datastor',
                'tls'}
BYTE_KEYS = { 'value_size', 'block_size', 'key_size'}

class Aggregator:
    """ Aggregator aggregates average throughput over a set of benchmarks """

    def __init__(self, benchmark=None):
        self.benchmark = benchmark
        self.throughput = []

    def new(self):
        self.throughput.append([])

class Report:
    """
    Class Report is used to collect results of benchmarking
        and to create final report.
    """
    def __init__(self, directory='report', report='report.md', timeplots='timeplots.md'):
        self.directory = directory # set output directory for report files
        self.main_file = "{0}/{1}".format(self.directory, report)
        self.timeplots_collection = "{0}/{1}".format(self.directory, timeplots)

        if not os.path.exists(self.directory):
            os.makedirs(self.directory)

        with open(self.main_file, 'w+') as outfile:
            outfile.write("# Benchmark report\n")
            outfile.write("[Timeplot collection is here]({0})\n".format(timeplots))

        with open(self.timeplots_collection, 'w+') as outfile:
            outfile.write("# Timeplot collection report\n")
            outfile.write("[Main report in here]({0}) \n\n".format(report))

        self.reports_added = 0   # keep track of added reports

        self.timeplots_added = 0 # keep track of number of timeplots added

        self.scenario = Scenario()

    def fetch_benchmark_output(self, input_file):
        """ Fetch new results from @input_file """

        self.scenario.load_result(input_file)
        filter_dict(self.scenario.scenario, FILTER_KEYS)
        self.aggregate()

    def init_aggregator(self, benchmark=None):
        self.aggregator = Aggregator(benchmark)

    def aggregate(self, benchmark=None):
        """" Init iterator for benchmark scenarios """
        
        #self.aggregator = Aggregator(benchmark)
        th = self._get_throughput() # calculate throughput

        if self.aggregator.throughput:
            self.aggregator.throughput[-1].append(th)
        else:
            self.aggregator.throughput.append(th)

    def _get_throughput(self):
        """ Calculate throughput of the benchmark """
        throughput = 0
        for result in self.scenario.results:
            # get duration of the benchmarking
            try:
                duration = float(result['duration'])
            except:
                raise InvalidBenchmarkResult('duration format is not valid')

            # number of operations in the benchmarking
            try:
                count = int(result['count'])
            except:
                raise InvalidBenchmarkResult('count is not given, or format is not int')

            # get size of each value to calculate the throughput
            try:
                value_size = int(self.scenario.bench_config['value_size'])
            except:
                raise InvalidBenchmarkResult('value size is not given, or format is not int')

            throughput += count*value_size/duration/len(self.scenario.results)

        return int(throughput)

    def add_aggregation(self):
        # count reports added to a report file
        self.reports_added += 1

        fig_name = 'fig' +str(self.reports_added) + '.png'

        # filter results form scenario config before dumping to the report

        with open(self.main_file, 'a+') as outfile:
            # refer the figure in the report
            outfile.write("\n # Report {0} \n".format(str(self.reports_added)))

            # add benchmark config
            outfile.write('**Benchmark config:** \n')
            outfile.write('```yaml \n')
            yaml.dump(self.scenario.config, outfile, default_flow_style=False)
            outfile.write('\n```')

        # check if more then one output was collected
        if sum(map(len, self.aggregator.throughput)) > 1:
            # create a bar plot
            self._bar_plot( fig_name)

            # incerst bar plot to the report
            with open(self.main_file, 'a+') as outfile:
                outfile.write("\n![Fig: throughput vs parameter]({0})".format(fig_name))

        # add the table of the data sets
        self._add_table()

    @staticmethod
    def humanize_bitrate(value_in_bytes):
        ''' Shortens large values of bitrate to KiB/s or MiB/s '''

        byte, kbyte, mbyte = 'Byte/s', 'KiB/s', 'MiB/s', 
        dim = {byte: 1, kbyte: 1024, mbyte: 1048576}
        if value_in_bytes > dim[mbyte]:
            return value_in_bytes/dim[mbyte], dim[mbyte], mbyte

        if value_in_bytes > dim[kbyte]:
            return value_in_bytes/dim[kbyte], dim[kbyte], kbyte

        return value_in_bytes, dim[byte], byte

    def humanize_bytes(self, key, value):
        if key in BYTE_KEYS:
            return humanize.naturalsize(value,  binary=True)
        return value

    def _bar_plot(self, fig_name):
        # define range  from prime parameter
        prime_parameter = re.sub('[\n|...]', '', yaml.dump(self.aggregator.benchmark.prime.id))
        second_parameter = re.sub('[\n|...]', '', yaml.dump(self.aggregator.benchmark.second.id))

        ticks_labels = self.aggregator.benchmark.prime.range
        ticks_labels = [self.humanize_bytes(prime_parameter, tick) for tick in ticks_labels]

        # af first results are plot vs counting number of samples
        rng = [i for i, tmp in enumerate(ticks_labels)]

        # number of data sets combined in the figure
        if len(self.aggregator.throughput) == 0:
            raise InvalidBenchmarkResult("results are empty")

        max_throughput = 0
        for thr in self.aggregator.throughput:
            try:
                max_throughput = max(max_throughput, max(thr))
            except TypeError:
                max_throughput = max(self.aggregator.throughput)
        max_throughput, reduce_times, dim_name = self.humanize_bitrate(max_throughput)

        # figure settings
        n_plots = len(self.aggregator.throughput[0]) # number of plots in the figure
        n_samples = len(rng) # number of samples for each data set
        width = rng[-1]/(n_samples*n_plots+1) # bar width
        gap = width/10  # gap between bars
        diff_y = 0.1 # minimal relative difference in throughput between neighboring bars

        # create figure
        fig, ax = plt.subplots()

        # limmit number of ticks to the number of samples
        plt.xticks(rng)

        # substitute tick labels
        ax.set_xticklabels(ticks_labels)

        # define color cycle
        ax.set_color_cycle(['blue', 'red', 'green', 'yellow', 'black', 'brown'])

        ax.set_xlabel(prime_parameter)
        ax.set_ylabel('throughput, %s'%dim_name)

        # loop over data sets
        for i, th in enumerate(self.aggregator.throughput):
            # define plot label
            legend = " "
            if self.aggregator.benchmark.second.id:
                value = self.aggregator.benchmark.second.range[i]
                legend = "%s=%s"%(second_parameter,
                                  self.humanize_bytes(second_parameter, value))

            # add bar plot to the figure
            th_humanized = [t/reduce_times for t in th]
            ax.bar(rng, th_humanized, width, label=legend)

            # add space for a bar label on top of the plot
            plt.ylim(ymax=max_throughput*1.3)

            lgd = ax.legend(loc='upper left', bbox_to_anchor=(1, 1))

            # add labels to bars
            for j, v in enumerate(th):
                text_x = rng[j]
                text_y = max(0, v/reduce_times)
                va = 'bottom' # alinement of the bar label
                ax.text(text_x, text_y, ' %s/s '%humanize.naturalsize(v, gnu=True), color='black', fontweight='bold', rotation=90, ha='center',va=va)

            # shift bars for the next plot
            rng = [x+gap+width for x in rng]

        # label axes
        plt.savefig(os.path.join(self.directory,fig_name), 
                    bbox_extra_artists=(lgd,), 
                    bbox_inches='tight')
        plt.close()

    def _add_table(self):
        """ Add table with data """

        with open(self.main_file, 'a+') as outfile:
            # create a table
            prime_parameter = re.sub('[\n|.]', '', yaml.dump(self.aggregator.benchmark.prime.id))
            second_parameter = re.sub('[\n|.]','', yaml.dump(self.aggregator.benchmark.second.id))
            outfile.write('\n ### Throughput: \n')
            
            # add titles to the columns
            row_title = '| %s | '%prime_parameter
            row_line = '|---|'
            for item in self.aggregator.benchmark.second.range:
                if self.aggregator.benchmark.second.id:
                    if second_parameter in BYTE_KEYS:
                        item = humanize.naturalsize(item, binary=True)
                    row_title += '%s = %s |'%(second_parameter, item)
                else:
                    row_title += ' |'
                row_line += '---|'
            outfile.write("%s \n%s \n "%(row_title, row_line))
            # fill in the table
            for row, val in enumerate(self.aggregator.benchmark.prime.range):
                if prime_parameter in BYTE_KEYS:
                    val = humanize.naturalsize(val)                
                row_values = '| %s |'%val
                for col, _ in enumerate(self.aggregator.benchmark.second.range):
                    row_values +=  '%s/s |'%str(humanize.naturalsize(self.aggregator.throughput[col][row], binary=True))
                outfile.write('%s \n'%row_values)

    def add_timeplot(self):
        """ Add timeplots to the report """

        files = self._plot_per_interval()

        if files:
            with open(self.timeplots_collection, 'a+') as outfile:
                outfile.write("\n ## Timeplot %s \n"%(str(self.timeplots_added)))                
                outfile.write('\n**Config:**\n```yaml \n')

                yaml.dump(self.scenario.config, outfile, default_flow_style=False)
                yaml.dump({'results': self.scenario.results}, outfile, default_flow_style=False)
                outfile.write('\n```')
                outfile.write("\n _____________ \n")
                for file in files:
                    outfile.write("\n![Fig]({0}) \n".format(file))

    def _plot_per_interval(self):
        """ Create timeplots """
        file_names = [] # list of the output files

        # time_unit_literal represents the time unit for aggregation of the results
        time_unit_literal = self.scenario.result_output
        time_unit = TIME_UNITS.get(time_unit_literal)

        for result in self.scenario.results:
            # duration of the benchmarking
            try:
                duration = float(result['duration'])
            except:
                raise InvalidBenchmarkResult('duration format is not valid')

            # per_interval represents number of opperations per time unit
            per_interval = result.get('perinterval')

            # plot number of operations vs time if per_interval is not empty
            if per_interval:
                # define time samples
                max_time = min(int(duration), len(per_interval))
                time_line = [i for i in range(time_unit, max_time+time_unit)]

                plt.figure()
                plt.plot(time_line, per_interval[:len(time_line)], 'bo--', label=self.timeplots_added)
                plt.xlabel('time, '+time_unit_literal[4:])
                plt.ylabel('operations per '+time_unit_literal[4:])

                # define file name of the figure
                file_name = 'plot_per_interval_{0}.png'.format(str(self.timeplots_added))
                file = '{0}/{1}'.format(self.directory, file_name)

                # save figure to file
                plt.savefig(file)
                plt.close()

                # add the file name to the list of files
                file_names.append(file_name)

                # increment timeplot count
                self.timeplots_added += 1
        return file_names
