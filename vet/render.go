package vet

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

// Format identifies a supported vet report encoding.
type Format string

const (
	// FormatText writes diagnostics in a go vet-like line format.
	FormatText Format = "text"
	// FormatJSON writes the Report JSON representation.
	FormatJSON Format = "json"
	// FormatSARIF writes a SARIF 2.1.0 log.
	FormatSARIF Format = "sarif"
)

// ParseFormat validates a report format name.
func ParseFormat(value string) (Format, error) {
	switch Format(value) {
	case FormatText, FormatJSON, FormatSARIF:
		return Format(value), nil
	default:
		return "", fmt.Errorf("unknown vet format %q: expected text, json, or sarif", value)
	}
}

// WriteReport renders report using format.
func WriteReport(writer io.Writer, report Report, format Format) error {
	switch format {
	case FormatText:
		return writeTextReport(writer, report)
	case FormatJSON:
		if report.Diagnostics == nil {
			report.Diagnostics = []Diagnostic{}
		}
		return writeJSON(writer, report)
	case FormatSARIF:
		return writeJSON(writer, buildSARIF(report))
	default:
		return fmt.Errorf("unknown vet format %q", format)
	}
}

func writeTextReport(writer io.Writer, report Report) error {
	for _, diagnostic := range report.Diagnostics {
		location := diagnostic.Location.Path
		if diagnostic.Location.Line > 0 {
			location += fmt.Sprintf(":%d", diagnostic.Location.Line)
			if diagnostic.Location.Column > 0 {
				location += fmt.Sprintf(":%d", diagnostic.Location.Column)
			}
		}
		if _, err := fmt.Fprintf(writer, "%s: %s[%s]: %s\n", location, diagnostic.Severity, diagnostic.Rule, diagnostic.Message); err != nil {
			return err
		}
	}
	return nil
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

type sarifLog struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name  string          `json:"name"`
	Rules []sarifRuleInfo `json:"rules"`
}

type sarifRuleInfo struct {
	ID               string       `json:"id"`
	ShortDescription sarifMessage `json:"shortDescription"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations,omitempty"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifLocation struct {
	PhysicalLocation *sarifPhysicalLocation `json:"physicalLocation,omitempty"`
	LogicalLocations []sarifLogicalLocation `json:"logicalLocations,omitempty"`
}

type sarifLogicalLocation struct {
	FullyQualifiedName string `json:"fullyQualifiedName"`
	Kind               string `json:"kind"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           *sarifRegion          `json:"region,omitempty"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn,omitempty"`
}

func buildSARIF(report Report) sarifLog {
	ruleMessages := make(map[string]string)
	results := make([]sarifResult, 0, len(report.Diagnostics))
	for _, diagnostic := range report.Diagnostics {
		if _, exists := ruleMessages[diagnostic.Rule]; !exists {
			ruleMessages[diagnostic.Rule] = diagnostic.Message
		}
		result := sarifResult{
			RuleID:  diagnostic.Rule,
			Level:   sarifLevel(diagnostic.Severity),
			Message: sarifMessage{Text: diagnostic.Message},
		}
		if diagnostic.Location.Line > 0 {
			result.Locations = []sarifLocation{{
				PhysicalLocation: &sarifPhysicalLocation{
					ArtifactLocation: sarifArtifactLocation{URI: diagnostic.Location.Path},
					Region: &sarifRegion{
						StartLine:   diagnostic.Location.Line,
						StartColumn: diagnostic.Location.Column,
					},
				},
			}}
		} else if diagnostic.Location.Path != "" {
			result.Locations = []sarifLocation{{
				LogicalLocations: []sarifLogicalLocation{{
					FullyQualifiedName: diagnostic.Location.Path,
					Kind:               "Loom design expression",
				}},
			}}
		}
		results = append(results, result)
	}
	ruleIDs := make([]string, 0, len(ruleMessages))
	for ruleID := range ruleMessages {
		ruleIDs = append(ruleIDs, ruleID)
	}
	sort.Strings(ruleIDs)
	rules := make([]sarifRuleInfo, 0, len(ruleIDs))
	for _, ruleID := range ruleIDs {
		rules = append(rules, sarifRuleInfo{
			ID:               ruleID,
			ShortDescription: sarifMessage{Text: ruleMessages[ruleID]},
		})
	}
	return sarifLog{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs: []sarifRun{{
			Tool:    sarifTool{Driver: sarifDriver{Name: "loom vet", Rules: rules}},
			Results: results,
		}},
	}
}

func sarifLevel(severity Severity) string {
	if severity == SeverityError {
		return "error"
	}
	return "warning"
}
