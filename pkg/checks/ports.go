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

	var conn net.Conn
	var err error

	dialer := &net.Dialer{}

	if port.Protocol == "tcp" {
		conn, err = dialer.DialContext(ctx, "tcp", address)
	} else if port.Protocol == "udp" {
		conn, err = dialer.DialContext(ctx, "udp", address)
		if err == nil {
			details.Open = true
			details.LatencyMS = float64(time.Since(start).Microseconds()) / 1000.0
		}
	}

	if conn != nil {
		defer conn.Close()
	}

	elapsed := time.Since(start)

	if err == nil && port.Protocol == "tcp" {
		details.Open = true
		details.LatencyMS = float64(elapsed.Microseconds()) / 1000.0
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
				portDetails = append(portDetails, fmt.Sprintf("%d/%s: CLOSED", port, protocol))
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
