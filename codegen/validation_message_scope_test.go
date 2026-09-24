package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/CaliLuke/loom/expr"
)

type (
	// messageScope is an attribute scope that generates unions as oneof
	// interfaces and names the union fields of an object and their branches
	// with the names that it records per field and object.
	messageScope struct {
		scope *NameScope
		names map[*expr.Object]map[string][]string
	}
)

// Name returns the name of the type of att.
func (s *messageScope) Name(att *expr.AttributeExpr, pkg string, ptr, useDefault bool) string {
	return s.scope.GoFullTypeName(att, pkg)
}

// Ref returns the reference to the type of att.
func (s *messageScope) Ref(att *expr.AttributeExpr, pkg string) string {
	return s.scope.GoFullTypeRef(att, pkg)
}

// Field returns the Goified name.
func (*messageScope) Field(_ *expr.AttributeExpr, name string, firstUpper bool) string {
	return Goify(name, firstUpper)
}

// Scope returns the name scope.
func (s *messageScope) Scope() *NameScope {
	return s.scope
}

// UnionFieldNames returns the names recorded for the union field name of the
// object attribute obj: the name of the field followed by the names of the
// branches.
func (s *messageScope) UnionFieldNames(obj *expr.AttributeExpr, name string) (string, []string) {
	names := s.names[expr.AsObject(obj.Type)][name]
	return names[0], names[1:]
}

// TestValidationCodeUsesMessageFieldScopeNames checks that the validation
// code of an object refers to a union field and to the fields that hold its
// branches with the names that a messageFieldScope gives them, including in
// the required field check, that the names apply to the branches of that
// union only, and that two fields of the same named union get their own code.
func TestValidationCodeUsesMessageFieldScopeNames(t *testing.T) {
	minLength := 2
	code := &expr.UserTypeExpr{
		TypeName:      "Code",
		AttributeExpr: &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{MinLength: &minLength}},
	}
	choice := &expr.UserTypeExpr{
		TypeName: "Choice",
		AttributeExpr: &expr.AttributeExpr{Type: &expr.Union{
			TypeName: "Choice",
			Values: []*expr.NamedAttributeExpr{
				{Name: "code", Attribute: &expr.AttributeExpr{Type: code}},
				{Name: "count", Attribute: &expr.AttributeExpr{Type: expr.Int}},
			},
		}},
	}
	fields := &expr.Object{
		{Name: "first", Attribute: &expr.AttributeExpr{Type: choice}},
		{Name: "second", Attribute: &expr.AttributeExpr{Type: choice}},
	}
	message := &expr.UserTypeExpr{
		TypeName: "Message",
		AttributeExpr: &expr.AttributeExpr{
			Type:       fields,
			Validation: &expr.ValidationExpr{Required: []string{"second"}},
		},
	}
	cases := []struct {
		name     string
		scope    Attributor
		contains []string
	}{
		{"message scope", &messageScope{scope: NewNameScope(), names: map[*expr.Object]map[string][]string{
			fields: {
				"first":  {"FirstOneof", "Code", "Count"},
				"second": {"SecondOneof", "SecondCode", "SecondCount"},
			},
		}}, []string{
			"if message.SecondOneof == nil {",
			"switch v := message.FirstOneof.(type) {\ncase *Message_Code:\n\tif utf8.RuneCountInString(string(v.Code)) < 2 {",
			"switch v := message.SecondOneof.(type) {\ncase *Message_SecondCode:\n\tif utf8.RuneCountInString(string(v.SecondCode)) < 2 {",
		}},
		{"sum type scope", NewAttributeScope(NewNameScope()), []string{
			"if message.Second.Kind() == \"\" {",
			"switch string(message.First.Kind()) {",
			"switch string(message.Second.Kind()) {",
			"InvalidLengthError(\"message.second.value\"",
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := NewAttributeContext(false, false, true, "", NewNameScope())
			ctx.Scope = c.scope
			code := ValidationCode(message.Attribute(), message, ctx, true, false, false, "message")
			for _, want := range c.contains {
				assert.Contains(t, code, want)
			}
		})
	}
}
