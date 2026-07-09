package output

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

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
	status := "PASS"
	if result.Status == types.StatusFail {
		status = "FAIL"
	} else if result.Status == types.StatusIncomplete {
		status = "UNKNOWN"
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "CHECK\tTARGET\tSTATUS\tDURATION\n")
	fmt.Fprintf(w, "%s\t%s\t%s\t%v\n", result.Check, result.Target, status, result.Duration)
	w.Flush()

	if result.Error != "" {
		fmt.Printf("\nError: %s\n", result.Error)
	}

	if len(result.Details) > 0 {
		fmt.Println("\nDetails:")
		detailsJSON, _ := json.MarshalIndent(result.Details, "  ", "  ")
		fmt.Printf("  %s\n", string(detailsJSON))
	}

	return nil
}

func printTableSummary(summary *types.TestSummary) error {
	fmt.Printf("TEST SUMMARY\n")
	fmt.Printf("============\n\n")

	w1 := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w1, "TOTAL\tPASSED\tFAILED\tINCOMPLETE\tDURATION\n")
	fmt.Fprintf(w1, "%d\t%d\t%d\t%d\t%v\n\n", summary.TotalTests, summary.Passed, summary.Failed, summary.Incomplete, summary.Duration)
	w1.Flush()

	if len(summary.Results) > 0 {
		fmt.Println("RESULTS")
		fmt.Println("-------")
		w2 := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintf(w2, "NODE\tTARGET\tCHECK\tSTATUS\tERROR\n")
		for _, result := range summary.Results {
			status := "PASS"
			if result.Status == types.StatusFail {
				status = "FAIL"
			} else if result.Status == types.StatusIncomplete {
				status = "UNKNOWN"
			}

			errStr := result.Error
			if errStr == "" {
				errStr = "-"
			}
			fmt.Fprintf(w2, "%s\t%s\t%s\t%s\t%s\n", result.Node, result.Target, result.Check, status, errStr)
		}
		w2.Flush()
	}

	return nil
}

func FormatEvents(events []*types.Event, format string, quiet bool) error {
	switch format {
	case "json":
		return printJSON(events)
	case "yaml":
		return printYAML(events)
	case "table":
		return printEventsTable(events, quiet)
	default:
		return fmt.Errorf("unknown format: %s", format)
	}
}

func printEventsTable(events []*types.Event, quiet bool) error {
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
			check := types.DefaultRegistry.Get(event.Check)
			alwaysShow := check != nil && check.AlwaysShow()

			// Only include in display if failed OR not quiet OR check says always show
			if event.Status == "fail" || !quiet || alwaysShow {
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
		checkInstance := types.DefaultRegistry.Get(check)
		isLocal := checkInstance != nil && checkInstance.IsLocal()

		// Special sorting for bandwidth - sort by source node name
		if check == "bandwidth" {
			sort.Slice(checkEvents, func(i, j int) bool {
				return checkEvents[i].Node < checkEvents[j].Node
			})
		}

		// Print check header
		fmt.Printf("\n%s\n", strings.ToUpper(check))
		if checkInstance != nil {
			if desc := checkInstance.Description(); desc != "" {
				fmt.Printf("%s\n", desc)
			}
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		if isLocal {
			fmt.Fprintf(w, "NODE\tSTATUS\tDETAILS\n")
			for _, event := range checkEvents {
				status := "PASS"
				if event.Status == "fail" {
					status = "FAIL"
				}

				details := ""
				if checkInstance != nil {
					details = checkInstance.FormatSummary(event.Details, quiet)
				}
				if event.Error != "" {
					if details != "" {
						details = details + " | " + event.Error
					} else {
						details = event.Error
					}
				}
				if details == "" {
					details = "-"
				}

				fmt.Fprintf(w, "%s\t%s\t%s\n", event.Node, status, details)
			}
		} else {
			fmt.Fprintf(w, "NODE\tTARGET\tSTATUS\tDETAILS\n")
			for _, event := range checkEvents {
				status := "PASS"
				if event.Status == "fail" {
					status = "FAIL"
				}

				details := ""
				if checkInstance != nil {
					details = checkInstance.FormatSummary(event.Details, quiet)
				}

				if event.Error != "" {
					if details != "" {
						details = details + " | " + event.Error
					} else {
						details = event.Error
					}
				}
				if details == "" {
					details = "-"
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", event.Node, event.Target, status, details)
			}
		}
		w.Flush()
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
		checkInstance := types.DefaultRegistry.Get(check)
		if checkInstance != nil {
			if desc := checkInstance.Description(); desc != "" {
				fmt.Printf("%s\n", desc)
			}
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintf(w, "NODE\tTARGET\tSTATUS\tDETAILS\n")

		for _, event := range checkEvents {
			status := "PASS"
			if event.Status == "fail" {
				status = "FAIL"
			}

			details := ""
			if checkInstance != nil {
				details = checkInstance.FormatSummary(event.Details, quiet)
			}

			if event.Error != "" {
				if details != "" {
					details = details + " | " + event.Error
				} else {
					details = event.Error
				}
			}
			if details == "" {
				details = "-"
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", event.Node, event.Target, status, details)
		}
		w.Flush()
	}

	fmt.Println()
	fmt.Printf("Summary: %d passed, %d failed, %d errors\n", passed, failed, errors)

	return nil
}
