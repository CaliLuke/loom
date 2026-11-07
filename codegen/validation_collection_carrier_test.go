package codegen

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestCollectionValidationResolvesElementCarrier(t *testing.T) {
	item := &expr.UserTypeExpr{TypeName: "Item", AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Object{{Name: "name", Attribute: &expr.AttributeExpr{
			Type: expr.String, Validation: &expr.ValidationExpr{Pattern: "^[a-z]+$"},
		}}},
	}}
	for _, mapping := range []bool{false, true} {
		for _, extracted := range []bool{false, true} {
			for _, jsonPresence := range []bool{false, true} {
				t.Run(fmt.Sprintf("map=%t/extracted=%t/json=%t", mapping, extracted, jsonPresence), func(t *testing.T) {
					ctx := NewAttributeContext(true, false, false, "", NewNameScope())
					ctx.validationValue = extracted
					ctx.JSONPresence, ctx.CollectionElementPresence = jsonPresence, jsonPresence
					child := &expr.AttributeExpr{Type: item}
					require.Equal(t, "*Item", ctx.Scope.Ref(child, ""), "collection wrappers retain the referenced type")
					attribute := &expr.AttributeExpr{Type: &expr.Array{ElemType: child}}
					variable := "e"
					if mapping {
						attribute.Type = &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: child}
						variable = "v"
					}
					code := ValidationCode(attribute, nil, ctx, true, false, false, "body")
					if jsonPresence {
						require.Contains(t, code, "ValidateItem(actual,")
						require.NotContains(t, code, "ValidateItem(&actual,")
					} else {
						require.Contains(t, code, "ValidateItem("+variable+",")
						require.NotContains(t, code, "ValidateItem(&"+variable+",")
					}
					require.Equal(t, extracted, ctx.validationValue, "child mode must not mutate its parent")
				})
			}
		}
	}
}

func TestCollectionElementValidationUsesPhysicalReference(t *testing.T) {
	for _, collection := range []bool{false, true} {
		for _, extracted := range []bool{false, true} {
			for _, jsonPresence := range []bool{false, true} {
				t.Run(fmt.Sprintf("collection=%t/extracted=%t/json=%t", collection, extracted, jsonPresence), func(t *testing.T) {
					var typ expr.DataType = &expr.Object{}
					if collection {
						typ = &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}
					}
					attribute := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Element", AttributeExpr: &expr.AttributeExpr{Type: typ}}}
					parent := NewAttributeContext(true, false, false, "", NewNameScope())
					parent.validationValue = extracted
					parent.JSONPresence, parent.CollectionElementPresence = jsonPresence, jsonPresence
					child := collectionElementValidationContext(parent, attribute)
					require.Equal(t, collection, child.validationValue, "native and JSON wrappers retain the same underlying type reference")
					require.Equal(t, jsonPresence, child.JSONPresence, "object field layout is separate from current-value storage")
				})
			}
		}
	}
}

func TestCollectionValidationUsesCollectionStorage(t *testing.T) {
	object := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String}}}}
	union := &expr.AttributeExpr{Type: &expr.Union{TypeName: "Choice", Untagged: true, Values: []*expr.NamedAttributeExpr{
		{Name: "Text", Attribute: &expr.AttributeExpr{Type: expr.String}},
		{Name: "Number", Attribute: &expr.AttributeExpr{Type: expr.Int}},
	}}}
	for _, tc := range []struct {
		name      string
		attribute *expr.AttributeExpr
		value     bool
	}{
		{"inline object", object, false},
		{"named object", &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Object", AttributeExpr: object}}, false},
		{"inline union", union, true},
		{"named union", &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "NamedChoice", AttributeExpr: union}}, true},
		{"nullable object", &expr.AttributeExpr{Type: object.Type, Nullable: true}, true},
	} {
		for _, jsonPresence := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/json=%t", tc.name, jsonPresence), func(t *testing.T) {
				ctx := NewAttributeContext(true, false, false, "", NewNameScope())
				ctx.JSONPresence, ctx.CollectionElementPresence = jsonPresence, jsonPresence
				child := collectionElementValidationContext(ctx, tc.attribute)
				require.Equal(t, tc.value, child.validationValue)
			})
		}
	}
}
