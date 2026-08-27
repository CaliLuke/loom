package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func TestGoTypeDef(t *testing.T) {
	// types to test
	var (
		mixed = &expr.AttributeExpr{
			Type: &expr.Object{
				&expr.NamedAttributeExpr{
					Name:      "required",
					Attribute: &expr.AttributeExpr{Type: expr.String},
				},
				&expr.NamedAttributeExpr{
					Name:      "default",
					Attribute: &expr.AttributeExpr{Type: expr.Int, DefaultValue: 0},
				},
				&expr.NamedAttributeExpr{
					Name:      "optional",
					Attribute: &expr.AttributeExpr{Type: expr.Float32},
				},
				&expr.NamedAttributeExpr{
					Name:      "bytes",
					Attribute: &expr.AttributeExpr{Type: expr.Bytes},
				},
				&expr.NamedAttributeExpr{
					Name:      "any",
					Attribute: &expr.AttributeExpr{Type: expr.Any},
				},
				&expr.NamedAttributeExpr{
					Name:      "required_bytes",
					Attribute: &expr.AttributeExpr{Type: expr.Bytes},
				},
				&expr.NamedAttributeExpr{
					Name:      "required_any",
					Attribute: &expr.AttributeExpr{Type: expr.Any},
				},
				&expr.NamedAttributeExpr{
					Name:      "default_bytes",
					Attribute: &expr.AttributeExpr{Type: expr.Bytes, DefaultValue: []byte("foo")},
				},
				&expr.NamedAttributeExpr{
					Name:      "default_any",
					Attribute: &expr.AttributeExpr{Type: expr.Any, DefaultValue: "foo"},
				},
				&expr.NamedAttributeExpr{
					Name:      "custom_type",
					Attribute: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:field:type": []string{"pkg.String"}}},
				},
				&expr.NamedAttributeExpr{
					Name:      "custom_tag",
					Attribute: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:tag:foo": []string{"bar"}}},
				},
			},
			Validation: &expr.ValidationExpr{
				Required: []string{"required", "required_bytes", "required_any"},
			},
		}
		union = &expr.Union{
			TypeName: "Choice",
			Values: []*expr.NamedAttributeExpr{
				{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}},
			},
		}
		unions = &expr.AttributeExpr{
			Type: &expr.Object{
				{Name: "optional", Attribute: &expr.AttributeExpr{Type: union}},
				{Name: "required", Attribute: &expr.AttributeExpr{Type: union}},
			},
			Validation: &expr.ValidationExpr{Required: []string{"required"}},
		}
	)

	cases := []struct {
		Name       string
		Attr       *expr.AttributeExpr
		UsePtr     bool
		UseDefault bool
		Def        string
	}{
		{"no-default", mixed, false, false, mixedNoDefault},
		{"use-default", mixed, false, true, mixedUseDefault},
		{"use-pointer", mixed, true, true, mixedUsePointer},
		{"union-presence", unions, false, true, unionPresence},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			def := goTypeDef(codegen.NewNameScope(), c.Attr, c.UsePtr, c.UseDefault, false)
			assert.Equal(t, c.Def, def)
		})
	}
}

func TestGoTypeDefUsesExplicitTypeForStructuredFields(t *testing.T) {
	scope := codegen.NewNameScope()
	field := &expr.AttributeExpr{
		Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}},
		Meta: expr.MetaExpr{
			"openapi:nullable":  []string{"true"},
			"struct:field:type": []string{"loom.Nullable[[]string]"},
		},
	}
	parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "values", Attribute: field}}}

	got := goTypeDef(scope, parent, true, false, false)
	require.Contains(t, got, "Values loom.Nullable[[]string]")
	require.NotContains(t, got, "*loom.Nullable")
}

