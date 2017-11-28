package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zero-os/0-stor/client"
	"github.com/zero-os/0-stor/client/itsyouonline"
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
	Short: "0-stor cli used to interact with a 0-stor server.",
}

var rootCfg struct {
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

	rootCmd.PersistentFlags().StringVarP(
		&rootCfg.ConfigFile, "config", "c", "config.yaml",
		"Path to the configuration file.")
}
