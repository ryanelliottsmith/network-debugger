package main

import (
	"os"

	"github.com/ryanelliottsmith/network-debugger/cmd/netdebug/commands"
)

// Version information (set via ldflags at build time)
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	// Set version info in commands package
	commands.SetVersionInfo(version, commit, buildDate)

	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