func TestGoTypeDefUsesJSONPresenceWrappers(t *testing.T) {
	attribute := &expr.AttributeExpr{
		Type: &expr.Object{
			{Name: "required", Attribute: &expr.AttributeExpr{Type: expr.String}},
			{Name: "optional", Attribute: &expr.AttributeExpr{Type: expr.String}},
			{Name: "empty", Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}}},
			{Name: "nullable", Attribute: &expr.AttributeExpr{Type: expr.Int, Nullable: true}},
			{Name: "anything", Attribute: &expr.AttributeExpr{Type: expr.Any}},
			{Name: "renamed", Attribute: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{
				"struct:tag:json": {"wire_name,omitempty,string"},
				"struct:tag:xml":  {"legacy"},
			}}},
		},
		Validation: &expr.ValidationExpr{Required: []string{"required"}},
	}

	got := goTypeDef(codegen.NewNameScope(), attribute, true, false, true)
	require.Contains(t, got, "Required *string")
	require.Contains(t, got, "Optional loom.Optional[string]")
	require.Contains(t, got, "Empty loom.Optional[[]loom.Nullable[string]]")
	require.Contains(t, got, "Nullable loom.Nullable[int]")
	require.Contains(t, got, "Anything loom.Nullable[any]")
	require.Contains(t, got, `json:"optional,omitzero"`)
	require.Contains(t, got, `json:"empty,omitzero"`)
	require.Contains(t, got, `json:"nullable,omitzero"`)
	require.Contains(t, got, `json:"wire_name,omitzero,string"`)
	require.Contains(t, got, `xml:"legacy"`)
}

func TestGoTypeDefUsesJSONPresenceForArrayElements(t *testing.T) {
	nonNullable := &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}}
	nullable := &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String, Nullable: true}}}
	requiredAny := &expr.AttributeExpr{Type: &expr.Array{
		ElemType:         &expr.AttributeExpr{Type: expr.Any},
		NonNullableElems: true,
	}}
	scope := codegen.NewNameScope()

	require.Equal(t, "[]string", goTypeDef(scope, nonNullable, true, false, false))
	require.Equal(t, "[]loom.Nullable[string]", goTypeDef(scope, nonNullable, true, false, true))
	require.Equal(t, "[]loom.Nullable[string]", goTypeDef(scope, nullable, true, false, true))
	require.Equal(t, "[]loom.Nullable[any]", goTypeDef(scope, requiredAny, true, false, true))
}

func TestGoValueTypeDefStripsNamedRootPresence(t *testing.T) {
	root := &expr.AttributeExpr{Type: expr.String, Nullable: true}
	named := &expr.UserTypeExpr{TypeName: "NullableText", AttributeExpr: root}
	attribute := &expr.AttributeExpr{Type: named, Nullable: true}
	scope := codegen.NewNameScope()

	require.Equal(t, "NullableText", goValueTypeDef(scope, attribute, false, true, true))
	require.Equal(t, "loom.Nullable[NullableText]", goTypeDef(scope, attribute, false, true, true))
}

