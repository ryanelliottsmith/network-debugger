package checks

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"

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

	// TODO: Compare MTU between nodes
	mtu, err := c.getMTU(ctx)
	if err != nil {
		issues = append(issues, fmt.Sprintf("failed to get MTU: %v", err))
	} else {
		details.MTU = mtu
		if mtu < 1450 {
			issues = append(issues, fmt.Sprintf("MTU is low (%d), may cause issues with overlay networks", mtu))
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
		result.Error = fmt.Sprintf("%d configuration issues found", len(issues))
		details.Issues = issues
	}

	if result.Details == nil {
		result.Details = make(map[string]interface{})
	}
	result.Details["hostconfig"] = details

	return result, nil
}

func (c *HostConfigCheck) getMTU(ctx context.Context) (int, error) {
	cmd := exec.CommandContext(ctx, "ip", "link", "show")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to run ip link: %w", err)
	}

	re := regexp.MustCompile(`mtu (\d+)`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) >= 2 {
		mtu, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, err
		}
		return mtu, nil
	}

	return 0, fmt.Errorf("could not find MTU in ip link output")
}

func NewHostConfigCheck() *HostConfigCheck {
	return &HostConfigCheck{}
}
