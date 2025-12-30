package output

import (
	"encoding/json"
	"fmt"
	"os"
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

func FormatEvents(events []*types.Event, format string) error {
	switch format {
	case "json":
		return printJSON(events)
	case "yaml":
		return printYAML(events)
	case "table":
		return printEventsTable(events)
	default:
		return fmt.Errorf("unknown format: %s", format)
	}
}

func printEventsTable(events []*types.Event) error {
	if len(events) == 0 {
		fmt.Println("No test results collected.")
		return nil
	}

	// Count results by status
	passed := 0
	failed := 0
	errors := 0

	// Print test results
	fmt.Println("Test Results:")
	fmt.Println(fmt.Sprintf("%-15s %-30s %-15s %-10s %s", "Node", "Check", "Target", "Status", "Details"))
	fmt.Println(fmt.Sprintf("%-15s %-30s %-15s %-10s %s", strings.Repeat("-", 15), strings.Repeat("-", 30), strings.Repeat("-", 15), strings.Repeat("-", 10), strings.Repeat("-", 20)))

	for _, event := range events {
		if event.Type == types.EventTypeTestResult {
			status := "✓ PASS"
			if event.Status == "fail" {
				status = "✗ FAIL"
				failed++
			} else {
				passed++
			}

			details := ""
			if event.Error != "" {
				details = event.Error
			}

			node := event.Node
			if len(node) > 15 {
				node = node[:12] + "..."
			}

			target := event.Target
			if len(target) > 15 {
				target = target[:12] + "..."
			}

			fmt.Printf("%-15s %-30s %-15s %-10s %s\n", node, event.Check, target, status, details)
		} else if event.Type == types.EventTypeError {
			errors++
		}
	}

	// Print summary
	fmt.Println()
	fmt.Printf("Summary: %d passed, %d failed, %d errors\n", passed, failed, errors)

	return nil
}
