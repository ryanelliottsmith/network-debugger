package checks

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ryanelliottsmith/network-debugger/pkg/types"
)

type IptablesCheck struct{}

func (c *IptablesCheck) Name() string {
	return "iptables"
}

func (c *IptablesCheck) Run(ctx context.Context, target string) (*types.TestResult, error) {
	result := &types.TestResult{
		Check:  c.Name(),
		Target: "localhost",
		Status: types.StatusPass,
	}

	details := types.IptablesDetails{}
	var issues []string

	legacyCount, err := c.countIptablesRules(ctx, "iptables-legacy")
	if err == nil {
		details.LegacyRuleCount = legacyCount
	}

	nftCount, err := c.countIptablesRules(ctx, "iptables-nft")
	if err == nil {
		details.NftableRuleCount = nftCount
	}

	if details.LegacyRuleCount > 0 && details.NftableRuleCount > 0 {
		details.DuplicateRules = details.LegacyRuleCount + details.NftableRuleCount
		issues = append(issues, fmt.Sprintf("both iptables-legacy (%d rules) and iptables-nft (%d rules) are active, potential conflicts",
			details.LegacyRuleCount, details.NftableRuleCount))
	}

	backend, err := c.detectActiveBackend(ctx)
	if err == nil && backend != "" {
		issues = append(issues, fmt.Sprintf("detected active backend: %s", backend))
	}

	if len(issues) > 0 && details.DuplicateRules > 0 {
		result.Status = types.StatusFail
		result.Error = "iptables configuration conflict detected"
	}

	details.Issues = issues

	if result.Details == nil {
		result.Details = make(map[string]interface{})
	}
	result.Details["iptables"] = details

	return result, nil
}

func (c *IptablesCheck) countIptablesRules(ctx context.Context, binary string) (int, error) {
	cmd := exec.CommandContext(ctx, binary, "-S")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(output), "\n")
	count := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			count++
		}
	}

	return count, nil
}

func (c *IptablesCheck) detectActiveBackend(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "iptables", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	versionStr := strings.ToLower(string(output))
	if strings.Contains(versionStr, "nf_tables") {
		return "nftables", nil
	} else if strings.Contains(versionStr, "legacy") {
		return "legacy", nil
	}

	return "unknown", nil
}

func NewIptablesCheck() *IptablesCheck {
	return &IptablesCheck{}
}
