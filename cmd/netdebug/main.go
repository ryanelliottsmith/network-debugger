package main

import (
	"os"

	"github.com/ryanelliottsmith/network-debugger/cmd/netdebug/commands"
)

func main() {
	// Set version info in commands package
	commands.SetVersionInfo(Version, Commit, BuildDate)

	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
