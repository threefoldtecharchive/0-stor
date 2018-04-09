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
	"sync"

	"github.com/zero-os/0-stor/benchmark/bencher"
	"github.com/zero-os/0-stor/benchmark/config"
	"github.com/zero-os/0-stor/cmd"

	"github.com/pkg/profile"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

//rootCfg defines flags
var rootCfg struct {
	confFile         string
	benchmarkOutPath string
	profileOutPath   string
	profileMode      cmd.ProfileMode
	debugLog         bool
}

var (
	// RootCmd creates flags
	RootCmd = &cobra.Command{
		Use:   "zstorbench",
		Short: "A tool for benchmarking (and profiling) a zstor client",
		PersistentPreRun: func(*cobra.Command, []string) {
			if rootCfg.debugLog {
				log.SetLevel(log.DebugLevel)
				log.Debug("Debug logging enabled")
			}
		},
		RunE: rootFunc,
	}
)

func init() {
	RootCmd.PersistentFlags().BoolVarP(&rootCfg.debugLog, "debug", "D", false, "Enable debug logging.")
	RootCmd.Flags().StringVarP(&rootCfg.confFile, "conf", "C", "zstorbench_config.yaml", "path to a config file")
	RootCmd.Flags().StringVar(&rootCfg.benchmarkOutPath, "out-benchmark", "benchmark.yaml", "path and filename where benchmarking results are written")
	RootCmd.Flags().StringVar(&rootCfg.profileOutPath, "out-profile", "./profile", "path where profiling files are written")
	RootCmd.Flags().VarP(&rootCfg.profileMode, "profile-mode", "", rootCfg.profileMode.Description())
}

func rootFunc(*cobra.Command, []string) error {
	// get configuration
	log.Infof("reading config at %q...", rootCfg.confFile)
	clientConf, err := readConfig(rootCfg.confFile)
	if err != nil {
		return err
	}

	// Start profiling if profiling flag is given
	if rootCfg.profileMode != cmd.ProfileModeDisabled {
		switch rootCfg.profileMode {
		case cmd.ProfileModeCPU:
			defer profile.Start(
				profile.ProfilePath(rootCfg.profileOutPath),
				profile.CPUProfile).Stop()
		case cmd.ProfileModeMem:
			defer profile.Start(
				profile.ProfilePath(rootCfg.profileOutPath),
				profile.MemProfile).Stop()
		case cmd.ProfileModeTrace:
			defer profile.Start(
				profile.ProfilePath(rootCfg.profileOutPath),
				profile.TraceProfile).Stop()
		case cmd.ProfileModeBlock:
			defer profile.Start(
				profile.ProfilePath(rootCfg.profileOutPath),
				profile.BlockProfile).Stop()
		}
	}

	output := NewOutputFormat()

	var (
		b       *bencher.Bencher
		clients []*bencher.Bencher
		cc      int // client count
		wg      sync.WaitGroup
		results []*bencher.Result
	)

	//Run benchmarking for provided scenarios
	for scID, sc := range clientConf.Scenarios {
		log.Infof("Setting up benchmark `%s`...", scID)

		// get concurrent clients
		cc = sc.BenchConf.Clients
		if cc < 1 {
			cc = 1
		}
		clients = make([]*bencher.Bencher, cc)
		results = make([]*bencher.Result, cc)

		// init clients
		for i := range clients {
			b, err = bencher.NewBencher(scID, &sc)
			if err != nil {
				goto WriteResult
			}
			clients[i] = b
		}

		// run benchmarks concurrently
		log.Infof("Running benchmark `%s`...", scID)
		for i := range clients {
			wg.Add(1)
			go func(m *bencher.Bencher, i int) {
				var result *bencher.Result
				result, err = b.RunBenchmark()
				results[i] = result
				wg.Done()
			}(clients[i], i)
		}
		wg.Wait()

		// collect results of the benchmarking cycle
	WriteResult:
		if err != nil {
			log.Error(err)
		}
		output.Scenarios[scID] = FormatOutput(results, sc, err)
	}

	// write results to file
	log.Info("Benchmarking done! Writing results!")
	return writeOutput(rootCfg.benchmarkOutPath, output)
}

// readConfig reads a YAML config file and returns a config.ClientConf
// based on that file
func readConfig(path string) (*config.ClientConf, error) {
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// parse the config file to clientConf structure
	return config.UnmarshalYAML(yamlFile)
}
