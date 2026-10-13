package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func TestElementAdmissionFallibility(t *testing.T) {
	leaf := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Leaf", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}}}}
	scalar := &expr.AttributeExpr{Type: expr.String}
	array := func(elem *expr.AttributeExpr) *expr.AttributeExpr {
		return &expr.AttributeExpr{Type: &expr.Array{ElemType: elem}}
	}
	mapping := func(elem *expr.AttributeExpr) *expr.AttributeExpr {
		return &expr.AttributeExpr{Type: &expr.Map{KeyType: scalar, ElemType: elem}}
	}
	wrapper := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "List", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{{Name: "field", Attribute: array(scalar)}}}}}
	for _, tc := range []struct {
		name           string
		source, target *expr.AttributeExpr
		want           bool
	}{
		{"optional object", leaf, leaf, false},
		{"object array", array(leaf), array(leaf), true},
		{"object map", mapping(leaf), mapping(leaf), true},
		{"incoming collection wrapper", array(wrapper), array(array(scalar)), true},
		{"outgoing native collection", array(array(scalar)), array(wrapper), false},
		{"scalar", array(scalar), array(scalar), false},
		{"Any null", array(&expr.AttributeExpr{Type: expr.Any}), array(&expr.AttributeExpr{Type: expr.Any}), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, conversionCanFail(tc.source, tc.target, serviceTypeContext("svc", codegen.NewNameScope())))
		})
	}
}

func TestElementAdmissionPhysicalCarrier(t *testing.T) {
	scalar := &expr.AttributeExpr{Type: expr.String}
	object := &expr.AttributeExpr{Type: &expr.Object{}}
	named := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Leaf", AttributeExpr: object}}
	union := &expr.AttributeExpr{Type: &expr.Union{TypeName: "Choice", Values: []*expr.NamedAttributeExpr{{Name: "text", Attribute: scalar}}}}
	for _, tc := range []struct {
		name              string
		attribute         *expr.AttributeExpr
		service, protobuf bool
	}{
		{"anonymous empty object", object, true, true},
		{"named object", named, true, true},
		{"union", union, false, true},
		{"native array", &expr.AttributeExpr{Type: &expr.Array{ElemType: scalar}}, false, false},
		{"native map", &expr.AttributeExpr{Type: &expr.Map{KeyType: scalar, ElemType: scalar}}, false, false},
		{"scalar", scalar, false, false},
		{"bytes", &expr.AttributeExpr{Type: expr.Bytes}, false, false},
		{"Any", &expr.AttributeExpr{Type: expr.Any}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scope := codegen.NewNameScope()
			serviceContext := serviceTypeContext("svc", scope)
			protobufContext := protoBufTypeContext("pb", scope, false)
			require.Equal(t, tc.service, elementNeedsAdmission(tc.attribute, serviceContext))
			require.Equal(t, tc.protobuf, elementNeedsAdmission(tc.attribute, protobufContext))
			// Fallibility derives from the physical occurrence, never from the target's carrier.
			if tc.attribute.Type != expr.Any {
				collection := &expr.AttributeExpr{Type: &expr.Array{ElemType: tc.attribute}}
				require.Equal(t, tc.service, conversionCanFail(collection, collection, serviceContext))
				require.Equal(t, tc.protobuf, conversionCanFail(collection, collection, protobufContext))
			}
		})
	}
}
