package expr

import (
	"testing"

	"github.com/CaliLuke/loom/eval"
	"github.com/stretchr/testify/require"
)

// TestHTTPWireNameEmpty rejects empty physical names without constraining other locations.
func TestHTTPWireNameEmpty(t *testing.T) {
	for _, location := range []string{"header", "cookie", "query", "path", "body"} {
		t.Run(location, func(t *testing.T) {
			plan := &httpWireBindingPlan{bindings: []*httpWireBinding{{location: location}}}
			verr := new(eval.ValidationErrors)
			plan.validate(&HTTPEndpointExpr{MethodExpr: &MethodExpr{Name: "show"}}, verr)
			if location == "header" || location == "cookie" {
				require.Len(t, verr.Errors, 1)
				require.Contains(t, verr.Error(), "nonempty HTTP token")
			} else {
				require.Empty(t, verr.Errors)
			}
		})
	}
}
