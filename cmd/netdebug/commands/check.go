package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/ryanelliottsmith/network-debugger/pkg/checks"
	"github.com/ryanelliottsmith/network-debugger/pkg/output"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Run standalone local checks",
	Long:  "Run individual network checks locally without DaemonSet coordination.",
}

var checkDNSCmd = &cobra.Command{
	Use:   "dns",
	Short: "Test DNS resolution",
	RunE: func(cmd *cobra.Command, args []string) error {
		names, _ := cmd.Flags().GetStringSlice("names")

		check := checks.NewDNSCheck(names, "")
		result := checks.RunWithTimeout(check, "dns-test", checks.DefaultCheckTimeout)

		format, _ := cmd.Flags().GetString("output")
		return output.PrintResult(result, format)
	},
}

var checkPingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Test ICMP connectivity",
	RunE: func(cmd *cobra.Command, args []string) error {
		targets, _ := cmd.Flags().GetStringSlice("targets")

		if len(targets) == 0 {
			return fmt.Errorf("at least one target required (use --targets)")
		}

		check := checks.NewPingCheck(0)
		format, _ := cmd.Flags().GetString("output")

		for _, target := range targets {
			result := checks.RunWithTimeout(check, target, checks.DefaultPingTimeout)
			if err := output.PrintResult(result, format); err != nil {
				return err
			}
			fmt.Println()
		}

		return nil
	},
}

var checkPortsCmd = &cobra.Command{
	Use:   "ports",
	Short: "Test port connectivity",
	RunE: func(cmd *cobra.Command, args []string) error {
		targets, _ := cmd.Flags().GetStringSlice("targets")

		if len(targets) == 0 {
			return fmt.Errorf("at least one target required (use --targets)")
		}

		// TODO: Parse custom ports from flags
		check := checks.NewPortsCheck(nil) // nil = use defaults
		format, _ := cmd.Flags().GetString("output")

		for _, target := range targets {
			result := checks.RunWithTimeout(check, target, checks.DefaultPortsTimeout)
			if err := output.PrintResult(result, format); err != nil {
				return err
			}
			fmt.Println()
		}

		return nil
	},
}

var checkBandwidthCmd = &cobra.Command{
	Use:   "bandwidth",
	Short: "Test network bandwidth",
	RunE: func(cmd *cobra.Command, args []string) error {
		target, _ := cmd.Flags().GetString("target")

		if target == "" {
			return fmt.Errorf("target required (use --target)")
		}

		debug, _ := cmd.Flags().GetBool("debug")
		check := checks.NewBandwidthCheck(debug)
		result := checks.RunWithTimeout(check, target, time.Duration(checks.BandwidthDuration+5)*time.Second)

		format, _ := cmd.Flags().GetString("output")
		return output.PrintResult(result, format)
	},
}

var checkHostConfigCmd = &cobra.Command{
	Use:   "hostconfig",
	Short: "Check host configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		check := checks.NewHostConfigCheck()
		result := checks.RunWithTimeout(check, "localhost", checks.DefaultCheckTimeout)

		format, _ := cmd.Flags().GetString("output")
		return output.PrintResult(result, format)
	},
}

var checkConntrackCmd = &cobra.Command{
	Use:   "conntrack",
	Short: "Check conntrack statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		check := checks.NewConntrackCheck()
		result := checks.RunWithTimeout(check, "localhost", checks.DefaultCheckTimeout)

		format, _ := cmd.Flags().GetString("output")
		return output.PrintResult(result, format)
	},
}

var checkIptablesCmd = &cobra.Command{
	Use:   "iptables",
	Short: "Check iptables configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		check := checks.NewIptablesCheck()
		result := checks.RunWithTimeout(check, "localhost", checks.DefaultCheckTimeout)

		format, _ := cmd.Flags().GetString("output")
		return output.PrintResult(result, format)
	},
}

func init() {
	checkCmd.AddCommand(checkDNSCmd)
	checkCmd.AddCommand(checkPingCmd)
	checkCmd.AddCommand(checkPortsCmd)
	checkCmd.AddCommand(checkBandwidthCmd)
	checkCmd.AddCommand(checkHostConfigCmd)
	checkCmd.AddCommand(checkConntrackCmd)
	checkCmd.AddCommand(checkIptablesCmd)

	checkDNSCmd.Flags().StringSlice("servers", []string{}, "DNS servers to test")
	checkDNSCmd.Flags().StringSlice("names", []string{}, fmt.Sprintf("Names to resolve (default: %s)", strings.Join(checks.DefaultDNSNames, ", ")))

	checkPingCmd.Flags().StringSlice("targets", []string{}, "Target hosts to ping")

	checkPortsCmd.Flags().StringSlice("targets", []string{}, "Target hosts")
	checkPortsCmd.Flags().StringSlice("ports", []string{}, "Ports to check (format: 8080/tcp:name)")

	checkBandwidthCmd.Flags().String("target", "", "Target host for bandwidth test")
}
