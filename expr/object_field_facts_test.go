package expr

import (
	"runtime"
	"testing"
	"weak"

	"github.com/stretchr/testify/require"
)

func TestObjectFieldFactsMappedParity(t *testing.T) {
	alias := &UserTypeExpr{TypeName: "DefaultText", AttributeExpr: &AttributeExpr{Type: String, DefaultValue: "inherited"}}
	object := &Object{
		{Name: "optional:wire:tail", Attribute: &AttributeExpr{Type: String}},
		{Name: "required", Attribute: &AttributeExpr{Type: String}},
		{Name: "default", Attribute: &AttributeExpr{Type: alias}},
		{Name: "bytes", Attribute: &AttributeExpr{Type: Bytes}},
		{Name: "any", Attribute: &AttributeExpr{Type: Any}},
	}
	parent := &AttributeExpr{Type: object, Validation: &ValidationExpr{Required: []string{"required", "default", "missing"}}}
	for _, test := range []struct {
		name   string
		parent *AttributeExpr
	}{
		{"direct", parent},
		{"named", &AttributeExpr{Type: &UserTypeExpr{TypeName: "Parent", AttributeExpr: parent}}},
		{"nil", nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			facts := NewObjectFields(test.parent)
			mapped := NewMappedAttributeExpr(test.parent)
			for _, name := range []string{"optional", "required", "default", "bytes", "any", "missing", "unknown"} {
				actual := facts.FieldFacts(name)
				require.True(t, actual.ParentPresent)
				require.Equal(t, mapped.IsRequired(name), actual.Required, name)
				require.Equal(t, mapped.HasDefaultValue(name), actual.HasDefault, name)
				require.Equal(t, mapped.IsRequiredNoDefault(name), actual.Required && !actual.HasDefault, name)
				require.Equal(t, mapped.IsPrimitivePointer(name, false), actual.PrimitivePointer(false), name)
				require.Equal(t, mapped.IsPrimitivePointer(name, true), actual.PrimitivePointer(true), name)
				if actual.Index >= 0 {
					require.Equal(t, mapped.ElemName(name), actual.ElementName)
					for _, pointer := range []bool{false, true} {
						for _, useDefault := range []bool{false, true} {
							field := AsObject(mapped.Type).Attribute(name)
							require.Equal(t, ServiceFieldLayout(mapped.AttributeExpr, name, field, pointer, useDefault), actual.ServiceLayout(field, pointer, useDefault))
						}
					}
				}
			}
		})
	}
}

func TestObjectFieldFactsInheritedAndTypedNilDefaults(t *testing.T) {
	var typedNil *string
	inherited := &AttributeExpr{Type: String, DefaultValue: "inherited"}
	child := &AttributeExpr{Type: String, constraintParents: []*AttributeExpr{inherited}}
	base := &UserTypeExpr{TypeName: "Base", AttributeExpr: &AttributeExpr{
		Type:       &Object{{Name: "value:wire", Attribute: child}},
		Validation: &ValidationExpr{Required: []string{"value"}},
	}}
	parent := &AttributeExpr{Type: &Object{
		{Name: "value:wire", Attribute: child},
		{Name: "typed", Attribute: &AttributeExpr{Type: String, DefaultValue: typedNil}},
	}, Bases: []DataType{base}}
	facts := NewObjectFields(parent)
	require.True(t, facts.FieldFacts("value").Required)
	require.True(t, facts.FieldFacts("value").HasDefault)
	require.True(t, facts.FieldFacts("typed").HasDefault)
	require.False(t, facts.FieldFacts("value").PrimitivePointer(true))
	require.Equal(t, "wire", facts.FieldFacts("value").ElementName)
}

func TestObjectFieldFactsDuplicateSelection(t *testing.T) {
	parent := &AttributeExpr{Type: &Object{
		{Name: "n:first", Attribute: &AttributeExpr{Type: String}},
		{Name: "other", Attribute: &AttributeExpr{Type: Boolean}},
		{Name: "n:second:tail", Attribute: &AttributeExpr{Type: Int}},
		{Name: "n", Attribute: &AttributeExpr{Type: String, DefaultValue: "winner"}},
	}, Validation: &ValidationExpr{Required: []string{"n:first", "missing"}}}
	facts := NewObjectFields(parent)
	var order []string
	for name := range facts.Fields() {
		order = append(order, name)
	}
	require.Equal(t, []string{"n", "other"}, order)
	require.Equal(t, ObjectFieldFacts{ParentPresent: true, Required: true, LayoutRequired: true, HasDefault: true, NativePrimitive: true, ElementName: "second:tail", OriginalKey: "n", Index: 3}, facts.FieldFacts("n"))
	require.Equal(t, ObjectFieldFacts{ParentPresent: true, Required: true, LayoutRequired: true, Index: -1}, facts.FieldFacts("missing"))
}

