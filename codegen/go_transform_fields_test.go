package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestTransformFieldsPreserveMappedSelections(t *testing.T) {
	defaultType := &expr.UserTypeExpr{TypeName: "DefaultText", AttributeExpr: &expr.AttributeExpr{Type: expr.String, DefaultValue: "inherited"}}
	parent := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "value:first", Attribute: &expr.AttributeExpr{Type: expr.String}},
		{Name: "other", Attribute: &expr.AttributeExpr{Type: expr.Boolean}},
		{Name: "value:second:tail", Attribute: &expr.AttributeExpr{Type: defaultType}},
		{Name: "value:first", Attribute: &expr.AttributeExpr{Type: expr.Int}},
		{Name: "bytes", Attribute: &expr.AttributeExpr{Type: expr.Bytes}},
		{Name: "any", Attribute: &expr.AttributeExpr{Type: expr.Any}},
		{Name: "nullable", Attribute: &expr.AttributeExpr{Type: expr.String, Nullable: true}},
	}, Validation: &expr.ValidationExpr{Required: []string{"value:first", "missing"}}}
	for _, test := range []struct {
		name      string
		attribute *expr.AttributeExpr
	}{
		{"direct", parent},
		{"named", &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Parent", AttributeExpr: parent}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			before := expr.DupAtt(test.attribute)
			fields := transformMappedAttribute(test.attribute)
			mapped := expr.NewMappedAttributeExpr(test.attribute)
			var names []string
			for name, facts := range fields.Fields() {
				names = append(names, name)
				child := test.attribute.PhysicalMemberAttribute(facts)
				require.Same(t, child, fields.Find(name))
				require.Equal(t, mapped.GetDefault(name), fields.GetDefault(name))
				require.Equal(t, mapped.IsRequired(name), fields.IsRequired(name))
				require.Equal(t, mapped.ElemName(name), facts.ElementName)
				for _, pointer := range []bool{false, true} {
					for _, defaults := range []bool{false, true} {
						context := &AttributeContext{Pointer: pointer, UseDefault: defaults, JSONPresence: true}
						require.Equal(t, context.FieldPresence(mapped, name, mapped.Find(name)), context.FieldPresence(fields, name, child))
						require.Equal(t, nativeObjectFieldPointer(mapped, name, mapped.Find(name), context), nativeObjectFieldPointer(fields, name, child, context))
					}
				}
			}
			require.Equal(t, []string{"value", "other", "bytes", "any", "nullable"}, names)
			require.Equal(t, "inherited", fields.GetDefault("value"))
			require.Nil(t, fields.Find("unknown"))
			require.Nil(t, fields.GetDefault("unknown"))
			require.True(t, fields.IsRequired("missing"))
			require.Equal(t, before, expr.DupAtt(test.attribute))
		})
	}
}

func TestTransformFieldsKeepDeclarationStorage(t *testing.T) {
	for _, required := range []bool{false, true} {
		field := &expr.AttributeExpr{Type: expr.String}
		declaration := &expr.AttributeExpr{Type: &expr.Object{{Name: "value:wire", Attribute: field}}}
		if required {
			declaration.Validation = &expr.ValidationExpr{Required: []string{"value:wire"}}
		}
		occurrence := &expr.AttributeExpr{
			Type:       &expr.UserTypeExpr{TypeName: "Parent", AttributeExpr: declaration},
			Validation: &expr.ValidationExpr{Required: []string{"value:wire"}},
		}
		fields := transformMappedAttribute(occurrence)
		require.True(t, fields.IsRequired("value"))
		require.Equal(t, required, fields.FieldFacts("value").LayoutRequired)
		require.Equal(t, !required, nativeObjectFieldPointer(fields, "value", field, &AttributeContext{}))
	}
}

func TestTransformFieldsRejectNonObjects(t *testing.T) {
	require.Panics(t, func() {
		transformMappedAttribute(&expr.AttributeExpr{Type: expr.String})
	})
}

func TestTransformFieldsWinningExactKeyDefault(t *testing.T) {
	for _, test := range []struct {
		name          string
		first, second any
	}{
		{"replace default", 1, 2},
		{"supply default", nil, 2},
		{"remove default", 1, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			selected := &expr.AttributeExpr{Type: &expr.UserTypeExpr{
				TypeName:      "Selected",
				AttributeExpr: &expr.AttributeExpr{Type: expr.Int, DefaultValue: test.second},
			}}
			parent := &expr.AttributeExpr{Type: &expr.Object{
				{Name: "value:wire", Attribute: &expr.AttributeExpr{Type: expr.Int, DefaultValue: test.first}},
				{Name: "value:wire", Attribute: selected},
			}}
			fields := transformMappedAttribute(parent)
			require.Same(t, selected, fields.Find("value"))
			require.Equal(t, test.second, fields.GetDefault("value"))
			require.Equal(t, test.second != nil, fields.FieldFacts("value").HasDefault)
		})
	}
}
