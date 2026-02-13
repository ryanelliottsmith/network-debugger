package checks

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/ryanelliottsmith/network-debugger/pkg/types"
	"github.com/ryanelliottsmith/network-debugger/pkg/util"
)

type HostConfigCheck struct{}

func (c *HostConfigCheck) Name() string {
	return "hostconfig"
}

func (c *HostConfigCheck) Run(ctx context.Context, target string) (*types.TestResult, error) {
	result := &types.TestResult{
		Check:  c.Name(),
		Target: "localhost",
		Status: types.StatusPass,
	}

	details := types.HostConfigDetails{
		KernelParams: make(map[string]string),
	}

	var issues []string

	ipForward, err := util.ReadSysctl("/proc/sys/net/ipv4/ip_forward")
	if err != nil {
		issues = append(issues, fmt.Sprintf("failed to read ip_forward: %v", err))
	} else if ipForward != "1" {
		details.IPForwarding = false
		issues = append(issues, "IP forwarding is disabled (should be enabled for Kubernetes)")
	} else {
		details.IPForwarding = true
	}

	mtu, err := c.getMTU(ctx)
	if err != nil {
		issues = append(issues, fmt.Sprintf("failed to get MTU: %v", err))
	} else {
		details.MTU = mtu
	}

	numCPU := runtime.NumCPU()
	details.NumCPU = numCPU
	loadAvg, err := c.getLoadAverage()
	if err != nil {
		issues = append(issues, fmt.Sprintf("failed to read load average: %v", err))
	} else {
		details.LoadAverage = loadAvg
		threshold := float64(numCPU) * 0.8
		if loadAvg > threshold {
			issues = append(issues, fmt.Sprintf("load average %.2f exceeds 80%% of available CPUs (%d) — threshold: %.2f", loadAvg, numCPU, threshold))
		}
	}

	kernelParams := map[string]string{
		"net.ipv4.conf.all.rp_filter":        "/proc/sys/net/ipv4/conf/all/rp_filter",
		"net.bridge.bridge-nf-call-iptables": "/proc/sys/net/bridge/bridge-nf-call-iptables",
		"net.ipv4.ip_local_port_range":       "/proc/sys/net/ipv4/ip_local_port_range",
	}

	for name, path := range kernelParams {
		value, err := util.ReadSysctl(path)
		if err == nil {
			details.KernelParams[name] = value
		}
	}

	if len(issues) > 0 {
		result.Status = types.StatusFail
		details.Issues = issues
	}

	if result.Details == nil {
		result.Details = make(map[string]interface{})
	}
	result.Details["hostconfig"] = details

	return result, nil
}

func (c *HostConfigCheck) getMTU(ctx context.Context) (int, error) {
	// Determine the default route interface from "ip route show default"
	routeOut, err := exec.CommandContext(ctx, "ip", "route", "show", "default").CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to get default route: %w", err)
	}

	devRe := regexp.MustCompile(`dev (\S+)`)
	devMatches := devRe.FindStringSubmatch(string(routeOut))
	if len(devMatches) < 2 {
		return 0, fmt.Errorf("no default route found")
	}
	iface := devMatches[1]

	// Get MTU for that specific interface
	linkOut, err := exec.CommandContext(ctx, "ip", "link", "show", iface).CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to get link info for %s: %w", iface, err)
	}

	mtuRe := regexp.MustCompile(`mtu (\d+)`)
	mtuMatches := mtuRe.FindStringSubmatch(string(linkOut))
	if len(mtuMatches) < 2 {
		return 0, fmt.Errorf("could not find MTU for interface %s", iface)
	}

	mtu, err := strconv.Atoi(mtuMatches[1])
	if err != nil {
		return 0, fmt.Errorf("failed to parse MTU for interface %s: %w", iface, err)
	}

	return mtu, nil
}

func (c *HostConfigCheck) getLoadAverage() (float64, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, fmt.Errorf("failed to read /proc/loadavg: %w", err)
	}

	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0, fmt.Errorf("unexpected /proc/loadavg format")
	}

	load1, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse load average: %w", err)
	}

	return load1, nil
}

func (c *HostConfigCheck) IsLocal() bool {
	return true
}

func (c *HostConfigCheck) AlwaysShow() bool {
	return true
}

func (c *HostConfigCheck) FormatSummary(details interface{}, debug bool) string {
	hc := extractHostConfig(details)
	if hc == nil {
		return ""
	}

	forwardingStr := "disabled"
	if ipFwd, _ := hc["ip_forwarding"].(bool); ipFwd {
		forwardingStr = "enabled"
	}

	mtu, _ := hc["mtu"].(float64)
	loadAvg, _ := hc["load_average"].(float64)
	numCPU, _ := hc["num_cpu"].(float64)
	summary := fmt.Sprintf("IP forwarding: %s, MTU: %d, Load avg: %.2f/%d CPUs", forwardingStr, int(mtu), loadAvg, int(numCPU))

	if issues, _ := hc["issues"].([]interface{}); len(issues) > 0 {
		strs := make([]string, len(issues))
		for i, issue := range issues {
			strs[i], _ = issue.(string)
		}
		summary += " | " + strings.Join(strs, "; ")
	}

	return summary
}

// extractHostConfig pulls the nested hostconfig map out of the raw details interface.
func extractHostConfig(details interface{}) map[string]interface{} {
	detailsMap, ok := details.(map[string]interface{})
	if !ok {
		return nil
	}
	hc, ok := detailsMap["hostconfig"].(map[string]interface{})
	if !ok {
		return nil
	}
	return hc
}

func NewHostConfigCheck() *HostConfigCheck {
	return &HostConfigCheck{}
}

func init() {
	DefaultRegistry.Register(NewHostConfigCheck())
}
