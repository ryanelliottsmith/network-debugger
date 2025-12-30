package checks

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/ryanelliottsmith/network-debugger/pkg/types"
)

type BandwidthCheck struct {
	Duration int
}

func (c *BandwidthCheck) Name() string {
	return "bandwidth"
}

func (c *BandwidthCheck) Run(ctx context.Context, target string) (*types.TestResult, error) {
	result := &types.TestResult{
		Check:  c.Name(),
		Target: target,
		Status: types.StatusPass,
	}

	duration := c.Duration
	if duration == 0 {
		duration = 10
	}

	cmd := exec.CommandContext(ctx, "iperf3", "-c", target, "-J", "-t", fmt.Sprintf("%d", duration))
	output, err := cmd.CombinedOutput()

	if err != nil {
		result.Status = types.StatusFail
		result.Error = fmt.Sprintf("iperf3 failed: %v", err)
		return result, nil
	}

	details, parseErr := c.parseIperf3Output(output)
	if parseErr != nil {
		result.Status = types.StatusFail
		result.Error = fmt.Sprintf("failed to parse iperf3 output: %v", parseErr)
		return result, nil
	}

	if result.Details == nil {
		result.Details = make(map[string]interface{})
	}
	result.Details["bandwidth"] = details

	return result, nil
}

func (c *BandwidthCheck) parseIperf3Output(output []byte) (types.BandwidthCheckDetails, error) {
	details := types.BandwidthCheckDetails{
		Protocol: "tcp",
		Duration: c.Duration,
	}

	var iperf3Result struct {
		End struct {
			SumSent struct {
				BitsPerSecond float64 `json:"bits_per_second"`
				Retransmits   int     `json:"retransmits"`
			} `json:"sum_sent"`
		} `json:"end"`
	}

	if err := json.Unmarshal(output, &iperf3Result); err != nil {
		return details, fmt.Errorf("failed to parse JSON: %w", err)
	}

	details.BandwidthMbps = iperf3Result.End.SumSent.BitsPerSecond / 1000000.0
	details.Retransmits = iperf3Result.End.SumSent.Retransmits

	return details, nil
}

func NewBandwidthCheck(duration int) *BandwidthCheck {
	return &BandwidthCheck{
		Duration: duration,
	}
}
