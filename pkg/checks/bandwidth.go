package checks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"

	"github.com/ryanelliottsmith/network-debugger/pkg/types"
)

const BandwidthDuration = 10

type BandwidthCheck struct{}

func (c *BandwidthCheck) Name() string {
	return "bandwidth"
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

	log.Printf("[bandwidth] Result: %.2f Mbps, %d retransmits", details.BandwidthMbps, details.Retransmits)
	result.Details = make(map[string]interface{})
	result.Details["bandwidth"] = details
	return result, nil
}

func (c *BandwidthCheck) runIperf3(ctx context.Context, target string) (types.BandwidthCheckDetails, error) {
	log.Printf("[bandwidth] Starting iperf3 test to %s for %d seconds", target, BandwidthDuration)
	cmd := exec.CommandContext(ctx, "iperf3", "-c", target, "-J", "-t", fmt.Sprintf("%d", BandwidthDuration))
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
	details := types.BandwidthCheckDetails{
		Protocol: "tcp",
		Duration: BandwidthDuration,
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

func NewBandwidthCheck() *BandwidthCheck {
	return &BandwidthCheck{}
}
