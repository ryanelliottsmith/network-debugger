package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	checkspkg "github.com/ryanelliottsmith/network-debugger/pkg/checks"
	"github.com/ryanelliottsmith/network-debugger/pkg/coordinator"
	"github.com/ryanelliottsmith/network-debugger/pkg/k8s"
	"github.com/ryanelliottsmith/network-debugger/pkg/output"
	"github.com/ryanelliottsmith/network-debugger/pkg/types"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run coordinated network tests via DaemonSet",
	Long: `Deploy a DaemonSet and run coordinated network tests across all nodes.
Tests connectivity, DNS, ports, and optionally bandwidth between nodes.`,
	RunE: runTests,
}

func init() {
	runCmd.Flags().StringSlice("checks", []string{"dns", "ping", "hostconfig", "conntrack", "iptables"}, "Checks to run (dns,ping,ports,bandwidth,hostconfig,conntrack,iptables)")
	runCmd.Flags().Bool("host-network", true, "Test host network path")
	runCmd.Flags().Bool("overlay", true, "Test overlay network path")
	runCmd.Flags().Bool("no-host-network", false, "Disable host network testing")
	runCmd.Flags().Bool("no-overlay", false, "Disable overlay network testing")
	runCmd.Flags().StringSlice("ports", []string{}, "Override default port list (format: 8080/tcp:name,9000/udp:name)")
	runCmd.Flags().StringP("namespace", "n", "netdebug", "Namespace for DaemonSet deployment")
	runCmd.Flags().Duration("timeout", 5*time.Minute, "Overall timeout (0 = no timeout)")
	runCmd.Flags().Bool("cleanup", false, "Remove DaemonSet after test completion")
	runCmd.Flags().Bool("tui", false, "Enable TUI dashboard (not yet implemented)")
	runCmd.Flags().StringP("output", "o", "table", "Output format (table, json, yaml)")
}

