package checks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/ryanelliottsmith/network-debugger/pkg/types"
)

const BandwidthDuration = 30

type BandwidthCheck struct {
	Debug bool
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

	const maxRetries = 3
	const retryDelay = 5 * time.Second

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		details, err := c.runIperf3(ctx, target, attempt)
		if err == nil {
			if c.Debug {
				log.Printf("[bandwidth] Result: %.2f Mbps, %d retransmits", details.BandwidthMbps, details.Retransmits)
			}
			if result.Details == nil {
				result.Details = make(map[string]interface{})
			}
			result.Details["bandwidth"] = details
			return result, nil
		}

		lastErr = err

		if strings.Contains(err.Error(), "server is busy") && attempt < maxRetries {
			if c.Debug {
				log.Printf("[bandwidth] Server busy, waiting %v before retry %d/%d", retryDelay, attempt+1, maxRetries)
			}
			select {
			case <-time.After(retryDelay):
			case <-ctx.Done():
				result.Status = types.StatusFail
				result.Error = fmt.Sprintf("cancelled while waiting to retry: %v", ctx.Err())
				return result, nil
			}
			continue
		}

		break
	}

	result.Status = types.StatusFail
	result.Error = lastErr.Error()
	if c.Debug {
		log.Printf("[bandwidth] Failed after %d attempts: %v", maxRetries, lastErr)
	}
	return result, nil
}

func (c *BandwidthCheck) runIperf3(ctx context.Context, target string, attempt int) (types.BandwidthCheckDetails, error) {
	if c.Debug {
		log.Printf("[bandwidth] Starting iperf3 test to %s for %d seconds (attempt %d)", target, BandwidthDuration, attempt)
	}
	cmd := exec.CommandContext(ctx, "iperf3", "-c", target, "-J", "-t", fmt.Sprintf("%d", BandwidthDuration))
	output, err := cmd.CombinedOutput()

	if c.Debug {
		log.Printf("[bandwidth] iperf3 output length: %d bytes", len(output))
		if len(output) > 0 {
			log.Printf("[bandwidth] iperf3 raw output:\n%s", string(output))
		}
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

func NewBandwidthCheck(debug bool) *BandwidthCheck {
	return &BandwidthCheck{
		Debug: debug,
	}
}
