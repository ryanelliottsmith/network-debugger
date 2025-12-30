package commands

import (
	"context"
	"fmt"

	"github.com/ryanelliottsmith/network-debugger/pkg/agent"
	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Run as agent (for DaemonSet or standalone container)",
	Long: `Run in agent mode. Can operate in two modes:
  1. ConfigMap mode: Watch ConfigMap for test triggers (for DaemonSet)
  2. Direct mode: Run checks specified via flags (for standalone use)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mode, _ := cmd.Flags().GetString("mode")
		configRef, _ := cmd.Flags().GetString("config")

		if mode == "configmap" {
			if configRef == "" {
				return fmt.Errorf("--config required for configmap mode (format: NAMESPACE/CONFIGMAPNAME)")
			}
		}

		ctx := context.Background()
		return agent.Run(ctx, mode, configRef)
	},
}

func init() {
	agentCmd.Flags().String("mode", "", "Agent mode: 'configmap' or empty for direct mode")
	agentCmd.Flags().String("config", "", "ConfigMap reference in format NAMESPACE/CONFIGMAPNAME (for configmap mode)")
	agentCmd.Flags().StringSlice("checks", []string{}, "Checks to run (direct mode)")
	agentCmd.Flags().Bool("host-network", false, "Test host network (direct mode)")
	agentCmd.Flags().Bool("overlay", false, "Test overlay network (direct mode)")
	agentCmd.Flags().StringSlice("ports", []string{}, "Override default ports (direct mode)")
}
