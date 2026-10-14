package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONRPCErrorSelector(t *testing.T) {
	field := &AttributeExpr{Type: String, Meta: MetaExpr{"struct:error:name": nil}}
	original := &ErrorExpr{Name: "left", AttributeExpr: &AttributeExpr{Type: &Object{&NamedAttributeExpr{Name: "kind:error_kind", Attribute: field}}}}
	for _, tc := range []struct {
		name string
		body *AttributeExpr
		want []string
	}{
		{"mapped", original.AttributeExpr, []string{"error_kind"}},
		{"root", &AttributeExpr{Type: String, Meta: MetaExpr{"origin:attribute": {"kind"}}}, []string{}},
		{"selected other object", &AttributeExpr{Type: &Object{&NamedAttributeExpr{Name: "kind", Attribute: &AttributeExpr{Type: String}}}, Meta: MetaExpr{"origin:attribute": {"details"}}}, nil},
		{"selected member", &AttributeExpr{Type: &Object{&NamedAttributeExpr{Name: "kind", Attribute: field}}}, []string{"kind"}},
		{"full tag", &AttributeExpr{Type: &Object{&NamedAttributeExpr{Name: "kind", Attribute: &AttributeExpr{Type: String, Meta: MetaExpr{"struct:tag:json": {"exact,omitempty"}}}}}}, []string{"exact"}},
		{"omitted", &AttributeExpr{Type: &Object{&NamedAttributeExpr{Name: "name", Attribute: &AttributeExpr{Type: String}}}}, nil},
		{"hidden", &AttributeExpr{Type: &Object{&NamedAttributeExpr{Name: "kind", Attribute: &AttributeExpr{Type: String, Meta: MetaExpr{"struct:tag:json": {"-"}}}}}}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := &HTTPErrorExpr{ErrorExpr: original, Response: &HTTPResponseExpr{Body: tc.body}}
			got := response.JSONRPCErrorSelector()
			require.Equal(t, tc.want, got)
			if len(got) > 0 {
				got[0] = "changed"
				require.Equal(t, tc.want, response.JSONRPCErrorSelector())
			}
		})
	}
}
