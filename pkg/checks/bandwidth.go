package checks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"

	"github.com/ryanelliottsmith/network-debugger/pkg/types"
)

const BandwidthDuration = 10

type BandwidthCheck struct {
	iperfArgs string
}

func (c *BandwidthCheck) Name() string {
	return "bandwidth"
}

func (c *BandwidthCheck) Description() string {
	return "Tests network bandwidth between nodes using iperf. Results display throughput speed, TCP retransmit counts, and test duration."
}

func (c *BandwidthCheck) Run(ctx context.Context, target string) (*types.TestResult, error) {
	result := &types.TestResult{
		Check:  c.Name(),
		Target: target,
		Status: types.StatusPass,
	}

	details, err := c.runIperf3(ctx, target)
	if err != nil {
		result.Status = types.StatusFail
		result.Error = err.Error()
		log.Printf("[bandwidth] Failed: %v", err)
		return result, nil
	}

	log.Printf("[bandwidth] Result: SPEED: %.2f Mbps, RETRANSMITS: %d, RUNTIME: %ds", details.BandwidthMbps, details.Retransmits, details.Duration)
	result.Details = make(map[string]interface{})
	result.Details["bandwidth"] = details
	return result, nil
}

func (c *BandwidthCheck) runIperf3(ctx context.Context, target string) (types.BandwidthCheckDetails, error) {
	duration := BandwidthDuration
	args := []string{"-c", target, "-J"}
	if c.iperfArgs != "" {
		args = append(args, strings.Fields(c.iperfArgs)...)
		fields := strings.Fields(c.iperfArgs)
		for i, arg := range fields {
			if arg == "-t" && i+1 < len(fields) {
				if t, err := strconv.Atoi(fields[i+1]); err == nil {
					duration = t
				}
			}
		}
	} else {
		args = append(args, "-t", fmt.Sprintf("%d", BandwidthDuration))
	}
	log.Printf("[bandwidth] Starting iperf3 test to %s for %d seconds", target, duration)
	cmd := exec.CommandContext(ctx, "iperf3", args...)
	output, err := cmd.CombinedOutput()

	log.Printf("[bandwidth] iperf3 output length: %d bytes", len(output))
	if len(output) > 0 {
		log.Printf("[bandwidth] iperf3 raw output:\n%s", string(output))
	}

	if err != nil {
		log.Printf("[bandwidth] iperf3 command error: %v", err)
		return types.BandwidthCheckDetails{}, fmt.Errorf("iperf3 failed: %v", err)
	}

	details, parseErr := c.parseIperf3Output(output)
	if parseErr != nil {
		log.Printf("[bandwidth] parse error: %v", parseErr)
		return details, parseErr
	}

	return details, nil
}

func (c *BandwidthCheck) parseIperf3Output(output []byte) (types.BandwidthCheckDetails, error) {
	duration := BandwidthDuration
	if c.iperfArgs != "" {
		args := strings.Fields(c.iperfArgs)
		for i, arg := range args {
			if arg == "-t" && i+1 < len(args) {
				if t, err := strconv.Atoi(args[i+1]); err == nil {
					duration = t
				}
			}
		}
	}

	details := types.BandwidthCheckDetails{
		Protocol: "tcp",
		Duration: duration,
	}

	var iperf3Result struct {
		Error string `json:"error"`
		End   struct {
			SumSent struct {
				BitsPerSecond float64 `json:"bits_per_second"`
				Retransmits   int     `json:"retransmits"`
			} `json:"sum_sent"`
		} `json:"end"`
	}

	if err := json.Unmarshal(output, &iperf3Result); err != nil {
		return details, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if iperf3Result.Error != "" {
		return details, fmt.Errorf("iperf3: %s", iperf3Result.Error)
	}

	details.BandwidthMbps = iperf3Result.End.SumSent.BitsPerSecond / 1000000.0
	details.Retransmits = iperf3Result.End.SumSent.Retransmits

	if details.BandwidthMbps == 0 {
		return details, fmt.Errorf("iperf3 reported 0 bandwidth - test may have failed")
	}

	return details, nil
}

func (c *BandwidthCheck) IsLocal() bool {
	return false
}

func (c *BandwidthCheck) HostNetworkOnly() bool {
	return false
}

func (c *BandwidthCheck) AlwaysShow() bool {
	return true
}

func (c *BandwidthCheck) FormatSummary(details interface{}, debug bool) string {
	if details == nil {
		return ""
	}

	// Details can be a map or struct depending on how it was serialized
	switch d := details.(type) {
	case map[string]interface{}:
		// Check for nested "bandwidth" key (as stored in TestResult.Details)
		if bw, ok := d["bandwidth"]; ok {
			if bwMap, ok := bw.(map[string]interface{}); ok {
				return formatBandwidthMap(bwMap)
			}
		}
		// Try direct format
		return formatBandwidthMap(d)
	}

	return ""
}

func formatBandwidthMap(m map[string]interface{}) string {
	mbps, ok := m["bandwidth_mbps"].(float64)
	if !ok {
		return ""
	}
	retransmits, _ := m["retransmits"].(float64) // JSON numbers are float64
	duration, _ := m["duration_seconds"].(float64)

	durationStr := ""
	if duration > 0 {
		durationStr = fmt.Sprintf(", RUNTIME: %ds", int(duration))
	}

	// if mbps >= 1000 {
	// 	return fmt.Sprintf("SPEED: %.2f Gbps, RETRANSMITS: %d%s", mbps/1000, int(retransmits), durationStr)
	// }
	return fmt.Sprintf("SPEED: %.2f Mbps, RETRANSMITS: %d%s", mbps, int(retransmits), durationStr)
}

func NewBandwidthCheck(args string) *BandwidthCheck {
	return &BandwidthCheck{iperfArgs: args}
}

func init() {
	DefaultRegistry.Register(NewBandwidthCheck(""))
}
