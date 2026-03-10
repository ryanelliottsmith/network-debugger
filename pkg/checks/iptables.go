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

func (c *IptablesCheck) Description() string {
	return "Compares iptables-nft and iptables-legacy rulesets to ensure kube-proxy and the CNI are utilizing the correct backend."
}

func (c *IptablesCheck) Run(ctx context.Context, target string) (*types.TestResult, error) {
	result := &types.TestResult{
		Check:  c.Name(),
		Target: "localhost",
		Status: types.StatusPass,
	}

	details := types.IptablesDetails{}
	var issues []string

	legacyCount, legacyHasKubeChains, legacyErr := c.analyzeIptablesBackend(ctx, "iptables-legacy")
	if legacyErr == nil {
		details.LegacyRuleCount = legacyCount
	}

	nftCount, nftHasKubeChains, nftErr := c.analyzeIptablesBackend(ctx, "iptables-nft")
	if nftErr == nil {
		details.NftableRuleCount = nftCount
	}

	if legacyHasKubeChains && nftHasKubeChains && legacyCount > 10 && nftCount > 10 {
		details.DuplicateRules = legacyCount + nftCount
		issues = append(issues, fmt.Sprintf("both iptables-legacy (%d rules) and iptables-nft (%d rules) have active KUBE/CNI chains — potential backend conflict",
			legacyCount, nftCount))
	}

	if len(issues) > 0 && details.DuplicateRules > 0 {
		result.Status = types.StatusFail
		result.Error = "iptables backend conflict detected: both legacy and nft have active Kubernetes/CNI chains"
	}

	details.Issues = issues

	if result.Details == nil {
		result.Details = make(map[string]interface{})
	}
	result.Details["iptables"] = details

	return result, nil
}

func (c *IptablesCheck) hasKubeCNIChains(output string) bool {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		tokens := strings.Fields(line)
		if len(tokens) >= 2 && (tokens[0] == "-N" || tokens[0] == "-A") {
			if strings.HasPrefix(tokens[1], "KUBE-") || strings.HasPrefix(tokens[1], "CNI-") {
				return true
			}
		}
	}
	return false
}

func (c *IptablesCheck) analyzeIptablesBackend(ctx context.Context, binary string) (int, bool, error) {
	filterCmd := exec.CommandContext(ctx, binary, "-S")
	filterOutput, filterErr := filterCmd.CombinedOutput()

	natCmd := exec.CommandContext(ctx, binary, "-t", "nat", "-S")
	natOutput, natErr := natCmd.CombinedOutput()

	if filterErr != nil && natErr != nil {
		return 0, false, fmt.Errorf("both filter and nat table commands failed: %v, %v", filterErr, natErr)
	}

	combined := string(filterOutput) + "\n" + string(natOutput)

	lines := strings.Split(combined, "\n")
	count := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			count++
		}
	}

	hasKubeChains := c.hasKubeCNIChains(combined)

	return count, hasKubeChains, nil
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
			strs := make([]string, 0, len(issues))
			for _, issue := range issues {
				if str, ok := issue.(string); ok {
					strs = append(strs, str)
				}
			}
			if len(strs) > 0 {
				summary += " | " + strings.Join(strs, "; ")
				return summary
			}
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
