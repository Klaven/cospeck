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
		Run: func(cmd *cobra.Command, args []string) {
			// Do Stuff Here
			testRunner(flags, testFlags)
		},
	}

	return cmd

}

func testRunner(flags *Flags, testFlags *tests.TestFlags) {
	fmt.Println("Running tests")

	//TODO: check to make sure namesapce is cleaned up first (and maybe should create the namespace, failing if it exists)
	//TODO: fail if not clean

}
