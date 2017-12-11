package commands

import (
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/zero-os/0-stor/client"
	"github.com/zero-os/0-stor/client/itsyouonline"
	"github.com/zero-os/0-stor/client/metastor"
	"github.com/zero-os/0-stor/client/metastor/etcd"
	"github.com/zero-os/0-stor/cmd"
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
}

func getClient() (*client.Client, error) {
	// create policy
	policy, err := readPolicy()
	if err != nil {
		return nil, err
	}

	// create client
	cl, err := client.New(policy)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	return cl, nil
}

func getMetaClient() (metastor.Client, error) {
	// create policy
	policy, err := readPolicy()
	if err != nil {
		return nil, err
	}

	return etcd.NewClient(policy.MetaShards)
}

func getNamespaceManager() (itsyouonline.IYOClient, error) {
	policy, err := readPolicy()
	if err != nil {
		return nil, err
	}

	return itsyouonline.NewClient(policy.Organization, policy.IYOAppID, policy.IYOSecret), nil
}

func readPolicy() (client.Policy, error) {
	// read config file
	f, err := os.Open(rootCfg.ConfigFile)
	if err != nil {
		return client.Policy{}, err
	}

	// parse config file and return it as a policy object if possible
	return client.NewPolicyFromReader(f)
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
}
