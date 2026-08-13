package openapiimport

import (
	"fmt"
	"sort"
	"strings"
)

// Diagnostic describes an OpenAPI construct that the strict import subset
// cannot represent without losing contract information.
type Diagnostic struct {
	Code    string
	Path    string
	Message string
}

// Diagnostics is a deterministic collection of unsupported-feature reports.
type Diagnostics []Diagnostic

// Error formats all diagnostics as a single error-style message.
func (d Diagnostics) Error() string {
	parts := make([]string, 0, len(d))
	for _, diagnostic := range d {
		parts = append(parts, fmt.Sprintf("%s: %s (%s)", diagnostic.Path, diagnostic.Message, diagnostic.Code))
	}
	return strings.Join(parts, "\n")
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
