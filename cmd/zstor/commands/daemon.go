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
	"os"

	"github.com/spf13/cobra"
	"github.com/zero-os/0-stor/client"
	"github.com/zero-os/0-stor/cmd"
	"github.com/zero-os/0-stor/daemon/api/grpc"
)

// daemonCfg represents the daemon subcommand configuration
var daemonCfg struct {
	ListenAddress cmd.ListenAddress
	MaxMsgSize    int
}

// daemonCmd represents the daemon subcommand
var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run the client API as a network-connected GRPC client.",
	RunE:  daemonFunc,
}

func daemonFunc(*cobra.Command, []string) error {
	cmd.LogVersion()

	confFile, err := os.Open(rootCfg.ConfigFile)
	if err != nil {
		return err
	}
	defer confFile.Close()

	policy, err := client.NewPolicyFromReader(confFile)
	if err != nil {
		return err
	}

	client, err := client.New(policy)
	if err != nil {
		return err
	}

	daem, err := grpc.New(client, daemonCfg.MaxMsgSize)
	if err != nil {
		return err
	}

	return daem.Listen(daemonCfg.ListenAddress.String())
}

func init() {
	daemonCmd.Flags().VarP(
		&daemonCfg.ListenAddress, "listen", "L", "Bind the proxy to the given host and port. Format has to be host:port, with host optional (default :8080)")
	daemonCmd.Flags().IntVar(
		&daemonCfg.MaxMsgSize, "max-msg-size", 32, "Configure the maximum size of the message this daemon can receive, in MiB")

}
