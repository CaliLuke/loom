package ir

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestPreparedEnumSchemaIdentity(t *testing.T) {
	for _, required := range []bool{false, true} {
		t.Run(map[bool]string{false: "no local validation", true: "local required"}[required], func(t *testing.T) {
			prepared := func(n int) *expr.AttributeExpr {
				declaration := &expr.UserTypeExpr{TypeName: "Declared", AttributeExpr: &expr.AttributeExpr{
					Type:       &expr.Object{{Name: "n", Attribute: &expr.AttributeExpr{Type: expr.Int}}},
					Validation: &expr.ValidationExpr{Values: []any{map[string]any{"n": n}}},
				}}
				source := &expr.AttributeExpr{Type: declaration}
				if required {
					source.Validation = &expr.ValidationExpr{Required: []string{"n"}}
				}
				mapped := expr.NewMappedAttributeExpr(source)
				mapped.Map("wire_n", "n")
				return mapped.Attribute()
			}
			first, second := prepared(1), prepared(2)
			analyzer := NewAnalyzer(expr.NewRandom("prepared-enum"), false)
			require.NotEqual(t, analyzer.FingerprintAttribute(first), analyzer.FingerprintAttribute(second))
			require.NoError(t, analyzer.Err())
			var references []string
			for i, attribute := range []*expr.AttributeExpr{first, second} {
				body := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "PreparedBody", AttributeExpr: attribute}}
				reference := analyzer.AnalyzeSchema(body)
				require.NoError(t, analyzer.Err())
				references = append(references, reference.Ref)
				schema := analyzer.AnalyzeSchema(attribute)
				require.NoError(t, analyzer.Err())
				require.JSONEq(t, fmt.Sprintf(`[{"wire_n":%d}]`, i+1), canonicalJSON(schema.Enum))
			}
			require.NotEmpty(t, references[0])
			require.NotEqual(t, references[0], references[1], "different retained enums must not share a component")
		})
	}
}
