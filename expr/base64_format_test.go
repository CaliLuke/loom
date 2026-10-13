package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	loom "github.com/CaliLuke/loom/pkg"
)

func TestBase64FormatExamples(t *testing.T) {
	for _, tc := range []struct {
		name    string
		minimum int
		maximum int
		present bool
	}{
		{"empty", 0, 0, true},
		{"one quantum", 4, 4, true},
		{"rounded minimum", 5, 8, true},
		{"large exact", 64, 64, true},
		{"no quantum", 1, 3, false},
		{"allocation bound", 1 << 30, 1 << 30, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			attribute := &expr.AttributeExpr{
				Type:       expr.String,
				Validation: &expr.ValidationExpr{Format: expr.FormatBase64, MinLength: &tc.minimum, MaxLength: &tc.maximum},
			}
			var example any
			require.NotPanics(t, func() {
				example = attribute.Example(expr.NewRandom(tc.name))
			})
			if !tc.present {
				require.Nil(t, example)
				return
			}
			require.IsType(t, "", example)
			value := example.(string)
			require.NoError(t, loom.ValidateFormat("example", value, loom.FormatBase64))
			require.GreaterOrEqual(t, len(value), tc.minimum)
			require.LessOrEqual(t, len(value), tc.maximum)
		})
	}
}
