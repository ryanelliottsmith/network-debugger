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

var rootCmd = &cobra.Command{
	Use:   "netdebug",
	Short: "Network debugger for Kubernetes clusters",
	Long: `A comprehensive network debugging tool for Kubernetes clusters (RKE2/K3s).
Helps diagnose connectivity issues, DNS problems, port accessibility, and more.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(deployCmd)

	rootCmd.PersistentFlags().StringP("output", "o", "table", "Output format (table, json, yaml)")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug output")
}

func exitWithError(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+msg+"\n", args...)
	os.Exit(1)
}
