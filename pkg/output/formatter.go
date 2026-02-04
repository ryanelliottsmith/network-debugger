package output

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/ryanelliottsmith/network-debugger/pkg/checks"
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

// calculateColumnWidths scans a set of events and returns the minimum column
// widths needed to display all Node and Target values without truncation.
// Each returned width is at least minWidth characters so short names don't look cramped.
func calculateColumnWidths(events []*types.Event, isLocal bool) (nodeWidth, targetWidth int) {
	const minWidth = 6
	nodeWidth = minWidth
	targetWidth = minWidth

	// Header labels set a floor as well
	if len("Node") > nodeWidth {
		nodeWidth = len("Node")
	}
	if !isLocal && len("Target") > targetWidth {
		targetWidth = len("Target")
	}

	for _, event := range events {
		if len(event.Node) > nodeWidth {
			nodeWidth = len(event.Node)
		}
		if !isLocal && len(event.Target) > targetWidth {
			targetWidth = len(event.Target)
		}
	}
	return nodeWidth, targetWidth
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

			// Check if this check should always show results
			check := checks.DefaultRegistry.Get(event.Check)
			alwaysShow := check != nil && check.AlwaysShow()

			// Only include in display if failed OR debug mode OR check says always show
			if event.Status == "fail" || debug || alwaysShow {
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

		// Get check from registry to check if it's local
		checkInstance := checks.DefaultRegistry.Get(check)
		isLocal := checkInstance != nil && checkInstance.IsLocal()

		// Special sorting for bandwidth - sort by source node name
		if check == "bandwidth" {
			sort.Slice(checkEvents, func(i, j int) bool {
				return checkEvents[i].Node < checkEvents[j].Node
			})
		}

		// Calculate dynamic column widths from actual data
		nodeWidth, targetWidth := calculateColumnWidths(checkEvents, isLocal)

		// Print check header
		fmt.Printf("\n%s\n", strings.ToUpper(check))
		if isLocal {
			// separator: nodeWidth + 3 + statusWidth(10) + 3 + len("Details")
			fmt.Println(strings.Repeat("-", nodeWidth+3+10+3+len("Details")))
			fmt.Printf("%-*s   %-10s   %s\n", nodeWidth, "Node", "Status", "Details")
			for _, event := range checkEvents {
				status := "✓ PASS"
				if event.Status == "fail" {
					status = "✗ FAIL"
				}

				details := ""
				if checkInstance != nil {
					details = checkInstance.FormatSummary(event.Details, debug)
				}
				if event.Error != "" {
					details = event.Error
				}

				fmt.Printf("%-*s   %-10s   %s\n", nodeWidth, event.Node, status, details)
			}
		} else {
			// separator: nodeWidth + 3 + targetWidth + 3 + statusWidth(10) + 3 + len("Details")
			fmt.Println(strings.Repeat("-", nodeWidth+3+targetWidth+3+10+3+len("Details")))
			fmt.Printf("%-*s   %-*s   %-10s   %s\n", nodeWidth, "Node", targetWidth, "Target", "Status", "Details")
			for _, event := range checkEvents {
				status := "✓ PASS"
				if event.Status == "fail" {
					status = "✗ FAIL"
				}

				details := ""
				if checkInstance != nil {
					details = checkInstance.FormatSummary(event.Details, debug)
				}

				if event.Error != "" {
					if details != "" {
						details = details + " | " + event.Error
					} else {
						details = event.Error
					}
				}

				fmt.Printf("%-*s   %-*s   %-10s   %s\n", nodeWidth, event.Node, targetWidth, event.Target, status, details)
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

		// Fallback checks are never local, so always show Node + Target
		nodeWidth, targetWidth := calculateColumnWidths(checkEvents, false)

		fmt.Printf("\n%s\n", strings.ToUpper(check))
		fmt.Println(strings.Repeat("-", nodeWidth+3+targetWidth+3+10+3+len("Details")))
		fmt.Printf("%-*s   %-*s   %-10s   %s\n", nodeWidth, "Node", targetWidth, "Target", "Status", "Details")

		for _, event := range checkEvents {
			status := "✓ PASS"
			if event.Status == "fail" {
				status = "✗ FAIL"
			}

			details := ""
			checkInstance := checks.DefaultRegistry.Get(check)
			if checkInstance != nil {
				details = checkInstance.FormatSummary(event.Details, debug)
			}

			if event.Error != "" {
				if details != "" {
					details = details + " | " + event.Error
				} else {
					details = event.Error
				}
			}

			fmt.Printf("%-*s   %-*s   %-10s   %s\n", nodeWidth, event.Node, targetWidth, event.Target, status, details)
		}
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Summary: %d passed, %d failed, %d errors\n", passed, failed, errors)

	return nil
}
