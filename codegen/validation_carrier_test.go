package codegen

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestAnonymousValidationCarrier(t *testing.T) {
	for _, nullable := range []bool{false, true} {
		for _, enum := range []bool{false, true} {
			t.Run(fmt.Sprintf("nullable=%t/enum=%t", nullable, enum), func(t *testing.T) {
				child := &expr.AttributeExpr{Type: &expr.Object{{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Pattern: "^[a-z]+$"}}}}}
				value := &expr.AttributeExpr{Nullable: nullable, Type: &expr.Object{{Name: "child", Attribute: child}}}
				if enum {
					value.Validation = &expr.ValidationExpr{Values: []any{map[string]any{"child": map[string]any{"text": "good"}}}}
				}
				parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: value}}}
				ctx := NewAttributeContext(true, false, false, "", NewNameScope())
				ctx.JSONPresence = !nullable
				code := ValidationCode(parent, nil, ctx, true, false, false, "body")
				require.Contains(t, code, "body.Value.Value()")
				require.NotContains(t, code, "if actual != nil", "unwrapped anonymous object is a struct value")
				if nullable {
					require.Contains(t, code, "if actual.Child != nil", "unwrapping does not clear native child field layout")
				}
			})
		}
	}
}
