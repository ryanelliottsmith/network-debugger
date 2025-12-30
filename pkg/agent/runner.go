package agent

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/ryanelliottsmith/network-debugger/pkg/checks"
	"github.com/ryanelliottsmith/network-debugger/pkg/types"
)

func RunTests(ctx context.Context, config *types.Config, self *SelfInfo) error {
	if err := EmitReady(self, config.RunID); err != nil {
		log.Printf("Failed to emit ready event: %v", err)
	}

	targets := filterTargets(config.Targets, self.NodeName)

	var wg sync.WaitGroup
	for _, checkName := range config.Checks {
		if checkName == "bandwidth" {
			continue
		}

		wg.Add(1)
		go func(checkName string) {
			defer wg.Done()
			runCheckAgainstAllTargets(ctx, checkName, targets, config, self)
		}(checkName)
	}
	wg.Wait()

	if config.BandwidthTest != nil && config.BandwidthTest.Active {
		if config.BandwidthTest.SourcePod == self.PodName {
			runBandwidthTest(ctx, config.BandwidthTest, self, config.RunID, config.Debug)
		}
	}

	summary := map[string]interface{}{
		"checks_completed": len(config.Checks),
		"targets_tested":   len(targets),
	}

	if err := EmitComplete(self, config.RunID, summary); err != nil {
		log.Printf("Failed to emit complete event: %v", err)
	}

	return nil
}

func filterTargets(targets []types.TargetNode, selfNodeName string) []types.TargetNode {
	var filtered []types.TargetNode
	for _, target := range targets {
		if target.NodeName != selfNodeName {
			filtered = append(filtered, target)
		}
	}
	return filtered
}

func runCheckAgainstAllTargets(ctx context.Context, checkName string, targets []types.TargetNode, config *types.Config, self *SelfInfo) {
	for _, target := range targets {
		runSingleCheck(ctx, checkName, target.IP, target.NodeName, config, self)
	}
}

func runSingleCheck(ctx context.Context, checkName, targetIP, targetNode string, config *types.Config, self *SelfInfo) {
	if err := EmitTestStart(self, checkName, targetNode, config.RunID); err != nil {
		log.Printf("Failed to emit test start: %v", err)
	}

	var check checks.Check

	switch checkName {
	case "dns":
		names := config.DNSNames
		if len(names) == 0 {
			names = []string{"kubernetes.default.svc.cluster.local", "google.com"}
		}
		check = checks.NewDNSCheck(names, "")
		targetIP = "dns-test"

	case "ping":
		check = checks.NewPingCheck(5)

	case "ports":
		ports := config.Ports
		if len(ports) == 0 {
			ports = types.DefaultPorts()
		}
		check = checks.NewPortsCheck(ports)

	case "hostconfig":
		check = checks.NewHostConfigCheck()
		targetIP = "localhost"

	case "conntrack":
		check = checks.NewConntrackCheck()
		targetIP = "localhost"

	case "iptables":
		check = checks.NewIptablesCheck()
		targetIP = "localhost"

	default:
		log.Printf("Unknown check type: %s", checkName)
		return
	}

	result := checks.RunWithTimeout(check, targetIP, 5*time.Second)
	result.Node = self.NodeName

	if err := EmitTestResult(self, result, config.RunID); err != nil {
		log.Printf("Failed to emit test result: %v", err)
	}
}

func runBandwidthTest(ctx context.Context, test *types.BandwidthTest, self *SelfInfo, runID string, debug bool) {
	log.Printf("Running bandwidth test to %s (%s)", test.TargetNode, test.TargetIP)

	if err := EmitTestStart(self, "bandwidth", test.TargetNode, runID); err != nil {
		log.Printf("Failed to emit test start: %v", err)
	}

	check := checks.NewBandwidthCheck(debug)
	result := checks.RunWithTimeout(check, test.TargetIP, time.Duration(checks.BandwidthDuration+5)*time.Second)
	result.Node = self.NodeName
	result.Target = test.TargetNode

	if err := EmitTestResult(self, result, runID); err != nil {
		log.Printf("Failed to emit test result: %v", err)
	}
}