func runTests(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	checks, _ := cmd.Flags().GetStringSlice("checks")
	hostNetwork, _ := cmd.Flags().GetBool("host-network")
	overlay, _ := cmd.Flags().GetBool("overlay")
	noHostNetwork, _ := cmd.Flags().GetBool("no-host-network")
	noOverlay, _ := cmd.Flags().GetBool("no-overlay")
	namespace, _ := cmd.Flags().GetString("namespace")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	cleanup, _ := cmd.Flags().GetBool("cleanup")
	outputFormat, _ := cmd.Flags().GetString("output")
	debug, _ := cmd.Flags().GetBool("debug")

	if noHostNetwork {
		hostNetwork = false
	}
	if noOverlay {
		overlay = false
	}

	if !hostNetwork && !overlay {
		return fmt.Errorf("at least one network mode must be enabled (--host-network or --overlay)")
	}

	bandwidthRequested := false
	checksWithoutBandwidth := []string{}
	for _, check := range checks {
		if check == "bandwidth" {
			bandwidthRequested = true
		} else {
			checksWithoutBandwidth = append(checksWithoutBandwidth, check)
		}
	}

	fmt.Println("🚀 Starting network tests...")
	fmt.Printf("Network modes: host=%v overlay=%v\n", hostNetwork, overlay)
	fmt.Printf("Checks: %s\n", strings.Join(checks, ", "))
	fmt.Println()

	clientset, err := k8s.GetClientset()
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %w", err)
	}

	dynamicClient, err := k8s.GetDynamicClient()
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	fmt.Println("📦 Checking DaemonSet deployment...")
	_, err = clientset.AppsV1().DaemonSets(namespace).Get(ctx, "netdebug-host", metav1.GetOptions{})
	needsDeployment := err != nil

	if needsDeployment {
		fmt.Println("Deploying DaemonSets...")
		if err := k8s.Install(ctx, clientset, dynamicClient, namespace, ""); err != nil {
			return fmt.Errorf("failed to deploy: %w", err)
		}
		fmt.Println("✅ DaemonSets deployed")
	} else {
		fmt.Println("✅ DaemonSets already deployed")
	}

	fmt.Println("\n⏳ Waiting for DaemonSets to be ready...")
	if hostNetwork {
		if err := k8s.WaitForDaemonSetReady(ctx, clientset, namespace, "netdebug-host", 2*time.Minute); err != nil {
			return fmt.Errorf("host network DaemonSet not ready: %w", err)
		}
		fmt.Println("✅ Host network DaemonSet ready")
	}
	if overlay {
		if err := k8s.WaitForDaemonSetReady(ctx, clientset, namespace, "netdebug-overlay", 2*time.Minute); err != nil {
			return fmt.Errorf("overlay network DaemonSet not ready: %w", err)
		}
		fmt.Println("✅ Overlay network DaemonSet ready")
	}

	coord := coordinator.NewCoordinator(clientset, namespace, "netdebug-config")

	fmt.Println("\n🔍 Discovering pods...")
	var hostPods, overlayPods []types.TargetNode
	var hostTargets, overlayTargets []types.TargetNode

	if hostNetwork {
		hostPods, err = k8s.DiscoverDaemonSetPods(ctx, clientset, namespace, "netdebug-host")
		if err != nil {
			return fmt.Errorf("failed to discover host pods: %w", err)
		}
		hostTargets, err = k8s.GetHostIPsForPods(ctx, clientset, namespace, hostPods)
		if err != nil {
			return fmt.Errorf("failed to get host IPs: %w", err)
		}
		fmt.Printf("Found %d host network pods\n", len(hostPods))
	}

	if overlay {
		overlayPods, err = k8s.DiscoverDaemonSetPods(ctx, clientset, namespace, "netdebug-overlay")
		if err != nil {
			return fmt.Errorf("failed to discover overlay pods: %w", err)
		}
		overlayTargets = overlayPods
		fmt.Printf("Found %d overlay network pods\n", len(overlayPods))
	}

	allEvents := []*types.Event{}

	if len(checksWithoutBandwidth) > 0 {
		fmt.Println("\n🧪 Running standard checks...")

		if hostNetwork {
			fmt.Println("\n--- Host Network Tests ---")
			events, err := runStandardTests(ctx, coord, hostTargets, hostPods, checksWithoutBandwidth, timeout, debug, true)
			if err != nil {
				fmt.Printf("Warning: host network tests failed: %v\n", err)
			}
			allEvents = append(allEvents, events...)
		}

		if overlay {
			fmt.Println("\n--- Overlay Network Tests ---")
			overlayChecks := filterOutCheck(checksWithoutBandwidth, "ports")
			events, err := runStandardTests(ctx, coord, overlayTargets, overlayPods, overlayChecks, timeout, debug, false)
			if err != nil {
				fmt.Printf("Warning: overlay network tests failed: %v\n", err)
			}
			allEvents = append(allEvents, events...)
		}
	}

	if bandwidthRequested {
		fmt.Println("\n📊 Running bandwidth tests...")

		if hostNetwork {
			fmt.Println("\n--- Host Network Bandwidth ---")
			events, err := runBandwidthTests(ctx, coord, hostTargets, hostPods, timeout, debug)
			if err != nil {
				fmt.Printf("Warning: host bandwidth tests failed: %v\n", err)
			}
			allEvents = append(allEvents, events...)
		}

		if overlay {
			fmt.Println("\n--- Overlay Network Bandwidth ---")
			events, err := runBandwidthTests(ctx, coord, overlayTargets, overlayPods, timeout, debug)
			if err != nil {
				fmt.Printf("Warning: overlay bandwidth tests failed: %v\n", err)
			}
			allEvents = append(allEvents, events...)
		}
	}

	if cleanup {
		fmt.Println("\n🧹 Cleaning up...")
		if err := k8s.Uninstall(ctx, dynamicClient, namespace); err != nil {
			fmt.Printf("Warning: cleanup failed: %v\n", err)
		} else {
			fmt.Println("✅ Resources cleaned up")
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📊 Test Results")
	fmt.Println(strings.Repeat("=", 80) + "\n")

	if err := output.FormatEvents(allEvents, outputFormat, debug); err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}

func runStandardTests(ctx context.Context, coord *coordinator.Coordinator, targets []types.TargetNode, pods []types.TargetNode, checks []string, timeout time.Duration, debug bool, isHostNetwork bool) ([]*types.Event, error) {
	testCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	runID := coordinator.GenerateRunID()

	dnsNames := checkspkg.DefaultDNSNames
	if isHostNetwork {
		dnsNames = filterClusterLocalNames(dnsNames)
	}

	config := &types.Config{
		RunID:       runID,
		TriggeredAt: time.Now(),
		Targets:     targets,
		Checks:      checks,
		Ports:       types.DefaultPorts(),
		DNSNames:    dnsNames,
		Timeout:     5,
		Debug:       debug,
	}

	podNames := make([]string, len(pods))
	for i, pod := range pods {
		podNames[i] = pod.PodName
	}

	fmt.Printf("Starting test run %s with %d pods...\n", runID[:8], len(podNames))

	events, err := coord.RunTests(testCtx, config, podNames, timeout)
	if err != nil {
		return nil, err
	}

	fmt.Printf("✅ Test run completed (%d events collected)\n", len(events))
	return events, nil
}

func runBandwidthTests(ctx context.Context, coord *coordinator.Coordinator, targets []types.TargetNode, pods []types.TargetNode, timeout time.Duration, debug bool) ([]*types.Event, error) {
	pairs := coordinator.GenerateBandwidthPairs(targets)

	fmt.Printf("Running %d bandwidth tests (sequential)...\n", len(pairs))

	allEvents := []*types.Event{}

	for idx, pair := range pairs {
		testCtx, cancel := context.WithCancel(ctx)

		source := pair[0]
		target := pair[1]

		fmt.Printf("[%d/%d] Testing %s -> %s... ", idx+1, len(pairs), source.NodeName, target.NodeName)

		runID := coordinator.GenerateRunID()

		config := &types.Config{
			RunID:       runID,
			TriggeredAt: time.Now(),
			Targets:     []types.TargetNode{target},
			Checks:      []string{},
			BandwidthTest: &types.BandwidthTest{
				Active:     true,
				SourceNode: source.NodeName,
				SourcePod:  source.PodName,
				TargetNode: target.NodeName,
				TargetIP:   target.IP,
			},
			Timeout: 5,
			Debug:   debug,
		}

		podNames := []string{source.PodName}

		events, err := coord.RunTests(testCtx, config, podNames, timeout)
		cancel()

		if err != nil {
			fmt.Printf("❌ Failed: %v\n", err)
			continue
		}

		allEvents = append(allEvents, events...)
		fmt.Println("✅")

		time.Sleep(2 * time.Second)
	}

	fmt.Printf("✅ All bandwidth tests completed\n")
	return allEvents, nil
}

func filterClusterLocalNames(names []string) []string {
	var filtered []string
	for _, name := range names {
		if !strings.HasSuffix(name, ".cluster.local") {
			filtered = append(filtered, name)
		}
	}
	return filtered
}

func filterOutCheck(checks []string, checkToRemove string) []string {
	var filtered []string
	for _, check := range checks {
		if check != checkToRemove {
			filtered = append(filtered, check)
		}
	}
	return filtered
}
