package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func RootCmd() *cobra.Command {
	return &cobra.Command {
		Use: "cospeck",
		Short: "A container runtime speed test",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("starting tests")
		},
	}
}
