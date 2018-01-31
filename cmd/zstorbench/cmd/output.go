/*
 * Copyright (C) 2017-2018 GIG Technology NV and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"io/ioutil"

	"github.com/zero-os/0-stor/benchmark/bencher"
	"github.com/zero-os/0-stor/benchmark/config"

	yaml "gopkg.in/yaml.v2"
)

// NewOutputFormat returns a new OutputFormat
func NewOutputFormat() OutputFormat {
	var o OutputFormat
	o.Scenarios = make(map[string]ScenarioOutputFormat)
	return o
}

// OutputFormat represents the output format of a full benchmark
type OutputFormat struct {
	Scenarios map[string]ScenarioOutputFormat
}

//ScenarioOutputFormat represents a scenario result for outputting
type ScenarioOutputFormat struct {
	Results      []*bencher.Result `yaml:"results,omitempty"`
	ScenarioConf config.Scenario   `yaml:"scenario,omitempty"`
	Error        string            `yaml:"error,omitempty"`
}

//FormatOutput formats the output of the benchmarking scenario
func FormatOutput(results []*bencher.Result, scenarioConfig config.Scenario, err error) ScenarioOutputFormat {
	var output ScenarioOutputFormat
	output.ScenarioConf = scenarioConfig
	if err != nil {
		output.Error = err.Error()
		return output
	}
	output.Results = results
	return output
}

// writeOutput writes OutputFormat to provided file
func writeOutput(filePath string, output OutputFormat) error {
	yamlBytes, err := yaml.Marshal(output)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filePath, yamlBytes, 0644)
}
