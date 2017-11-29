package commands

import (
	"errors"

	"github.com/spf13/cobra"
)

// daemonCmd represents the daemon subcommand
var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run the client API as a network-connected GRPC client.",
	RunE: func(*cobra.Command, []string) error {
		return errors.New("the daemon is not yet supported")
	},
}
