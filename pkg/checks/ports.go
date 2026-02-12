package checks

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ryanelliottsmith/network-debugger/pkg/types"
)

type PortsCheck struct {
	Ports []types.PortCheck
}

func (c *PortsCheck) Name() string {
	return "ports"
}

func (c *PortsCheck) Run(ctx context.Context, target string) (*types.TestResult, error) {
	result := &types.TestResult{
		Check:   c.Name(),
		Target:  target,
		Status:  types.StatusPass,
		Details: make(map[string]interface{}),
	}

	var portResults []types.PortCheckDetails
	var failedPorts []string

	for _, port := range c.Ports {
		portResult := c.checkPort(ctx, target, port)
		portResults = append(portResults, portResult)

		if !portResult.Open {
			failedPorts = append(failedPorts, fmt.Sprintf("%d/%s:%s", port.Port, port.Protocol, port.Name))
		}
	}

	result.Details["ports"] = portResults

	if len(failedPorts) > 0 {
		result.Status = types.StatusFail
		result.Error = strings.Join(failedPorts, ", ")
		result.Details["failed_ports"] = failedPorts
	}

	return result, nil
}

func (c *PortsCheck) checkPort(ctx context.Context, host string, port types.PortCheck) types.PortCheckDetails {
	details := types.PortCheckDetails{
		Port:     port.Port,
		Protocol: port.Protocol,
		Open:     false,
	}

	address := fmt.Sprintf("%s:%d", host, port.Port)
	start := time.Now()

	if port.Protocol == "tcp" {
		dialer := &net.Dialer{}
		conn, err := dialer.DialContext(ctx, "tcp", address)
		if err == nil {
			defer conn.Close()
			details.Open = true
			details.LatencyMS = float64(time.Since(start).Microseconds()) / 1000.0
		} else {
			details.Error = err.Error()
		}
	} else if port.Protocol == "udp" {
		// Use DialContext to create a "connected" UDP socket.
		// This is required to receive ICMP "Connection Refused" errors on Read.
		dialer := &net.Dialer{}
		conn, err := dialer.DialContext(ctx, "udp", address)
		if err == nil {
			defer conn.Close()

			// Set a deadline from context or default to 2 seconds
			deadline, ok := ctx.Deadline()
			if !ok {
				deadline = time.Now().Add(2 * time.Second)
			}
			conn.SetDeadline(deadline)

			// Write a dummy byte
			_, err = conn.Write([]byte{0})
			if err == nil {
				b := make([]byte, 1)
				_, readErr := conn.Read(b)

				if readErr == nil {
					// We received data, so it's definitely OPEN
					details.Open = true
					details.LatencyMS = float64(time.Since(start).Microseconds()) / 1000.0
				} else {
					details.Error = readErr.Error()
				}
			} else {
				details.Error = err.Error()
			}
		} else {
			details.Error = err.Error()
		}
	}

	return details
}

func (c *PortsCheck) IsLocal() bool {
	return false
}

func (c *PortsCheck) AlwaysShow() bool {
	return false
}

func (c *PortsCheck) FormatSummary(details interface{}, debug bool) string {
	if details == nil {
		return ""
	}

	detailsMap, ok := details.(map[string]interface{})
	if !ok {
		return ""
	}

	portsRaw, ok := detailsMap["ports"]
	if !ok {
		return ""
	}

	portsList, ok := portsRaw.([]interface{})
	if !ok {
		return ""
	}

	var open, total int
	var portDetails []string

	for _, p := range portsList {
		portMap, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		total++

		port := int(portMap["port"].(float64))
		protocol := portMap["protocol"].(string)
		isOpen, _ := portMap["open"].(bool)

		if isOpen {
			open++
			if debug {
				latency, _ := portMap["latency_ms"].(float64)
				portDetails = append(portDetails, fmt.Sprintf("%d/%s: %.2fms", port, protocol, latency))
			}
		} else {
			if debug {
				msg := fmt.Sprintf("%d/%s: CLOSED", port, protocol)
				if errStr, ok := portMap["error"].(string); ok && errStr != "" {
					msg = fmt.Sprintf("%s (%s)", msg, errStr)
				}
				portDetails = append(portDetails, msg)
			}
		}
	}

	summary := fmt.Sprintf("%d/%d open", open, total)
	if debug && len(portDetails) > 0 {
		return summary + " | " + strings.Join(portDetails, ", ")
	}
	return summary
}

func NewPortsCheck(ports []types.PortCheck) *PortsCheck {
	if len(ports) == 0 {
		ports = types.DefaultPorts()
	}
	return &PortsCheck{
		Ports: ports,
	}
}

func init() {
	DefaultRegistry.Register(NewPortsCheck(nil))
}
