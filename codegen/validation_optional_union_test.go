package codegen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/CaliLuke/loom/expr"
)

// TestValidationCodeChecksOptionalUnionsForNil checks that the validation
// code of an optional union field, which a sum type scope declares as a
// pointer, runs only when the field is not nil, in service and view contexts,
// for named and inline unions and for a union nested in an inline object. A
// required union field holds a value and a protocol buffer oneof is an
// interface, so neither gets a nil check.
func TestValidationCodeChecksOptionalUnionsForNil(t *testing.T) {
	minLength := 2
	code := &expr.UserTypeExpr{
		TypeName:      "Code",
		AttributeExpr: &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{MinLength: &minLength}},
	}
	branches := func() []*expr.NamedAttributeExpr {
		return []*expr.NamedAttributeExpr{
			{Name: "code", Attribute: &expr.AttributeExpr{Type: code}},
			{Name: "count", Attribute: &expr.AttributeExpr{Type: expr.Int}},
		}
	}
	choice := &expr.UserTypeExpr{
		TypeName:      "Choice",
		AttributeExpr: &expr.AttributeExpr{Type: &expr.Union{TypeName: "Choice", Values: branches()}},
	}
	nested := &expr.Object{
		{Name: "inner", Attribute: &expr.AttributeExpr{Type: choice}},
	}
	fields := &expr.Object{
		{Name: "named", Attribute: &expr.AttributeExpr{Type: choice}},
		{Name: "req", Attribute: &expr.AttributeExpr{Type: choice}},
		{Name: "inline", Attribute: &expr.AttributeExpr{Type: &expr.Union{TypeName: "Inline", Values: branches()}}},
		{Name: "nested", Attribute: &expr.AttributeExpr{Type: nested}},
	}
	message := &expr.UserTypeExpr{
		TypeName: "Message",
		AttributeExpr: &expr.AttributeExpr{
			Type:       fields,
			Validation: &expr.ValidationExpr{Required: []string{"req"}},
		},
	}
	sumType := []string{
		"if message.Named != nil {\n\tswitch string(message.Named.Kind()) {\n\tcase \"code\":",
		"if message.Inline != nil {\n\tswitch string(message.Inline.Kind()) {\n\tcase \"code\":",
		"if message.Nested != nil {\nif message.Nested.Inner != nil {\n\tswitch string(message.Nested.Inner.Kind()) {",
		"if message.Req.Kind() == \"\" {",
		"}\nswitch string(message.Req.Kind()) {",
	}
	cases := []struct {
		name     string
		scope    Attributor
		pointer  bool
		view     bool
		contains []string
		excludes []string
	}{
		{"service", NewAttributeScope(NewNameScope()), false, false, sumType, []string{"message.Req != nil"}},
		{"view", NewAttributeScope(NewNameScope()), true, true, sumType, []string{"message.Req != nil"}},
		{"message scope", &messageScope{scope: NewNameScope(), names: map[*expr.Object]map[string][]string{
			fields: {
				"named":  {"NamedOneof", "Code", "Count"},
				"req":    {"ReqOneof", "ReqCode", "ReqCount"},
				"inline": {"InlineOneof", "InlineCode", "InlineCount"},
			},
			nested: {"inner": {"InnerOneof", "InnerCode", "InnerCount"}},
		}}, false, false, []string{
			"switch v := message.NamedOneof.(type) {",
			"switch v := message.InlineOneof.(type) {",
			"switch v := message.Nested.InnerOneof.(type) {",
		}, []string{"message.NamedOneof != nil", "message.InlineOneof != nil", "message.Nested.InnerOneof != nil"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := NewAttributeContext(c.pointer, false, !c.pointer, "", NewNameScope())
			ctx.Scope = c.scope
			code := ValidationCode(message.Attribute(), message, ctx, true, false, c.view, "message")
			for _, want := range c.contains {
				assert.Contains(t, code, want)
			}
			for _, unwanted := range c.excludes {
				assert.NotContains(t, code, unwanted)
			}
		})
	}
}

func TestValidationCodeGuardsNestedSumUnionBranches(t *testing.T) {
	minLength := 1
	leaf := &expr.UserTypeExpr{
		TypeName: "Leaf",
		AttributeExpr: &expr.AttributeExpr{
			Type: &expr.Object{{
				Name: "name",
				Attribute: &expr.AttributeExpr{
					Type:       expr.String,
					Validation: &expr.ValidationExpr{MinLength: &minLength},
				},
			}},
		},
	}
	other := &expr.UserTypeExpr{
		TypeName:      "Other",
		AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}},
	}
	choice := &expr.UserTypeExpr{
		TypeName: "Choice",
		AttributeExpr: &expr.AttributeExpr{Type: &expr.Union{
			TypeName: "Choice",
			Values: []*expr.NamedAttributeExpr{
				{Name: "leaf", Attribute: &expr.AttributeExpr{Type: leaf}},
				{Name: "other", Attribute: &expr.AttributeExpr{Type: other}},
			},
		}},
	}
	tags := &expr.UserTypeExpr{
		TypeName: "Tags",
		AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{{
			Name:      "values",
			Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}},
		}}},
	}
	holder := &expr.UserTypeExpr{
		TypeName: "Holder",
		AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{{
			Name: "p",
			Attribute: &expr.AttributeExpr{Type: &expr.Union{
				TypeName: "ChoiceOrTags",
				Values: []*expr.NamedAttributeExpr{
					{Name: "choice", Attribute: &expr.AttributeExpr{Type: choice}},
					{Name: "tags", Attribute: &expr.AttributeExpr{Type: tags}},
				},
			}},
		}}},
	}
	for _, c := range []struct {
		name        string
		pointer     bool
		useDefaults bool
	}{
		{name: "pointer context", pointer: true},
		{name: "value context", useDefaults: true},
	} {
		t.Run(c.name, func(t *testing.T) {
			ctx := NewAttributeContext(c.pointer, false, c.useDefaults, "", NewNameScope())
			ctx.Scope = NewAttributeScope(NewNameScope())
			code := ValidationCode(holder.Attribute(), holder, ctx, true, false, false, "holder")

			choiceValue := strings.Index(code, "actual, _ := holder.P.AsChoice()")
			guard := strings.Index(code, "if actual == nil {")
			nestedSwitch := strings.Index(code, "switch string(actual.Kind()) {")
			if choiceValue < 0 || guard < choiceValue || nestedSwitch < 0 || guard > nestedSwitch {
				t.Fatalf("nested sum union validator must guard its nil branch before Kind():\n%s", code)
			}
		})
	}
}
