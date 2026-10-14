package openapiimport

import (
	"fmt"

	"github.com/CaliLuke/loom/internal/patternsyntax"
)

func (a *analyzer) schemaPattern(schema *Schema, path string) {
	if schema.Pattern == "" {
		return
	}
	pattern, err := patternsyntax.FromECMA(schema.Pattern)
	if err != nil {
		schema.patternError = &Diagnostic{
			Code:    "schema-pattern",
			Path:    path + "/pattern",
			Message: fmt.Sprintf("cannot preserve ECMAScript pattern %q in Loom validation: %v", schema.Pattern, err),
		}
		a.diagnostics = append(a.diagnostics, *schema.patternError)
		return
	}
	schema.Pattern = pattern
}
