package codegen

import (
	"strings"
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestJSONPhysicalLayoutReadiness(t *testing.T) {
	for _, tc := range []struct {
		name                                string
		field                               *expr.AttributeExpr
		required, pointer, defaults, decode bool
		service, transport                  string
	}{
		{"optional string", &expr.AttributeExpr{Type: expr.String}, false, false, true, false, "*string", "*string"},
		{"materialized string", &expr.AttributeExpr{Type: expr.String, DefaultValue: "fallback"}, false, false, true, false, "string", "string"},
		{"unmaterialized string", &expr.AttributeExpr{Type: expr.String, DefaultValue: "fallback"}, false, false, false, false, "*string", "*string"},
		{"required native", &expr.AttributeExpr{Type: expr.String}, true, false, true, false, "string", "string"},
		{"required decoder pointer", &expr.AttributeExpr{Type: expr.String}, true, true, false, true, "*string", "*string"},
		{"default decoder carrier", &expr.AttributeExpr{Type: expr.String, DefaultValue: "fallback"}, false, true, false, true, "*string", "loom.Optional[string]"},
		{"bytes", &expr.AttributeExpr{Type: expr.Bytes}, false, false, true, false, "[]byte", "[]byte"},
		{"array", &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}}, false, false, true, false, "[]string", "[]string"},
		{"map", &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}}, false, false, true, false, "map[string]string", "map[string]string"},
		{"named scalar", &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Label", AttributeExpr: &expr.AttributeExpr{Type: expr.String}}}, false, false, true, false, "*Label", "*Label"},
		{"named materialized scalar", &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Label", AttributeExpr: &expr.AttributeExpr{Type: expr.String}}, DefaultValue: "fallback"}, false, false, true, false, "Label", "Label"},
		{"named array", &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Names", AttributeExpr: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}}}}, false, false, true, false, "Names", "Names"},
		{"named map", &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Names", AttributeExpr: &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}}}}, false, false, true, false, "Names", "Names"},
		{"nullable", &expr.AttributeExpr{Type: expr.String, Nullable: true}, false, false, true, false, "loom.Nullable[string]", "loom.Nullable[string]"},
		{"raw JSON", &expr.AttributeExpr{Type: expr.Any}, false, false, true, false, "loom.JSONValue", "loom.JSONValue"},
		{"custom scalar", &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:field:type": {"time.Time", "time"}}}, false, false, true, false, "*time.Time", "*time.Time"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: tc.field}}}
			if tc.required {
				parent.Validation = &expr.ValidationExpr{Required: []string{"value"}}
			}
			service := codegen.NewNameScope().GoTypeDef(parent, tc.pointer, tc.defaults)
			transport := goTypeDef(codegen.NewNameScope(), parent, tc.pointer, tc.defaults, tc.decode)
			for _, output := range []struct{ name, definition, want string }{
				{"service", service, tc.service},
				{"transport", transport, tc.transport},
			} {
				tokens := strings.Fields(output.definition)
				require.GreaterOrEqual(t, len(tokens), 4, output.definition)
				require.Equal(t, "Value", tokens[2])
				require.Equal(t, output.want, tokens[3], output.name+" "+output.definition)
			}
		})
	}
}
