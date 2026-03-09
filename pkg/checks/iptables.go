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

func (c *IptablesCheck) IsLocal() bool {
	return true
}

func (c *IptablesCheck) HostNetworkOnly() bool {
	return true
}

func (c *IptablesCheck) AlwaysShow() bool {
	return false
}

func (c *IptablesCheck) FormatSummary(details interface{}, debug bool) string {
	if details == nil {
		return ""
	}

	detailsMap, ok := details.(map[string]interface{})
	if !ok {
		return ""
	}

	iptablesRaw, ok := detailsMap["iptables"]
	if !ok {
		return ""
	}

	iptablesMap, ok := iptablesRaw.(map[string]interface{})
	if !ok {
		return ""
	}

	legacyCount, _ := iptablesMap["legacy_rule_count"].(float64)
	nftCount, _ := iptablesMap["nftable_rule_count"].(float64)

	summary := fmt.Sprintf("%.0f legacy, %.0f nftables rules", legacyCount, nftCount)

	if issuesRaw, ok := iptablesMap["issues"]; ok {
		if issues, ok := issuesRaw.([]interface{}); ok && len(issues) > 0 {
			summary += fmt.Sprintf(" | %d issues", len(issues))
		}
	}

	if debug {
		return summary
	}

	return "OK"
}

func NewIptablesCheck() *IptablesCheck {
	return &IptablesCheck{}
}

func init() {
	DefaultRegistry.Register(NewIptablesCheck())
}
