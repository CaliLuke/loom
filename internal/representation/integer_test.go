package representation

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestClassifyIntegerConversion(t *testing.T) {
	stored := func(dt expr.DataType, goType string) *expr.AttributeExpr {
		attribute := &expr.AttributeExpr{Type: dt}
		if goType != "" {
			attribute.Meta = expr.MetaExpr{"struct:field:type": {goType}}
		}
		return attribute
	}
	big := &expr.UserTypeExpr{AttributeExpr: stored(expr.Int, "uint64"), TypeName: "Big"}
	cases := []struct {
		name           string
		source, target *expr.AttributeExpr
		checked        bool
		sourceOpaque   bool
		err            string
	}{
		{"native int to int", stored(expr.Int, ""), stored(expr.Int, ""), false, false, ""},
		{"narrow storage widens to wire", stored(expr.Int, "int8"), stored(expr.Int, ""), false, false, ""},
		{"wire narrows to storage", stored(expr.Int, ""), stored(expr.Int, "int8"), true, false, ""},
		{"int64 storage is checked through 386", stored(expr.Int, "int64"), stored(expr.Int, ""), true, false, ""},
		{"int32 widens to int64", stored(expr.Int64, "int32"), stored(expr.Int64, ""), false, false, ""},
		{"named uint64 under signed wire", &expr.AttributeExpr{Type: big}, stored(expr.Int, ""), true, false, ""},
		{"signed storage under unsigned wire", stored(expr.UInt, "int8"), stored(expr.UInt, ""), true, false, ""},
		{"opaque external storage", stored(expr.Int, "time.Duration"), stored(expr.Int, ""), true, true, ""},
		{"non-integer storage", stored(expr.Int, "float64"), stored(expr.Int, ""), false, false, "requires integer storage, got float64"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conversion, ok, err := ClassifyIntegerConversion(tc.source, tc.target)
			require.True(t, ok)
			if tc.err != "" {
				require.ErrorContains(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.checked, conversion.Checked)
			require.Equal(t, tc.sourceOpaque, conversion.SourceOpaque)
		})
	}
	_, ok, err := ClassifyIntegerConversion(stored(expr.String, ""), stored(expr.Int, ""))
	require.NoError(t, err)
	require.False(t, ok)
}

func TestIntegerConversionText(t *testing.T) {
	require.Equal(t, "const _ = ^time.Duration(0)\n", IntegerCapabilityAssertion("time.Duration"))
	require.Equal(t, "int64(w) != v || (v < 0) != (w < 0)", IntegerLossCondition("int64", "v", "w"))
	require.Equal(t, `fmt.Errorf("value %v cannot be represented as int", v)`, IntegerLossError("int", "v"))
}
