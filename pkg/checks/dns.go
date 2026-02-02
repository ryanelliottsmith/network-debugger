package checks

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ryanelliottsmith/network-debugger/pkg/types"
)

// DefaultDNSNames are the default DNS names to resolve when none are specified
var DefaultDNSNames = []string{"kubernetes.default.svc.cluster.local", "google.com"}

type DNSCheck struct {
	Names  []string
	Server string
}

func (c *DNSCheck) Name() string {
	return "dns"
}

func (c *DNSCheck) Run(ctx context.Context, target string) (*types.TestResult, error) {
	result := &types.TestResult{
		Check:  c.Name(),
		Target: target,
		Status: types.StatusPass,
	}

	names := c.Names
	if len(names) == 0 {
		names = DefaultDNSNames
	}

	var allDetails []types.DNSCheckDetails
	var errors []string

	for _, name := range names {
		details, err := c.resolveWithTiming(ctx, name)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", name, err))
			result.Status = types.StatusFail
		}
		allDetails = append(allDetails, details)
	}

	// Store details
	if result.Details == nil {
		result.Details = make(map[string]interface{})
	}
	result.Details["lookups"] = allDetails

	if len(errors) > 0 {
		result.Error = strings.Join(errors, "; ")
		result.Details["errors"] = errors
	}

	return result, nil
}

func (c *DNSCheck) resolveWithTiming(ctx context.Context, name string) (types.DNSCheckDetails, error) {
	details := types.DNSCheckDetails{
		Query:  name,
		Server: c.Server,
	}

	if details.Server == "" {
		details.Server = "system-default"
	}

	start := time.Now()

	var resolver *net.Resolver
	if c.Server != "" && c.Server != "system-default" {
		resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{}
				return d.DialContext(ctx, "udp", c.Server+":53")
			},
		}
	} else {
		resolver = net.DefaultResolver
	}

	ips, err := resolver.LookupIP(ctx, "ip", name)
	elapsed := time.Since(start)

	details.LatencyMS = float64(elapsed.Microseconds()) / 1000.0

	if err != nil {
		return details, err
	}

	for _, ip := range ips {
		details.ResolvedIPs = append(details.ResolvedIPs, ip.String())
	}

	return details, nil
}

func (c *DNSCheck) IsLocal() bool {
	return false
}

func (c *DNSCheck) AlwaysShow() bool {
	return false
}

func (c *DNSCheck) FormatSummary(details interface{}, debug bool) string {
	if details == nil {
		return ""
	}

	detailsMap, ok := details.(map[string]interface{})
	if !ok {
		return ""
	}

	lookupsRaw, ok := detailsMap["lookups"]
	if !ok {
		return ""
	}

	lookups, ok := lookupsRaw.([]interface{})
	if !ok {
		return ""
	}

	if len(lookups) == 0 {
		return ""
	}

	// Count successful lookups
	successCount := 0
	var lookupDetails []string

	for _, lookup := range lookups {
		lookupMap, ok := lookup.(map[string]interface{})
		if !ok {
			continue
		}

		query, _ := lookupMap["query"].(string)
		resolvedIPsRaw, hasIPs := lookupMap["resolved_ips"]
		latency, _ := lookupMap["latency_ms"].(float64)

		if hasIPs {
			resolvedIPs, ok := resolvedIPsRaw.([]interface{})
			if ok && len(resolvedIPs) > 0 {
				successCount++
			}
		}

		if debug && query != "" {
			if resolvedIPsRaw != nil {
				if resolvedIPs, ok := resolvedIPsRaw.([]interface{}); ok && len(resolvedIPs) > 0 {
					var ips []string
					for _, ip := range resolvedIPs {
						if ipStr, ok := ip.(string); ok {
							ips = append(ips, ipStr)
						}
					}
					lookupDetails = append(lookupDetails, fmt.Sprintf("%s: %s (%.2fms)", query, fmt.Sprintf("%v", ips), latency))
				} else {
					lookupDetails = append(lookupDetails, fmt.Sprintf("%s: failed", query))
				}
			}
		}
	}

	summary := fmt.Sprintf("%d/%d lookups OK", successCount, len(lookups))
	if debug && len(lookupDetails) > 0 {
		return summary + " | " + strings.Join(lookupDetails, ", ")
	}
	return summary
}

func NewDNSCheck(names []string, server string) *DNSCheck {
	if len(names) == 0 {
		names = DefaultDNSNames
	}
	return &DNSCheck{
		Names:  names,
		Server: server,
	}
}

func init() {
	DefaultRegistry.Register(NewDNSCheck(nil, ""))
}
