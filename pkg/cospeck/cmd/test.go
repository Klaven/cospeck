package cmd

import (
	"fmt"

	"github.com/Klaven/cospeck/internal/tests"
	"github.com/spf13/cobra"
)

func testCmd(flags *Flags, testFlags *tests.TestFlags) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test your container runtime",
	}

	cmd.AddCommand(GeneralTest(testFlags))

	return cmd

}

// GeneralTest test memory consumption
func GeneralTest(testFlags *tests.TestFlags) *cobra.Command {

	var pods int
	cmd := &cobra.Command{
		Use:   "general",
		Short: "general container runtime memory and cpu usage test",
		Run: func(cmd *cobra.Command, args []string) {
			// Do Stuff Here
			tests.GeneralTest(testFlags, pods)
		},
	}

	cmd.Flags().IntVarP(&pods, "Pods", "p", 100, "Number of pods to use when testing memory")

	return cmd
}

func testRunner(flags *Flags, testFlags *tests.TestFlags) {
	fmt.Println("Running tests")

	//TODO: check to make sure namesapce is cleaned up first (and maybe should create the namespace, failing if it exists)
	//TODO: fail if not clean

}
