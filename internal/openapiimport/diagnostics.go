package openapiimport

import (
	"fmt"
	"sort"
	"strings"
)

// Diagnostic describes an OpenAPI construct that the strict import subset
// cannot represent without losing contract information.
type Diagnostic struct {
	// Code identifies the importer limit class, for example "media-type".
	Code string
	// Path is the JSON Pointer of the offending construct in the source document.
	Path string
	// Message explains why the construct cannot be imported faithfully.
	Message string
}

// Diagnostics is a deterministic collection of unsupported-feature reports.
type Diagnostics []Diagnostic

var lossyDiagnosticCodes = map[string]struct{}{
	"examples":               {},
	"external-docs":          {},
	"header-deprecated":      {},
	"info-metadata":          {},
	"parameter-deprecated":   {},
	"path-metadata":          {},
	"response-summary":       {},
	"schema-allof-flattened": {},
	"schema-format":          {},
	"tag-metadata":           {},
}

// Error formats all diagnostics as a single error-style message.
func (d Diagnostics) Error() string {
	parts := make([]string, 0, len(d))
	for _, diagnostic := range d {
		parts = append(parts, fmt.Sprintf("%s: %s (%s)", diagnostic.Path, diagnostic.Message, diagnostic.Code))
	}
	return strings.Join(parts, "\n")
}

// Classify separates diagnostics that may be omitted with explicit user
// consent from diagnostics that always prevent a faithful import. Unknown
// diagnostic codes remain fatal so new importer limits cannot be silently
// downgraded.
func (d Diagnostics) Classify(allowLossy bool) (fatal, warnings Diagnostics) {
	if !allowLossy {
		return append(Diagnostics(nil), d...), nil
	}
	for _, diagnostic := range d {
		if _, ok := lossyDiagnosticCodes[diagnostic.Code]; ok {
			warnings = append(warnings, diagnostic)
			continue
		}
		fatal = append(fatal, diagnostic)
	}
	return fatal, warnings
}

func (d *Diagnostics) add(code, path, message string) {
	*d = append(*d, Diagnostic{Code: code, Path: path, Message: message})
}

func (d Diagnostics) sorted() Diagnostics {
	sort.SliceStable(d, func(i, j int) bool {
		if d[i].Path != d[j].Path {
			return d[i].Path < d[j].Path
		}
		if d[i].Code != d[j].Code {
			return d[i].Code < d[j].Code
		}
		return d[i].Message < d[j].Message
	})
	return d
}
