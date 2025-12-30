package checks

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/ryanelliottsmith/network-debugger/pkg/types"
)

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
		names = []string{"kubernetes.default.svc.cluster.local", "google.com"}
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
		result.Error = fmt.Sprintf("%d/%d lookups failed", len(errors), len(names))
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

func NewDNSCheck(names []string, server string) *DNSCheck {
	return &DNSCheck{
		Names:  names,
		Server: server,
	}
}
