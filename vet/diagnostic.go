// Package vet analyzes evaluated Loom designs and their consuming Go module.
package vet

import (
	"cmp"
	"slices"
)

// Severity classifies the impact of a diagnostic.
type Severity string

const (
	// SeverityError identifies a high-confidence adoption defect.
	SeverityError Severity = "error"
	// SeverityWarning identifies a heuristic design-quality finding.
	SeverityWarning Severity = "warning"
)

const (
	// RuleVetAnalysisIncomplete identifies a target-module package that source
	// analysis could not inspect completely.
	RuleVetAnalysisIncomplete = "vet-analysis-incomplete"
	// RuleRouteOutsideDesign identifies routes registered directly on a Loom mux.
	RuleRouteOutsideDesign = "route-outside-design"
	// RuleDuplicateRouteRegistration identifies exact manual routes registered
	// more than once.
	RuleDuplicateRouteRegistration = "duplicate-route-registration"
	// RuleRouteConflictWithDesign identifies a manual route that duplicates a
	// designed route.
	RuleRouteConflictWithDesign = "route-conflict-with-design"
	// RuleGeneratedVersionSkew identifies generated output from another Loom version.
	RuleGeneratedVersionSkew = "generated-version-skew"
	// RuleGeneratedDesignSkew identifies generated output from another evaluated design.
	RuleGeneratedDesignSkew = "generated-design-skew"
	// RuleServerErrorFault identifies 5xx errors that are not marked as faults.
	RuleServerErrorFault = "server-error-fault"
	// RuleRetryableError identifies retryable HTTP statuses without retry metadata.
	RuleRetryableError = "retryable-error"
	// RuleDescriptionMinimum identifies index descriptions without a lower bound.
	RuleDescriptionMinimum = "description-minimum"
	// RuleDescriptionRange identifies numeric range descriptions without matching bounds.
	RuleDescriptionRange = "description-range"
	// RuleStringFormat identifies semantic strings without a format or pattern.
	RuleStringFormat = "string-format"
	// RuleNormalizedRange identifies normalized numbers without zero-to-one bounds.
	RuleNormalizedRange = "normalized-range"
	// RuleUntypedSemanticAttribute identifies Any attributes with strong scalar semantics.
	RuleUntypedSemanticAttribute = "untyped-semantic-attribute"
	// RuleServiceNotMounted identifies designed HTTP services missing from configured mount packages.
	RuleServiceNotMounted = "service-not-mounted"
)

// SuppressionMeta is the DSL metadata key used to suppress vet diagnostics.
const SuppressionMeta = "loom:vet:ignore"

// Location identifies either a Go source position or an evaluated design path.
type Location struct {
	Path   string `json:"path"`
	Line   int    `json:"line,omitempty"`
	Column int    `json:"column,omitempty"`
}

// Diagnostic is one actionable Loom adoption finding.
type Diagnostic struct {
	Rule     string   `json:"rule"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Location Location `json:"location"`
}

// Report contains all diagnostics emitted by one vet run.
type Report struct {
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// HasDiagnostics reports whether the report contains findings.
func (r Report) HasDiagnostics() bool {
	return len(r.Diagnostics) > 0
}

// Sort orders diagnostics deterministically by location, severity, rule, and message.
func (r *Report) Sort() {
	slices.SortFunc(r.Diagnostics, func(left, right Diagnostic) int {
		if n := cmp.Compare(left.Location.Path, right.Location.Path); n != 0 {
			return n
		}
		if n := cmp.Compare(left.Location.Line, right.Location.Line); n != 0 {
			return n
		}
		if n := cmp.Compare(left.Location.Column, right.Location.Column); n != 0 {
			return n
		}
		if n := cmp.Compare(left.Severity, right.Severity); n != 0 {
			return n
		}
		if n := cmp.Compare(left.Rule, right.Rule); n != 0 {
			return n
		}
		return cmp.Compare(left.Message, right.Message)
	})
}
