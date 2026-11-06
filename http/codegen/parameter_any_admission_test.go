package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestTextAnyParameterAdmission(t *testing.T) {
	for _, tc := range []struct {
		name   string
		design func()
	}{
		{"query-any", testdata.PayloadQueryAnyDSL},
		{"query-any-validate", testdata.PayloadQueryAnyValidateDSL},
		{"query-array-any", testdata.PayloadQueryArrayAnyDSL},
		{"query-array-any-validate", testdata.PayloadQueryArrayAnyValidateDSL},
		{"path-interface-slice", testdata.PathInterfaceSliceParamDSL},
		{"response-header-any", testdata.ResultHeaderAnyDSL},
		{"response-header-array-any", testdata.ResultHeaderArrayAnyDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.ErrorContains(t, expr.RunInvalidDSL(t, tc.design), "value shape has no supported parameter codec")
		})
	}
}
