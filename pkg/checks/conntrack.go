package checks

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ryanelliottsmith/network-debugger/pkg/types"
	"github.com/ryanelliottsmith/network-debugger/pkg/util"
)

type ConntrackCheck struct{}

func (c *ConntrackCheck) Name() string {
	return "conntrack"
}

func (c *ConntrackCheck) Run(ctx context.Context, target string) (*types.TestResult, error) {
	result := &types.TestResult{
		Check:  c.Name(),
		Target: "localhost",
		Status: types.StatusPass,
	}

	details := types.ConntrackDetails{}
	var issues []string

	// Check if conntrack is available by looking for the count file
	entries, err := util.ReadSysctl("/proc/sys/net/netfilter/nf_conntrack_count")
	if os.IsNotExist(err) {
		// Conntrack module not loaded
		result.Status = types.StatusFail
		result.Error = "conntrack module not loaded"
		details.Issues = []string{"conntrack module not loaded"}
		result.Details = map[string]interface{}{"conntrack": details}
		return result, nil
	} else if err != nil {
		issues = append(issues, fmt.Sprintf("failed to read conntrack count: %v", err))
	} else {
		details.Entries, _ = strconv.Atoi(entries)
	}

	maxEntries, err := util.ReadSysctl("/proc/sys/net/netfilter/nf_conntrack_max")
	if err != nil && !os.IsNotExist(err) {
		issues = append(issues, fmt.Sprintf("failed to read conntrack max: %v", err))
	} else if err == nil {
		details.MaxEntries, _ = strconv.Atoi(maxEntries)
	}

	stats, err := c.readConntrackStats()
	if err != nil && !os.IsNotExist(err) {
		issues = append(issues, fmt.Sprintf("failed to read conntrack stats: %v", err))
	} else if err == nil {
		details.InsertsFailed = stats["insert_failed"]
		details.DropCount = stats["drop"]

		if details.InsertsFailed > 0 {
			issues = append(issues, fmt.Sprintf("conntrack insert failures detected: %d", details.InsertsFailed))
		}

		if details.DropCount > 0 {
			issues = append(issues, fmt.Sprintf("conntrack drops detected: %d", details.DropCount))
		}
	}

	if details.MaxEntries > 0 {
		utilization := float64(details.Entries) / float64(details.MaxEntries) * 100.0
		if utilization > 80.0 {
			issues = append(issues, fmt.Sprintf("conntrack table %.1f%% full (%d/%d)", utilization, details.Entries, details.MaxEntries))
		}
	}

	if len(issues) > 0 {
		result.Status = types.StatusFail
		result.Error = strings.Join(issues, "; ")
		details.Issues = issues
	}

	if result.Details == nil {
		result.Details = make(map[string]interface{})
	}
	result.Details["conntrack"] = details

	return result, nil
}

func (c *ConntrackCheck) readConntrackStats() (map[string]int, error) {
	stats := make(map[string]int)

	data, err := os.ReadFile("/proc/net/stat/nf_conntrack")
	if err != nil {
		return stats, err
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) < 2 {
		return stats, fmt.Errorf("unexpected conntrack stats format")
	}

	headers := strings.Fields(lines[0])
	values := strings.Fields(lines[1])

	if len(headers) != len(values) {
		return stats, fmt.Errorf("header/value count mismatch")
	}

	for i, header := range headers {
		if i < len(values) {
			val, err := strconv.ParseInt(values[i], 16, 64)
			if err == nil {
				stats[header] = int(val)
			}
		}
	}

	return stats, nil
}

func (c *ConntrackCheck) IsLocal() bool {
	return true
}

func (c *ConntrackCheck) AlwaysShow() bool {
	return false
}

func (c *ConntrackCheck) FormatSummary(details interface{}, debug bool) string {
	if details == nil {
		return ""
	}

	detailsMap, ok := details.(map[string]interface{})
	if !ok {
		return ""
	}

	conntrackRaw, ok := detailsMap["conntrack"]
	if !ok {
		return ""
	}

	conntrackMap, ok := conntrackRaw.(map[string]interface{})
	if !ok {
		return ""
	}

	// Get issues if present
	issuesRaw, hasIssues := conntrackMap["issues"]
	if hasIssues {
		if issues, ok := issuesRaw.([]interface{}); ok && len(issues) > 0 {
			return fmt.Sprintf("%d issues", len(issues))
		}
	}

	entries, _ := conntrackMap["entries"].(float64)
	maxEntries, _ := conntrackMap["max_entries"].(float64)

	if maxEntries > 0 {
		utilization := entries / maxEntries * 100.0
		return fmt.Sprintf("%.0f/%.0f entries (%.1f%%)", entries, maxEntries, utilization)
	} else if entries > 0 {
		return fmt.Sprintf("%.0f entries", entries)
	}

	return "OK"
}

func NewConntrackCheck() *ConntrackCheck {
	return &ConntrackCheck{}
}

func init() {
	DefaultRegistry.Register(NewConntrackCheck())
}
