package cmd

import (
	"github.com/Klaven/cospeck/internal/tests"
	"github.com/spf13/cobra"
)

// Flags represent cmd line flags
type Flags struct {
	Runtime       string
	CreateCluster bool
}

// RootCmd is the root command builder thing
func RootCmd() *cobra.Command {
	globalFlags := &Flags{}
	testFlags := &tests.TestFlags{}

	cmd := &cobra.Command{
		Use:   "cospeck",
		Short: "A container runtime speed test",
	}

	// subcommands
	cmd.AddCommand(testCmd(globalFlags, testFlags), nodeBusterCmd(globalFlags, testFlags))

	// Flags
	cmd.PersistentFlags().StringVarP(&globalFlags.Runtime, "runtime", "r", "/var/run/crio/crio.sock", "Runtime to use default: /var/run/crio/crio.sock")

	// really I would like to take the kubernetes cluster out of it eventually. but right now it makes some things easy
	cmd.PersistentFlags().BoolP("create-runtime", "c", true, "Create a cluster")

	return cmd
}
