package cmd

import (
	"github.com/Klaven/cospeck/internal/tests"
	"github.com/spf13/cobra"
)

func nodeBusterCmd(flags *Flags, testFlags *tests.TestFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nodebuster",
		Short: "Test your nodes container runtime.... to it's limits",
		Long:  "Test your container runtime, to it's limits! \n WARNING, do not run this on a node that is running production anything!!!!",
		Run: func(cmd *cobra.Command, args []string) {
			nodeBusterRunner(flags, testFlags)
		},
	}

	return cmd
}

// nodeBusterRunner will try and break your node
func nodeBusterRunner(flags *Flags, testFlags *tests.TestFlags) {
	tests.NodeBusterTest(testFlags)
}
