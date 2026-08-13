// Package openapiversion owns validation and normalization of Loom's
// openapi:version metadata value.
package openapiversion

import (
	"fmt"
	"strings"
)

// Target identifies the OpenAPI renderer family selected by metadata.
type Target uint8

const (
	// Target31 selects OpenAPI 3.1 compatibility output.
	Target31 Target = iota + 1
	// Target32 selects canonical OpenAPI 3.2 output.
	Target32
)

// Parse validates value and returns its canonical renderer target. Empty
// values select the default OpenAPI 3.2 target.
func Parse(value string) (Target, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return Target32, nil
	}
	switch value {
	case "3.1", "3.1.0", "3.1.1", "3.1.2":
		return Target31, nil
	case "3.2", "3.2.0":
		return Target32, nil
	default:
		return 0, fmt.Errorf(
			"unsupported OpenAPI version %q; supported values are 3.1, 3.1.0, 3.1.1, 3.1.2, 3.2, and 3.2.0",
			value,
		)
	}
}
