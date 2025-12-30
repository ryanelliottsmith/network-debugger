package checks

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"

	"github.com/ryanelliottsmith/network-debugger/pkg/types"
)

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
		count = 5
	}

	cmd := exec.CommandContext(ctx, "ping", "-c", strconv.Itoa(count), "-W", "1", target)
	output, err := cmd.CombinedOutput()

	if err != nil {
		result.Status = types.StatusFail
		result.Error = fmt.Sprintf("ping failed: %v", err)
		return result, nil
	}

	details, parseErr := c.parsePingOutput(string(output))
	if parseErr != nil {
		result.Status = types.StatusFail
		result.Error = fmt.Sprintf("failed to parse ping output: %v", parseErr)
		return result, nil
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

func (c *PingCheck) parsePingOutput(output string) (types.PingCheckDetails, error) {
	details := types.PingCheckDetails{}

	statsRe := regexp.MustCompile(`(\d+) packets transmitted, (\d+) received, ([\d.]+)% packet loss`)
	matches := statsRe.FindStringSubmatch(output)
	if len(matches) >= 4 {
		sent, _ := strconv.Atoi(matches[1])
		received, _ := strconv.Atoi(matches[2])
		loss, _ := strconv.ParseFloat(matches[3], 64)

		details.PacketsSent = sent
		details.PacketsReceived = received
		details.PacketLoss = loss
	}

	rttRe := regexp.MustCompile(`rtt min/avg/max/mdev = ([\d.]+)/([\d.]+)/([\d.]+)/([\d.]+) ms`)
	matches = rttRe.FindStringSubmatch(output)
	if len(matches) >= 4 {
		details.MinLatencyMS, _ = strconv.ParseFloat(matches[1], 64)
		details.AvgLatencyMS, _ = strconv.ParseFloat(matches[2], 64)
		details.MaxLatencyMS, _ = strconv.ParseFloat(matches[3], 64)
	} else {
		responseRe := regexp.MustCompile(`time=([\d.]+) ms`)
		responseMatches := responseRe.FindAllStringSubmatch(output, -1)
		if len(responseMatches) > 0 {
			var sum float64
			var min, max float64
			min = 999999.0

			for _, match := range responseMatches {
				if len(match) >= 2 {
					latency, _ := strconv.ParseFloat(match[1], 64)
					sum += latency
					if latency < min {
						min = latency
					}
					if latency > max {
						max = latency
					}
				}
			}

			if len(responseMatches) > 0 {
				details.MinLatencyMS = min
				details.AvgLatencyMS = sum / float64(len(responseMatches))
				details.MaxLatencyMS = max
			}
		}
	}

	return details, nil
}

func NewPingCheck(count int) *PingCheck {
	return &PingCheck{
		Count: count,
	}
}
