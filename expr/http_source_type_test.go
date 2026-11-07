package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPSourceTypeCompatibility(t *testing.T) {
	union := func(branches ...*AttributeExpr) *AttributeExpr {
		values := make([]*NamedAttributeExpr, len(branches))
		for index, branch := range branches {
			values[index] = &NamedAttributeExpr{Name: string(rune('a' + index)), Attribute: branch}
		}
		return &AttributeExpr{Type: &Union{Untagged: true, Values: values}}
	}
	for _, tc := range []struct {
		name, source string
		attribute    *AttributeExpr
		valid        bool
	}{
		{"mixed scalar", "string", union(&AttributeExpr{Type: String}, &AttributeExpr{Type: Int}), false},
		{"scalar alternatives", "string", union(&AttributeExpr{Type: String}, &AttributeExpr{Type: String}), true},
		{"array alternatives", "array", union(&AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String}}}, &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: Int}}}), true},
		{"array scalar mismatch", "array", union(&AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String}}}, &AttributeExpr{Type: Int}), false},
		{"integer subset number", "number", &AttributeExpr{Type: Int}, true},
		{"number is not integer", "integer", &AttributeExpr{Type: Float64}, false},
		{"integer enum", "integer", &AttributeExpr{Type: Float64, Validation: &ValidationExpr{Values: []any{1.0, 2.0}}}, true},
		{"fractional enum", "integer", &AttributeExpr{Type: Float64, Validation: &ValidationExpr{Values: []any{1.5}}}, false},
		{"Any enum refinement", "string", &AttributeExpr{Type: Any, Validation: &ValidationExpr{Values: []any{"ok"}}}, true},
		{"nullable scalar", "string", &AttributeExpr{Type: String, Nullable: true}, true},
		{"nullable branch", "string", union(&AttributeExpr{Type: String, Nullable: true}), true},
		{"null only refinement", "string", &AttributeExpr{Type: Any, Validation: &ValidationExpr{Values: []any{nil}}}, true},
		{"absent is no assertion", "absent", union(&AttributeExpr{Type: String}, &AttributeExpr{Type: Int}), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.attribute.Meta = MetaExpr{"http:encoding:sourceType": {tc.source}}
			body := &AttributeExpr{Type: &Object{{Name: "value", Attribute: tc.attribute}}, Meta: MetaExpr{"openapi:encoding:value:contentType": {"application/json"}}}
			_, err := HTTPFormBodyEncoding(body)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "source type")
				require.ErrorContains(t, err, "normalized value")
			}
			_, err = HTTPMultipartBodyEncoding(body)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "source type")
			}
		})
	}
}

func TestHTTPSourceTypeRefinedDomains(t *testing.T) {
	point := func(value float64) *NumericBound {
		bound, err := NewNumericBound(value)
		require.NoError(t, err)
		return &bound
	}
	for _, tc := range []struct {
		name      string
		attribute *AttributeExpr
		source    string
		valid     bool
	}{
		{"integer point", &AttributeExpr{Type: Float64, Validation: &ValidationExpr{Minimum: point(2), Maximum: point(2)}}, "integer", true},
		{"fractional point", &AttributeExpr{Type: Float64, Validation: &ValidationExpr{Minimum: point(2.5), Maximum: point(2.5)}}, "integer", false},
		{"numeric interval", &AttributeExpr{Type: Float64, Validation: &ValidationExpr{Minimum: point(1), Maximum: point(2)}}, "integer", false},
		{"conjoined enum", &AttributeExpr{Type: Any, Validation: &ValidationExpr{EnumClauses: [][]any{{"ok", 1}, {"ok", 2}}}}, "string", true},
		{"tagged union envelope", &AttributeExpr{Type: &Union{Values: []*NamedAttributeExpr{{Name: "text", Attribute: &AttributeExpr{Type: String}}}}}, "object", true},
		{"tagged union is not branch", &AttributeExpr{Type: &Union{Values: []*NamedAttributeExpr{{Name: "text", Attribute: &AttributeExpr{Type: String}}}}}, "string", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateHTTPSourceType(tc.attribute, tc.source)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "source type")
			}
		})
	}
}
