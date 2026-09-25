package codegen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/testdata"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/expr"
)

func TestGoTransformUnion(t *testing.T) {
	root := RunDSL(t, testdata.TestUnionDSL)
	var (
		scope = NewNameScope()

		// types to test
		unionString     = root.UserType("Container").Attribute().Find("UnionString").Find("UnionString")
		unionString2    = root.UserType("Container").Attribute().Find("UnionString2").Find("UnionString2")
		unionStringInt  = root.UserType("Container").Attribute().Find("UnionStringInt").Find("UnionStringInt")
		unionStringInt2 = root.UserType("Container").Attribute().Find("UnionStringInt2").Find("UnionStringInt2")
		unionSomeType   = root.UserType("Container").Attribute().Find("UnionSomeType").Find("UnionSomeType")
		unionSomeType2  = root.UserType("Container").Attribute().Find("UnionSomeType2").Find("UnionSomeType2")
		defaultCtx      = NewAttributeContext(false, false, true, "", scope)
	)
	tc := []struct {
		Name   string
		Source *expr.AttributeExpr
		Target *expr.AttributeExpr
	}{
		{"UnionString to UnionString2", unionString, unionString2},
		{"UnionStringInt to UnionStringInt2", unionStringInt, unionStringInt2},
		{"UnionSomeType to UnionSomeType2", unionSomeType, unionSomeType2},
	}
	for _, c := range tc {
		t.Run(c.Name, func(t *testing.T) {
			code, _, err := GoTransform(c.Source, c.Target, "source", "target", defaultCtx, defaultCtx, "", true)
			require.NoError(t, err)
			code = FormatTestCode(t, "package foo\nfunc transform(){\n"+code+"}")
			testutil.AssertGo(t, "testdata/golden/go_transform_union_"+c.Name+".go.golden", code)
		})
	}
}

func TestGoTransformUnionError(t *testing.T) {
	root := RunDSL(t, testdata.TestUnionDSL)
	var (
		scope = NewNameScope()

		// types to test
		unionString    = root.UserType("Container").Attribute().Find("UnionString").Find("UnionString")
		unionStringInt = root.UserType("Container").Attribute().Find("UnionStringInt").Find("UnionStringInt")
		unionSomeType  = root.UserType("Container").Attribute().Find("UnionSomeType").Find("UnionSomeType")
		defaultCtx     = NewAttributeContext(false, false, true, "", scope)
	)
	tc := []struct {
		Name   string
		Source *expr.AttributeExpr
		Target *expr.AttributeExpr
		Error  string
	}{
		{"UnionString to UnionStringInt", unionString, unionStringInt, "cannot transform union: number of union types differ (UnionString has 1, UnionStringInt has 2)"},
		{"UnionString to UnionSomeType", unionString, unionSomeType, "cannot transform union UnionString to UnionSomeType: type at index 0: source is a string but target type is object"},
	}
	for _, c := range tc {
		t.Run(c.Name, func(t *testing.T) {
			_, _, err := GoTransform(c.Source, c.Target, "source", "target", defaultCtx, defaultCtx, "", true)
			if err == nil {
				t.Errorf("unexpected success")
				return
			}
			if err.Error() != c.Error {
				t.Errorf("unexpected error, got: %s, expected: %s", err, c.Error)
			}
		})
	}
}

func TestGoTransformUnionUsesExplicitVariantTagsInSwitchCases(t *testing.T) {
	scope := NewNameScope()
	ctx := NewAttributeContext(false, false, true, "", scope)

	source := &expr.AttributeExpr{
		Type: &expr.Union{
			TypeName: "SourceAction",
			Values: []*expr.NamedAttributeExpr{
				{
					Name: "Single",
					Attribute: &expr.AttributeExpr{
						Type: expr.String,
						Meta: expr.MetaExpr{"oneof:type:tag": []string{"single"}},
					},
				},
				{
					Name: "Batch",
					Attribute: &expr.AttributeExpr{
						Type: expr.String,
						Meta: expr.MetaExpr{"oneof:type:tag": []string{"batch"}},
					},
				},
			},
		},
	}
	target := &expr.AttributeExpr{
		Type: &expr.Union{
			TypeName: "TargetAction",
			Values: []*expr.NamedAttributeExpr{
				{
					Name: "Single",
					Attribute: &expr.AttributeExpr{
						Type: expr.String,
						Meta: expr.MetaExpr{"oneof:type:tag": []string{"single"}},
					},
				},
				{
					Name: "Batch",
					Attribute: &expr.AttributeExpr{
						Type: expr.String,
						Meta: expr.MetaExpr{"oneof:type:tag": []string{"batch"}},
					},
				},
			},
		},
	}

	code, _, err := GoTransform(source, target, "source", "target", ctx, ctx, "", true)
	require.NoError(t, err)
	formatted := FormatTestCode(t, "package foo\nfunc transform(){\n"+code+"}")

	require.Contains(t, formatted, `case "single":`)
	require.Contains(t, formatted, `case "batch":`)
	require.False(t, strings.Contains(formatted, `case "Single":`))
	require.False(t, strings.Contains(formatted, `case "Batch":`))
}

