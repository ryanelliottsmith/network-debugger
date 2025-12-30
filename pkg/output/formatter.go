package output

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/ryanelliottsmith/network-debugger/pkg/types"
	"gopkg.in/yaml.v3"
)

func PrintResult(result *types.TestResult, format string) error {
	switch format {
	case "json":
		return printJSON(result)
	case "yaml":
		return printYAML(result)
	case "table":
		return printTable(result)
	default:
		return fmt.Errorf("unknown format: %s", format)
	}
}

func PrintSummary(summary *types.TestSummary, format string) error {
	switch format {
	case "json":
		return printJSON(summary)
	case "yaml":
		return printYAML(summary)
	case "table":
		return printTableSummary(summary)
	default:
		return fmt.Errorf("unknown format: %s", format)
	}
}

func printJSON(v interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

func printYAML(v interface{}) error {
	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	return encoder.Encode(v)
}

func printTable(result *types.TestResult) error {
	status := "✓"
	if result.Status == types.StatusFail {
		status = "✗"
	} else if result.Status == types.StatusIncomplete {
		status = "?"
	}

	fmt.Printf("Check:    %s\n", result.Check)
	fmt.Printf("Target:   %s\n", result.Target)
	fmt.Printf("Status:   %s %s\n", status, result.Status)
	fmt.Printf("Duration: %v\n", result.Duration)

	if result.Error != "" {
		fmt.Printf("Error:    %s\n", result.Error)
	}

	if len(result.Details) > 0 {
		fmt.Println("\nDetails:")
		detailsJSON, _ := json.MarshalIndent(result.Details, "  ", "  ")
		fmt.Printf("  %s\n", string(detailsJSON))
	}

	return nil
}

func printTableSummary(summary *types.TestSummary) error {
	fmt.Printf("Test Summary\n")
	fmt.Printf("============\n\n")
	fmt.Printf("Total:      %d\n", summary.TotalTests)
	fmt.Printf("Passed:     %d\n", summary.Passed)
	fmt.Printf("Failed:     %d\n", summary.Failed)
	fmt.Printf("Incomplete: %d\n", summary.Incomplete)
	fmt.Printf("Duration:   %v\n\n", summary.Duration)

	if len(summary.Results) > 0 {
		fmt.Println("Results:")
		fmt.Println("--------")
		for _, result := range summary.Results {
			status := "✓"
			if result.Status == types.StatusFail {
				status = "✗"
			} else if result.Status == types.StatusIncomplete {
				status = "?"
			}

			fmt.Printf("[%s] %s -> %s (%s)\n", status, result.Node, result.Target, result.Check)
			if result.Error != "" {
				fmt.Printf("    Error: %s\n", result.Error)
			}
		}
	}

	return nil
}

func FormatEvents(events []*types.Event, format string, debug bool) error {
	switch format {
	case "json":
		return printJSON(events)
	case "yaml":
		return printYAML(events)
	case "table":
		return printEventsTable(events, debug)
	default:
		return fmt.Errorf("unknown format: %s", format)
	}
}

// isLocalCheck returns true for checks that run locally and don't have a meaningful target
func isLocalCheck(check string) bool {
	switch check {
	case "conntrack", "hostconfig", "iptables":
		return true
	default:
		return false
	}
}

// formatBandwidthDetails extracts bandwidth info from event details for display
func formatBandwidthDetails(details interface{}) string {
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

	if mbps >= 1000 {
		return fmt.Sprintf("%.2f Gbps, %d retransmits", mbps/1000, int(retransmits))
	}
	return fmt.Sprintf("%.2f Mbps, %d retransmits", mbps, int(retransmits))
}

func printEventsTable(events []*types.Event, debug bool) error {
	if len(events) == 0 {
		fmt.Println("No test results collected.")
		return nil
	}

	// Group events by check type
	checkOrder := []string{"ping", "dns", "ports", "bandwidth", "hostconfig", "conntrack", "iptables"}
	eventsByCheck := make(map[string][]*types.Event)

	passed := 0
	failed := 0
	errors := 0

	for _, event := range events {
		if event.Type == types.EventTypeTestResult {
			// Always count for summary
			if event.Status == "fail" {
				failed++
			} else {
				passed++
			}
			// Only include in display if failed OR debug mode
			if event.Status == "fail" || debug {
				eventsByCheck[event.Check] = append(eventsByCheck[event.Check], event)
			}
		} else if event.Type == types.EventTypeError {
			errors++
			eventsByCheck[event.Check] = append(eventsByCheck[event.Check], event)
		}
	}

	// Print grouped results
	for _, check := range checkOrder {
		checkEvents, ok := eventsByCheck[check]
		if !ok || len(checkEvents) == 0 {
			continue
		}

		// Sort bandwidth results by source node name
		if check == "bandwidth" {
			sort.Slice(checkEvents, func(i, j int) bool {
				return checkEvents[i].Node < checkEvents[j].Node
			})
		}

		// Print check header
		fmt.Printf("\n%s\n", strings.ToUpper(check))
		fmt.Println(strings.Repeat("-", 60))

		isLocal := isLocalCheck(check)

		if isLocal {
			// Local checks: Node, Status, Details
			fmt.Printf("%-20s %-10s %s\n", "Node", "Status", "Details")
			for _, event := range checkEvents {
				status := "✓ PASS"
				if event.Status == "fail" {
					status = "✗ FAIL"
				}

				node := event.Node
				if len(node) > 20 {
					node = node[:17] + "..."
				}

				details := ""
				if event.Error != "" {
					details = event.Error
				}

				fmt.Printf("%-20s %-10s %s\n", node, status, details)
			}
		} else {
			// Connectivity checks: Node, Target, Status, Details
			fmt.Printf("%-20s %-20s %-10s %s\n", "Node", "Target", "Status", "Details")
			for _, event := range checkEvents {
				status := "✓ PASS"
				if event.Status == "fail" {
					status = "✗ FAIL"
				}

				node := event.Node
				if len(node) > 20 {
					node = node[:17] + "..."
				}

				target := event.Target
				if len(target) > 20 {
					target = target[:17] + "..."
				}

				details := ""
				if event.Error != "" {
					details = event.Error
				} else if check == "bandwidth" {
					details = formatBandwidthDetails(event.Details)
				}

				fmt.Printf("%-20s %-20s %-10s %s\n", node, target, status, details)
			}
		}
	}

	// Handle any checks not in our predefined order
	for check, checkEvents := range eventsByCheck {
		found := false
		for _, ordered := range checkOrder {
			if check == ordered {
				found = true
				break
			}
		}
		if found || len(checkEvents) == 0 {
			continue
		}

		fmt.Printf("\n%s\n", strings.ToUpper(check))
		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("%-20s %-20s %-10s %s\n", "Node", "Target", "Status", "Details")

		for _, event := range checkEvents {
			status := "✓ PASS"
			if event.Status == "fail" {
				status = "✗ FAIL"
			}

			node := event.Node
			if len(node) > 20 {
				node = node[:17] + "..."
			}

			target := event.Target
			if len(target) > 20 {
				target = target[:17] + "..."
			}

			details := ""
			if event.Error != "" {
				details = event.Error
			}

			fmt.Printf("%-20s %-20s %-10s %s\n", node, target, status, details)
		}
	}

	// Print summary
	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Summary: %d passed, %d failed, %d errors\n", passed, failed, errors)

	return nil
}
