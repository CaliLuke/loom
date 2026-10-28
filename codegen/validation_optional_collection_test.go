package codegen

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestOptionalCollectionValidationPresence(t *testing.T) {
	minimum, maximum := 2, 4
	for _, typ := range []struct {
		name   string
		typeOf expr.DataType
	}{
		{"array", &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}},
		{"map", &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}},
	} {
		for _, required := range []bool{false, true} {
			for _, pointer := range []bool{false, true} {
				for _, nilIsEmpty := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/required=%t/pointer=%t/nilIsEmpty=%t", typ.name, required, pointer, nilIsEmpty), func(t *testing.T) {
						attribute := &expr.AttributeExpr{Type: typ.typeOf, Validation: &expr.ValidationExpr{MinLength: &minimum, MaxLength: &maximum}}
						ctx := NewAttributeContext(pointer, false, true, "", NewNameScope())
						ctx.CollectionNilIsEmpty = nilIsEmpty
						ctx = ctx.Dup()
						code := ValidationCode(attribute, nil, ctx, required, false, false, "value")
						require.Equal(t, !required && !nilIsEmpty, strings.HasPrefix(code, "if value != nil {"), code)
						require.Contains(t, code, "len(value) < 2")
						require.Contains(t, code, "len(value) > 4")
					})
				}
			}
		}
	}
}
