package cospeck

import "github.com/Klaven/cospeck/pkg/cospeck/cmd"

// Run our cmd
func Run() {
	cmd.RootCmd().Execute()
}
