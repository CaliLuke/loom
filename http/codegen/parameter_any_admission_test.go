package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestTextAnyParameterAdmission(t *testing.T) {
	for _, tc := range []struct {
		name       string
		design     func()
		diagnostic string
	}{
		{"query-any", testdata.PayloadQueryAnyDSL, "nullable values require a JSON codec"},
		{"query-any-validate", testdata.PayloadQueryAnyValidateDSL, "value shape has no supported parameter codec"},
		{"query-array-any", testdata.PayloadQueryArrayAnyDSL, "value shape has no supported parameter codec"},
		{"query-array-any-validate", testdata.PayloadQueryArrayAnyValidateDSL, "value shape has no supported parameter codec"},
		{"path-interface-slice", testdata.PathInterfaceSliceParamDSL, "value shape has no supported parameter codec"},
		{"response-header-any", testdata.ResultHeaderAnyDSL, "nullable values require a JSON codec"},
		{"response-header-array-any", testdata.ResultHeaderArrayAnyDSL, "value shape has no supported parameter codec"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.ErrorContains(t, expr.RunInvalidDSL(t, tc.design), tc.diagnostic)
		})
	}
}
