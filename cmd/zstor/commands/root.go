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

package commands

import (
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/zero-os/0-stor/client"
	"github.com/zero-os/0-stor/client/itsyouonline"
	"github.com/zero-os/0-stor/client/metastor"
	"github.com/zero-os/0-stor/client/metastor/db/etcd"
	"github.com/zero-os/0-stor/cmd"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "zstor",
	Short: "Client used to manage 0-stor (meta)data and permissions.",
	PersistentPreRun: func(*cobra.Command, []string) {
		if rootCfg.DebugLog {
			log.SetLevel(log.DebugLevel)
			log.Debug("Debug logging enabled")
		}
	},
}

var rootCfg struct {
	DebugLog   bool
	ConfigFile string
	JobCount   int
}

func getClient() (*client.Client, error) {
	cfg, err := client.ReadConfig(rootCfg.ConfigFile)
	if err != nil {
		return nil, err
	}
	// create client
	cl, err := client.NewClientFromConfigWithoutCaching(*cfg, rootCfg.JobCount)
	if err != nil {
		return nil, fmt.Errorf("failed to create 0-stor client: %v", err)
	}

	return cl, nil
}

func getMetaClient() (*metastor.Client, error) {
	cfg, err := client.ReadConfig(rootCfg.ConfigFile)
	if err != nil {
		return nil, err
	}
	if len(cfg.MetaStor.Shards) == 0 {
		return nil, errors.New("failed to create metastor client: no metastor shards defined")
	}

	// create ETCD database (client)
	db, err := etcd.New(cfg.MetaStor.Shards)
	if err != nil {
		return nil, err
	}
	// create metastor client
	return metastor.NewClient(metastor.Config{Database: db})
}

func getNamespaceManager() (*itsyouonline.Client, error) {
	cfg, err := client.ReadConfig(rootCfg.ConfigFile)
	if err != nil {
		return nil, err
	}
	return itsyouonline.NewClient(cfg.IYO)
}

func init() {
	rootCmd.AddCommand(
		fileCmd,
		namespaceCmd,
		daemonCmd,
		cmd.VersionCmd,
	)

	rootCmd.PersistentFlags().BoolVarP(
		&rootCfg.DebugLog, "debug", "D", false, "Enable debug logging.")
	rootCmd.PersistentFlags().StringVarP(
		&rootCfg.ConfigFile, "config", "C", "config.yaml",
		"Path to the configuration file.")
	rootCmd.PersistentFlags().IntVarP(
		&rootCfg.JobCount, "jobs", "J", runtime.NumCPU()*2,
		"number of parallel jobs to run for tasks that support this")
}