func TestGoTransformOptionalUnionFieldsPreserveAbsence(t *testing.T) {
	union := &expr.Union{
		TypeName: "Choice",
		Values: []*expr.NamedAttributeExpr{
			{Name: "Text", Attribute: &expr.AttributeExpr{Type: expr.String}},
		},
	}
	source := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "choice", Attribute: &expr.AttributeExpr{Type: union}},
	}}
	target := expr.DupAtt(source)
	scope := NewNameScope()
	ctx := NewAttributeContext(false, false, true, "", scope)

	code, _, err := GoTransform(source, target, "source", "target", ctx, ctx, "", true)
	require.NoError(t, err)
	require.Contains(t, code, "if source.Choice != nil && source.Choice.Kind() != \"\"")
	require.Contains(t, code, "target.Choice = &u")
}

func TestGoTransformUnionWithUnionBranchKeepsOuterTempVar(t *testing.T) {
	leaf := &expr.UserTypeExpr{TypeName: "Leaf", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{
		{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}},
	}}}
	choice := &expr.UserTypeExpr{TypeName: "Choice", AttributeExpr: &expr.AttributeExpr{Type: &expr.Union{
		TypeName: "Choice",
		Values: []*expr.NamedAttributeExpr{
			{Name: "Leaf", Attribute: &expr.AttributeExpr{Type: leaf}},
			{Name: "Text", Attribute: &expr.AttributeExpr{Type: expr.String}},
		},
	}}}
	outer := &expr.AttributeExpr{Type: &expr.Union{
		TypeName: "ChoiceOrText",
		Values: []*expr.NamedAttributeExpr{
			{Name: "Choice", Attribute: &expr.AttributeExpr{Type: choice}},
			{Name: "Text", Attribute: &expr.AttributeExpr{Type: expr.String}},
		},
	}}
	ctx := NewAttributeContext(false, false, true, "", NewNameScope())

	code, _, err := GoTransform(outer, expr.DupAtt(outer), "source", "target", ctx, ctx, "", true)
	require.NoError(t, err)
	formatted := FormatTestCode(t, "package foo\nfunc transform(){\n"+code+"}")
	testutil.AssertGo(t, "testdata/golden/go_transform_union_union-branch.go.golden", formatted)
	// The inner union declares obj and converts its branches into tmp, so
	// its cases assign the outer obj and not a variable that shadows it.
	require.Contains(t, formatted, "var obj *Choice\n")
	require.Contains(t, formatted, "tmp := actual\n")
	require.Contains(t, formatted, "u.SetText((string)(tmp))\n\t\t\tobj = &u\n")
	require.Contains(t, formatted, "u.SetChoice((*Choice)(obj))\n")
}

func TestTransformUnionTempVarName(t *testing.T) {
	cases := []struct {
		TargetVar string
		Expected  string
	}{
		{"target", "obj"},
		{"body.P", "obj"},
		{"tv", "obj"},
		{"tmp", "obj"},
		{"tmp.Field", "obj"},
		{"objects", "obj"},
		{"object.Field", "obj"},
		{"obj", "tmp"},
		{"obj.Field", "tmp"},
		{"obj[i]", "tmp"},
		{"obj.Field[i]", "tmp"},
	}
	for _, c := range cases {
		t.Run(c.TargetVar, func(t *testing.T) {
			require.Equal(t, c.Expected, transformUnionTempVarName(c.TargetVar))
		})
	}
}
