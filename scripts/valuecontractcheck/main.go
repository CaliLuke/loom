// Command valuecontractcheck verifies that enabled Go/Lean conformance actually
// executed its required case families and passed, rather than skipping silently.
package main

import (
	"bufio"
	"encoding/json/v2"
	"flag"
	"fmt"
	"io"
	"os"
)

type (
	testEvent struct {
		Action  string
		Package string
		Test    string
	}
	conformanceReport struct {
		// Version identifies the reviewed report schema.
		Version int `json:"version"`
		// Cases contains only assertions that executed against their declared owner.
		Cases []conformanceCase `json:"cases"`
	}
	conformanceCase struct {
		// Name uniquely identifies the input and assertion across the corpus.
		Name string `json:"name"`
		// Owner identifies the implementation boundary checked by this case.
		Owner string `json:"owner"`
		// Passed records the complete assertion, not merely reference execution.
		Passed bool `json:"passed"`
	}
)

var requiredOwners = []string{"source-selection", "resolution", "projection", "codec-boundary", "negative-control"}

func main() {
	eventsPath := flag.String("events", "", "Go test -json output")
	reportPath := flag.String("report", "", "conformance-results.json path")
	flag.Parse()
	if err := run(*eventsPath, *reportPath); err != nil {
		fmt.Fprintln(os.Stderr, "value contract conformance:", err)
		os.Exit(1)
	}
}

func run(eventsPath, reportPath string) error {
	events, err := os.Open(eventsPath)
	if err != nil {
		return fmt.Errorf("open test events: %w", err)
	}
	eventErr := checkEvents(events)
	closeErr := events.Close()
	if eventErr != nil {
		return eventErr
	}
	if closeErr != nil {
		return fmt.Errorf("close test events: %w", closeErr)
	}
	report, err := os.ReadFile(reportPath)
	if err != nil {
		return fmt.Errorf("read conformance report: %w", err)
	}
	count, err := checkReport(report)
	if err != nil {
		return err
	}
	fmt.Printf("value contract conformance passed (%d executed assertion groups)\n", count)
	return nil
}

func checkEvents(input io.Reader) error {
	const testName = "TestLeanConformance"
	const packageName = "github.com/CaliLuke/loom/internal/valuecontract"
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), 16<<20)
	started, passed := 0, 0
	packagePassed := false
	for scanner.Scan() {
		var event testEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return fmt.Errorf("decode test event: %w", err)
		}
		if event.Package != packageName {
			continue
		}
		if event.Action == "fail" || event.Action == "skip" {
			return fmt.Errorf("conformance test failed or skipped: %s", event.Test)
		}
		if event.Test == testName {
			switch event.Action {
			case "run":
				started++
			case "pass":
				passed++
			}
		}
		if event.Test == "" && event.Action == "pass" {
			packagePassed = true
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read test events: %w", err)
	}
	if started != 1 || passed != 1 || !packagePassed {
		return fmt.Errorf("need one executed, passing %s and package completion", testName)
	}
	return nil
}

func checkReport(data []byte) (int, error) {
	var report conformanceReport
	if err := json.Unmarshal(data, &report, json.RejectUnknownMembers(true)); err != nil {
		return 0, fmt.Errorf("decode conformance report: %w", err)
	}
	if report.Version != 1 {
		return 0, fmt.Errorf("unsupported conformance report version %d", report.Version)
	}
	seen := make(map[string]bool)
	owners := make(map[string]int)
	for _, entry := range report.Cases {
		if entry.Name == "" || seen[entry.Name] || entry.Owner == "" || !entry.Passed {
			return 0, fmt.Errorf("invalid, duplicate or failed conformance case %q", entry.Name)
		}
		seen[entry.Name] = true
		owners[entry.Owner]++
	}
	for _, owner := range requiredOwners {
		if owners[owner] == 0 {
			return 0, fmt.Errorf("no executed conformance cases for %s", owner)
		}
	}
	return len(report.Cases), nil
}