var (
	unionPresence = `struct {
	Optional *Choice ` + "`" + `form:"optional,omitempty" json:"optional,omitempty" xml:"optional,omitempty"` + "`" + `
	Required Choice ` + "`" + `form:"required" json:"required" xml:"required"` + "`" + `
}`

	mixedNoDefault = `struct {
	Required string ` + "`" + `form:"required" json:"required" xml:"required"` + "`" + `
	Default *int ` + "`" + `form:"default,omitempty" json:"default,omitempty" xml:"default,omitempty"` + "`" + `
	Optional *float32 ` + "`" + `form:"optional,omitempty" json:"optional,omitempty" xml:"optional,omitempty"` + "`" + `
	Bytes []byte ` + "`" + `form:"bytes,omitempty" json:"bytes,omitempty" xml:"bytes,omitempty"` + "`" + `
	Any loom.Nullable[any] ` + "`" + `form:"any,omitempty" json:"any,omitzero" xml:"any,omitempty"` + "`" + `
	RequiredBytes []byte ` + "`" + `form:"required_bytes" json:"required_bytes" xml:"required_bytes"` + "`" + `
	RequiredAny loom.Nullable[any] ` + "`" + `form:"required_any" json:"required_any" xml:"required_any"` + "`" + `
	DefaultBytes []byte ` + "`" + `form:"default_bytes,omitempty" json:"default_bytes,omitempty" xml:"default_bytes,omitempty"` + "`" + `
	DefaultAny loom.Nullable[any] ` + "`" + `form:"default_any,omitempty" json:"default_any,omitzero" xml:"default_any,omitempty"` + "`" + `
	CustomType *pkg.String ` + "`" + `form:"custom_type,omitempty" json:"custom_type,omitempty" xml:"custom_type,omitempty"` + "`" + `
	CustomTag *string ` + "`" + `foo:"bar"` + "`" + `
}`

	mixedUseDefault = `struct {
	Required string ` + "`" + `form:"required" json:"required" xml:"required"` + "`" + `
	Default int ` + "`" + `form:"default" json:"default" xml:"default"` + "`" + `
	Optional *float32 ` + "`" + `form:"optional,omitempty" json:"optional,omitempty" xml:"optional,omitempty"` + "`" + `
	Bytes []byte ` + "`" + `form:"bytes,omitempty" json:"bytes,omitempty" xml:"bytes,omitempty"` + "`" + `
	Any loom.Nullable[any] ` + "`" + `form:"any,omitempty" json:"any,omitzero" xml:"any,omitempty"` + "`" + `
	RequiredBytes []byte ` + "`" + `form:"required_bytes" json:"required_bytes" xml:"required_bytes"` + "`" + `
	RequiredAny loom.Nullable[any] ` + "`" + `form:"required_any" json:"required_any" xml:"required_any"` + "`" + `
	DefaultBytes []byte ` + "`" + `form:"default_bytes" json:"default_bytes" xml:"default_bytes"` + "`" + `
	DefaultAny loom.Nullable[any] ` + "`" + `form:"default_any" json:"default_any,omitzero" xml:"default_any"` + "`" + `
	CustomType *pkg.String ` + "`" + `form:"custom_type,omitempty" json:"custom_type,omitempty" xml:"custom_type,omitempty"` + "`" + `
	CustomTag *string ` + "`" + `foo:"bar"` + "`" + `
}`

	mixedUsePointer = `struct {
	Required *string ` + "`" + `form:"required,omitempty" json:"required,omitempty" xml:"required,omitempty"` + "`" + `
	Default *int ` + "`" + `form:"default,omitempty" json:"default,omitempty" xml:"default,omitempty"` + "`" + `
	Optional *float32 ` + "`" + `form:"optional,omitempty" json:"optional,omitempty" xml:"optional,omitempty"` + "`" + `
	Bytes []byte ` + "`" + `form:"bytes,omitempty" json:"bytes,omitempty" xml:"bytes,omitempty"` + "`" + `
	Any loom.Nullable[any] ` + "`" + `form:"any,omitempty" json:"any,omitzero" xml:"any,omitempty"` + "`" + `
	RequiredBytes []byte ` + "`" + `form:"required_bytes,omitempty" json:"required_bytes,omitempty" xml:"required_bytes,omitempty"` + "`" + `
	RequiredAny loom.Nullable[any] ` + "`" + `form:"required_any,omitempty" json:"required_any,omitempty" xml:"required_any,omitempty"` + "`" + `
	DefaultBytes []byte ` + "`" + `form:"default_bytes,omitempty" json:"default_bytes,omitempty" xml:"default_bytes,omitempty"` + "`" + `
	DefaultAny loom.Nullable[any] ` + "`" + `form:"default_any,omitempty" json:"default_any,omitzero" xml:"default_any,omitempty"` + "`" + `
	CustomType *pkg.String ` + "`" + `form:"custom_type,omitempty" json:"custom_type,omitempty" xml:"custom_type,omitempty"` + "`" + `
	CustomTag *string ` + "`" + `foo:"bar"` + "`" + `
}`
)
