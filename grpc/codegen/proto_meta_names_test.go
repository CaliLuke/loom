package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func TestProtoMetaMessageName(t *testing.T) {
	cases := []struct {
		Name     string
		Value    string
		Expected string
	}{
		{"ascii kept verbatim", "CustomType", "CustomType"},
		{"ascii underscore kept verbatim", "Custom_Type", "Custom_Type"},
		{"ascii lower kept verbatim", "custom", "custom"},
		{"latin accent separates words", "MenüProto", "MenProto"},
		{"leading non-ascii", "Écran", "Cran"},
		{"cjk then ascii", "日本Menu", "Menu"},
		{"non-ascii only", "日本", "Val"},
		{"non-ascii then digit", "é1", "Val1"},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			got := protoMetaMessageName(c.Value)
			assert.Equal(t, c.Expected, got)
			assert.Regexp(t, protoIdentifier, got)
		})
	}
}

func TestProtoStructNameUsesMetaMessageName(t *testing.T) {
	ut := &expr.UserTypeExpr{TypeName: "Menü", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}}}
	att := &expr.AttributeExpr{Type: ut, Meta: expr.MetaExpr{"struct:name:proto": {"MenüProto"}}}
	assert.Equal(t, "MenProto", protoStructName(att, ut))
	assert.Equal(t, "Menü", protoStructName(&expr.AttributeExpr{Type: ut}, ut))
}

func TestQualifyResponseContractMessage(t *testing.T) {
	cases := []struct {
		Name     string
		Message  string
		Expected string
	}{
		{"empty", "", ""},
		{"ascii", "ShowResponse", "svc.ShowResponse"},
		{"qualified", "google.protobuf.Empty", "google.protobuf.Empty"},
		{"non-ascii", "MenüProto", "svc.MenProto"},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			assert.Equal(t, c.Expected, qualifyResponseContractMessage("svc", c.Message))
		})
	}
}

// TestProtoMetaGoTypeName checks that the Go type name of a message named by
// struct:name:proto metadata follows the GoCamelCase conversion of
// protoc-gen-go, while the message keeps the metadata name.
func TestProtoMetaGoTypeName(t *testing.T) {
	cases := []struct {
		Name   string
		Value  string
		GoName string
	}{
		{"camel case", "MenuProto", "MenuProto"},
		{"snake case", "menu_proto", "MenuProto"},
		{"lower case", "custom", "Custom"},
		{"underscore before upper", "Menu_Proto", "Menu_Proto"},
		{"leading underscore", "_menu", "XMenu"},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ut := &expr.UserTypeExpr{TypeName: "Menu", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}}}
			att := &expr.AttributeExpr{Type: ut, Meta: expr.MetaExpr{"struct:name:proto": {c.Value}}}
			scope := codegen.NewNameScope()
			assert.Equal(t, c.Value, protoBufMessageName(att, scope))
			assert.Equal(t, c.GoName, protoBufGoTypeName(att, scope))
			assert.Equal(t, "pb."+c.GoName, protoBufGoFullTypeName(att, "pb", scope))
		})
	}
}
