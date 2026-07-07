package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version   string
	commit    string
	buildDate string
)

func SetVersionInfo(v, c, b string) {
	version = v
	commit = c
	buildDate = b
}

var RootCmd = &cobra.Command{
	Use:   "netdebug",
	Short: "Network debugger for Kubernetes clusters",
	Long: `A comprehensive network debugging tool for Kubernetes clusters (RKE2/K3s).
Helps diagnose connectivity issues, DNS problems, port accessibility, and more.`,
}

func Execute() error {
	return RootCmd.Execute()
}

func init() {
	RootCmd.AddCommand(versionCmd)
	RootCmd.AddCommand(runCmd)
	RootCmd.AddCommand(checkCmd)
	RootCmd.AddCommand(agentCmd)
	RootCmd.AddCommand(deployCmd)

	RootCmd.PersistentFlags().StringP("output", "o", "table", "Output format (table, json, yaml)")
	RootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress detailed output")
}

func exitWithError(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+msg+"\n", args...)
	os.Exit(1)
}
