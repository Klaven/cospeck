package cospeck

import "github.com/Klaven/cospeck/internal/cospeck/cmd"

// Run our cmd
func Run() {
	cmd.RootCmd().Execute()
}
