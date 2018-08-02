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
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/threefoldtech/0-stor/cmd"
	"github.com/threefoldtech/0-stor/daemon"
	daemon_grpc "github.com/threefoldtech/0-stor/daemon/api/grpc"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// daemonCfg represents the daemon subcommand configuration
var daemonCfg struct {
	ListenAddress        cmd.ListenAddress
	MaxMsgSize           int
	DisableLocalFSAccess bool
}

// daemonCmd represents the daemon subcommand
var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run the client API as a network-connected GRPC client.",
	RunE:  daemonFunc,
}

func daemonFunc(*cobra.Command, []string) error {
	cmd.LogVersion()

	// create a UNIX/TCP listener for our daemon (server)
	listener, err := net.Listen(
		daemonCfg.ListenAddress.NetworkProtocol(),
		daemonCfg.ListenAddress.String())
	if err != nil {
		return err
	}

	// read the client config and create our daemon
	cfg, err := daemon.ReadConfig(rootCfg.ConfigFile)
	if err != nil {
		return err
	}
	daemon, err := daemon_grpc.NewFromConfig(
		*cfg, daemonCfg.MaxMsgSize, rootCfg.JobCount,
		daemonCfg.DisableLocalFSAccess)
	if err != nil {
		return err
	}

	// serve our daemon on a separate channel
	errChan := make(chan error, 1)
	go func() {
		err := daemon.Serve(listener)
		errChan <- err
	}()

	log.Infof("Daemon Server interface: grpc")
	log.Infof("Daemon Server listening on %s (net protocol: %s)",
		listener.Addr().String(),
		daemonCfg.ListenAddress.NetworkProtocol())

	// wait until the daemon has to be gracefully closed,
	// or until a fatal error occurs
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)
	select {
	case err := <-errChan:
		return err
	case <-sigChan:
		return daemon.Close()
	}
}

func init() {
	daemonCmd.Flags().VarP(
		&daemonCfg.ListenAddress, "listen", "L", daemonCfg.ListenAddress.Description())
	daemonCmd.Flags().IntVar(
		&daemonCfg.MaxMsgSize, "max-msg-size", daemon_grpc.DefaultMaxMsgSize,
		"Configure the maximum size of the message this daemon can receive and send, in MiB")
	daemonCmd.Flags().BoolVar(
		&daemonCfg.DisableLocalFSAccess, "no-local-fs", false,
		"disable any local file system access for r/w purposes")
}