func TestObjectFieldFactsSnapshotAndMutableQueries(t *testing.T) {
	child := &AttributeExpr{Type: String, DefaultValue: "before"}
	parent := &AttributeExpr{Type: &Object{{Name: "name:wire", Attribute: child}}, Validation: &ValidationExpr{Required: []string{"name:wire"}}}
	facts := NewObjectFields(parent)
	mapped := NewMappedAttributeExpr(parent)
	child.DefaultValue = nil
	parent.Validation.Required = nil
	mapped.Map("changed", "name")
	mapped.Validation.Required = nil
	require.True(t, facts.FieldFacts("name").Required)
	require.True(t, facts.FieldFacts("name").HasDefault)
	require.Equal(t, "wire", facts.FieldFacts("name").ElementName)
	require.False(t, NewObjectFields(parent).FieldFacts("name").Required)
	require.False(t, NewObjectFields(parent).FieldFacts("name").HasDefault)
	require.Equal(t, "changed", mapped.FieldFacts("name").ElementName)
	require.False(t, mapped.FieldFacts("name").Required)
	require.Equal(t, -1, mapped.FieldFacts("unknown").Index)
}

func TestObjectFieldFactsRecursiveIsolation(t *testing.T) {
	facts, original := isolatedObjectFieldFacts()
	for range 10 {
		runtime.GC()
		if original.Value() == nil {
			break
		}
	}
	require.Nil(t, original.Value(), "scalar facts must not retain the original recursive graph")
	require.Equal(t, "next", facts.FieldFacts("next").OriginalKey)
	runtime.KeepAlive(facts)
}

func TestObjectFieldFactsQueryAllocations(t *testing.T) {
	facts := NewObjectFields(&AttributeExpr{Type: &Object{{Name: "value", Attribute: &AttributeExpr{Type: String}}}})
	var result ObjectFieldFacts
	require.Zero(t, testing.AllocsPerRun(100, func() {
		result = facts.FieldFacts("value")
	}))
	require.Equal(t, "value", result.OriginalKey)
	require.Panics(t, func() {
		NewObjectFields(&AttributeExpr{Type: String})
	})
}

func TestServiceFieldLayoutSkipsUnneededParentQueries(t *testing.T) {
	attribute := &AttributeExpr{Type: String, Nullable: true}
	layout := ServiceFieldLayout(nil, "missing", attribute, false, true)
	require.False(t, layout.Pointer)
	require.True(t, layout.JSON.Optional)
	require.Equal(t, layout, (ObjectFieldFacts{Index: -1}).ServiceLayout(attribute, false, true))
}

//go:noinline
func isolatedObjectFieldFacts() (*ObjectFields, weak.Pointer[AttributeExpr]) {
	named := &UserTypeExpr{TypeName: "Node"}
	parent := &AttributeExpr{Type: &Object{{Name: "next", Attribute: &AttributeExpr{Type: named}}}}
	named.AttributeExpr = parent
	return NewObjectFields(parent), weak.Make(parent)
}

func TestObjectFieldFactsRepeatedExactKeys(t *testing.T) {
	parent := &AttributeExpr{Type: &Object{
		{Name: "n:first", Attribute: &AttributeExpr{Type: String}},
		{Name: "n:second", Attribute: &AttributeExpr{Type: String, DefaultValue: "winner"}},
		{Name: "n:first", Attribute: &AttributeExpr{Type: String}},
	}}
	facts := NewObjectFields(parent)
	mapped := NewMappedAttributeExpr(parent)
	actual := facts.FieldFacts("n")
	require.Equal(t, mapped.HasDefaultValue("n"), actual.HasDefault)
	require.Equal(t, mapped.ElemName("n"), actual.ElementName)
	require.Equal(t, "n:second", actual.OriginalKey)
	require.Equal(t, 1, actual.Index)
	var order []string
	for name := range facts.Fields() {
		order = append(order, name)
	}
	require.Equal(t, []string{"n"}, order)
}

func TestObjectFieldFactsNamedOccurrenceStorage(t *testing.T) {
	for _, required := range []bool{false, true} {
		field := &AttributeExpr{Type: String}
		decl := &AttributeExpr{Type: &Object{{Name: "name", Attribute: field}}}
		if required {
			decl.Validation = &ValidationExpr{Required: []string{"name"}}
		}
		named := &UserTypeExpr{TypeName: "Value", AttributeExpr: decl}
		alias := &UserTypeExpr{TypeName: "Alias", AttributeExpr: &AttributeExpr{Type: named}}
		occurrence := &AttributeExpr{Type: alias, Validation: &ValidationExpr{Required: []string{"name"}}}
		facts := NewObjectFields(occurrence).FieldFacts("name")
		require.True(t, facts.Required)
		require.Equal(t, required, facts.LayoutRequired)
		require.Equal(t, !required, facts.PrimitivePointer(false))
		require.Equal(t, ServiceFieldLayout(decl, "name", field, false, true), facts.ServiceLayout(field, false, true))
		require.Equal(t, ServiceFieldLayout(decl, "name", field, false, true), ServiceFieldLayout(occurrence, "name", field, false, true))
	}
}
