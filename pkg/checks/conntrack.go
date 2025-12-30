package checks

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ryanelliottsmith/network-debugger/pkg/types"
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

	entries, err := c.readSysctl("/proc/sys/net/netfilter/nf_conntrack_count")
	if err != nil {
		issues = append(issues, fmt.Sprintf("failed to read conntrack count: %v", err))
	} else {
		details.Entries, _ = strconv.Atoi(entries)
	}

	maxEntries, err := c.readSysctl("/proc/sys/net/netfilter/nf_conntrack_max")
	if err != nil {
		issues = append(issues, fmt.Sprintf("failed to read conntrack max: %v", err))
	} else {
		details.MaxEntries, _ = strconv.Atoi(maxEntries)
	}

	stats, err := c.readConntrackStats()
	if err != nil {
		issues = append(issues, fmt.Sprintf("failed to read conntrack stats: %v", err))
	} else {
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

func (c *ConntrackCheck) readSysctl(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
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

func NewConntrackCheck() *ConntrackCheck {
	return &ConntrackCheck{}
}
