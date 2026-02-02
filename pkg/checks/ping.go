package checks

import (
	"context"
	"fmt"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"github.com/ryanelliottsmith/network-debugger/pkg/types"
)

// DefaultPingCount is the default number of ping packets to send
const DefaultPingCount = 10

type PingCheck struct {
	Count int
}

func (c *PingCheck) Name() string {
	return "ping"
}

func (c *PingCheck) Run(ctx context.Context, target string) (*types.TestResult, error) {
	result := &types.TestResult{
		Check:  c.Name(),
		Target: target,
		Status: types.StatusPass,
	}

	count := c.Count
	if count == 0 {
		count = DefaultPingCount
	}

	pinger, err := probing.NewPinger(target)
	if err != nil {
		result.Status = types.StatusFail
		result.Error = fmt.Sprintf("failed to create pinger: %v", err)
		return result, nil
	}

	// Use privileged mode (raw ICMP sockets) - requires CAP_NET_RAW
	pinger.SetPrivileged(true)
	pinger.Count = count
	pinger.Timeout = time.Duration(count) * time.Second
	pinger.Interval = 200 * time.Millisecond

	if err := pinger.RunWithContext(ctx); err != nil {
		result.Status = types.StatusFail
		result.Error = fmt.Sprintf("ping failed: %v", err)
		return result, nil
	}

	stats := pinger.Statistics()

	details := types.PingCheckDetails{
		PacketsSent:     stats.PacketsSent,
		PacketsReceived: stats.PacketsRecv,
		PacketLoss:      stats.PacketLoss,
		MinLatencyMS:    float64(stats.MinRtt.Microseconds()) / 1000.0,
		AvgLatencyMS:    float64(stats.AvgRtt.Microseconds()) / 1000.0,
		MaxLatencyMS:    float64(stats.MaxRtt.Microseconds()) / 1000.0,
	}

	if details.PacketLoss > 0 {
		result.Status = types.StatusFail
		result.Error = fmt.Sprintf("%.1f%% packet loss", details.PacketLoss)
	}

	if result.Details == nil {
		result.Details = make(map[string]interface{})
	}
	result.Details["ping"] = details

	return result, nil
}

func (c *PingCheck) IsLocal() bool {
	return false
}

func (c *PingCheck) AlwaysShow() bool {
	return false
}

func (c *PingCheck) FormatSummary(details interface{}, debug bool) string {
	if details == nil {
		return ""
	}

	// Details can be a map or struct depending on how it was serialized
	switch d := details.(type) {
	case map[string]interface{}:
		// Check for nested "ping" key (as stored in TestResult.Details)
		if ping, ok := d["ping"]; ok {
			if pingMap, ok := ping.(map[string]interface{}); ok {
				return formatPingMap(pingMap)
			}
		}
		// Try direct format
		return formatPingMap(d)
	}

	return ""
}

func formatPingMap(m map[string]interface{}) string {
	packetsSent, _ := m["packets_sent"].(float64)
	packetsReceived, _ := m["packets_received"].(float64)
	packetLoss, _ := m["packet_loss_percent"].(float64)
	avgLatency, _ := m["avg_latency_ms"].(float64)

	return fmt.Sprintf("%.0f sent, %.0f received, %.1f%% loss, avg %.2fms",
		packetsSent, packetsReceived, packetLoss, avgLatency)
}

func NewPingCheck(count int) *PingCheck {
	if count == 0 {
		count = DefaultPingCount
	}
	return &PingCheck{
		Count: count,
	}
}

func init() {
	DefaultRegistry.Register(NewPingCheck(0))
}
