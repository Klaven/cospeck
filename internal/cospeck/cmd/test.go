package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func testCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test your container runtime",
		Run:   testRunner,
	}

	return cmd

}

func testRunner(cmd *cobra.Command, args []string) {
	fmt.Println("Running tests")
}
