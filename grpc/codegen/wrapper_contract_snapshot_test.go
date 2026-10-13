package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func TestAnonymousCollectionContractSnapshot(t *testing.T) {
	t.Run("detached enum and length facts", func(t *testing.T) {
		minimum := 2
		original := wrapperContractAttribute(&expr.ValidationExpr{MinLength: &minimum}, &expr.ValidationExpr{Values: []any{"one"}})
		copy := expr.DupAtt(original)
		changed := expr.DupAtt(original)
		sd := &ServiceData{Name: "Contract", Scope: codegen.NewNameScope()}
		wrapNestedCollection(original, "", sd)
		minimum = 3
		unwrapAttr(original.Type.(expr.UserType).Attribute()).Type.(*expr.Array).ElemType.Validation.Values[0] = "two"
		wrapNestedCollection(copy, "", sd)
		require.Equal(t, original.Type.(expr.UserType).ID(), copy.Type.(expr.UserType).ID())
		changed.Validation.MinLength = &minimum
		wrapNestedCollection(changed, "", sd)
		require.NotEqual(t, original.Type.(expr.UserType).ID(), changed.Type.(expr.UserType).ID())
	})
	for _, reverse := range []bool{false, true} {
		t.Run(map[bool]string{false: "permissive first", true: "strict first"}[reverse], func(t *testing.T) {
			minimum := 2
			free := wrapperContractAttribute(nil, nil)
			strict := wrapperContractAttribute(&expr.ValidationExpr{MinLength: &minimum}, nil)
			copy := expr.DupAtt(strict)
			attrs := []*expr.AttributeExpr{free, strict}
			if reverse {
				attrs[0], attrs[1] = attrs[1], attrs[0]
			}
			sd := &ServiceData{Name: "Contract", Scope: codegen.NewNameScope()}
			for _, att := range attrs {
				wrapNestedCollection(att, "", sd)
			}
			wrapNestedCollection(copy, "", sd)
			require.NotEqual(t, free.Type.(expr.UserType).ID(), strict.Type.(expr.UserType).ID())
			require.Equal(t, strict.Type.(expr.UserType).ID(), copy.Type.(expr.UserType).ID())
		})
	}
}

func TestAnonymousCollectionNamedContractBoundary(t *testing.T) {
	node := &expr.UserTypeExpr{TypeName: "Node", UID: "Node", AttributeExpr: &expr.AttributeExpr{}}
	node.Type = &expr.Object{{Name: "next", Attribute: &expr.AttributeExpr{Type: node, Meta: expr.MetaExpr{"rpc:tag": {"1"}}}}}
	attribute := &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: node}}}
	copy := expr.DupAtt(attribute)
	sd := &ServiceData{Name: "Contract", Scope: codegen.NewNameScope()}
	wrapNestedCollection(attribute, "", sd)
	wrapNestedCollection(copy, "", sd)
	require.Equal(t, attribute.Type.(expr.UserType).ID(), copy.Type.(expr.UserType).ID(), "recursive named references retain declaration identity across copies")
}

func TestAnonymousCollectionPresenceContractIdentity(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*expr.AttributeExpr)
	}{
		{name: "element nullability", change: func(att *expr.AttributeExpr) {
			expr.AsArray(att.Type).ElemType.Nullable = true
		}},
		{name: "required elements", change: func(att *expr.AttributeExpr) {
			expr.AsArray(att.Type).NonNullableElems = true
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			left := wrapperContractAttribute(nil, nil)
			right := expr.DupAtt(left)
			tc.change(right)
			sd := &ServiceData{Name: "Contract", Scope: codegen.NewNameScope()}
			wrapNestedCollection(left, "", sd)
			wrapNestedCollection(right, "", sd)
			require.NotEqual(t, left.Type.(expr.UserType).ID(), right.Type.(expr.UserType).ID())
		})
	}
}

func TestAnonymousCollectionCompletePredicateIdentity(t *testing.T) {
	bound, err := expr.NewNumericBound("2")
	require.NoError(t, err)
	for _, tc := range []struct {
		name  string
		kind  expr.DataType
		rules *expr.ValidationExpr
	}{
		{name: "minimum", kind: expr.Int, rules: &expr.ValidationExpr{Minimum: &bound}},
		{name: "exclusive minimum", kind: expr.Int, rules: &expr.ValidationExpr{ExclusiveMinimum: &bound}},
		{name: "maximum", kind: expr.Int, rules: &expr.ValidationExpr{Maximum: &bound}},
		{name: "exclusive maximum", kind: expr.Int, rules: &expr.ValidationExpr{ExclusiveMaximum: &bound}},
		{name: "multiple enum clauses", kind: expr.String, rules: &expr.ValidationExpr{EnumClauses: [][]any{{"one", "two"}, {"two", "three"}}}},
		{name: "multiple patterns", kind: expr.String, rules: &expr.ValidationExpr{PatternClauses: []string{"^one", "one$"}}},
		{name: "multiple formats", kind: expr.String, rules: &expr.ValidationExpr{FormatClauses: []expr.ValidationFormat{expr.FormatEmail, expr.FormatHostname}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			left := &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: tc.kind}}}
			right := expr.DupAtt(left)
			expr.AsArray(right.Type).ElemType.Validation = tc.rules
			copy := expr.DupAtt(right)
			sd := &ServiceData{Name: "Contract", Scope: codegen.NewNameScope()}
			wrapNestedCollection(left, "", sd)
			wrapNestedCollection(right, "", sd)
			wrapNestedCollection(copy, "", sd)
			require.NotEqual(t, left.Type.(expr.UserType).ID(), right.Type.(expr.UserType).ID())
			require.Equal(t, right.Type.(expr.UserType).ID(), copy.Type.(expr.UserType).ID())
		})
	}
}

func TestAnonymousCollectionNamedOccurrenceFacts(t *testing.T) {
	minimum := 2
	node := &expr.UserTypeExpr{TypeName: "Node", UID: "Node", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{{Name: "label", Attribute: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"rpc:tag": {"1"}}}}}}}
	for _, change := range []func(*expr.AttributeExpr){
		func(att *expr.AttributeExpr) {
			att.Nullable = true
		},
		func(att *expr.AttributeExpr) {
			att.Validation = &expr.ValidationExpr{Required: []string{"label"}}
		},
	} {
		left := &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: node}}, Validation: &expr.ValidationExpr{MinLength: &minimum}}
		right := expr.DupAtt(left)
		change(expr.AsArray(right.Type).ElemType)
		sd := &ServiceData{Name: "Contract", Scope: codegen.NewNameScope()}
		wrapNestedCollection(left, "", sd)
		wrapNestedCollection(right, "", sd)
		require.NotEqual(t, left.Type.(expr.UserType).ID(), right.Type.(expr.UserType).ID())
	}
}
