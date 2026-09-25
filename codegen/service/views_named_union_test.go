package service

import (
	"bytes"
	"go/format"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestStreamingNamedUnionPayload checks that a named union used as the
// streaming payload of one method and as a streaming result and a field of
// another is declared once, as the union type itself.
func TestStreamingNamedUnionPayload(t *testing.T) {
	root := codegen.RunDSL(t, namedUnionStreamingDSL)
	services := NewServicesData(root)
	talk := services.Get("svc").Method("talk")
	assert.Equal(t, "Choice", talk.StreamingPayload)
	assert.Empty(t, talk.StreamingPayloadDef)

	code := renderServiceFile(t, root, services)
	assert.Equal(t, 1, strings.Count(code, "\ntype Choice struct {"), code)
	assert.NotContains(t, code, "type Choice Choice")
	assert.NotContains(t, code, "Choice2")
}

// TestViewsNamedUnion checks that a named union held by a result type with
// views is declared in the views package as a union named after its
// projected type, with the projected branch types.
func TestViewsNamedUnion(t *testing.T) {
	root := codegen.RunDSL(t, namedUnionViewsDSL)
	services := NewServicesData(root)
	unions := collectViewsUnions(services.Get("svc"))
	require.Len(t, unions, 1)
	assert.Equal(t, "ChoiceView", unions[0].Name)
	fields := make([]string, 0, len(unions[0].Fields))
	for _, field := range unions[0].Fields {
		fields = append(fields, field.FieldType)
	}
	assert.Equal(t, []string{"*LeafView", "*OtherView"}, fields)

	code := renderViewsFile(t, root, services)
	assert.Contains(t, code, "\ntype ChoiceView struct {")
	assert.Contains(t, code, "C  *ChoiceView")
	assert.NotContains(t, code, "type ChoiceView Choice")
}

// TestViewedResultNestedHelperNames checks that the conversions of a
// collection of result types with views and of a result type that holds
// another one call the conversions generated for the element and the nested
// result type, in each view.
func TestViewedResultNestedHelperNames(t *testing.T) {
	root := codegen.RunDSL(t, namedUnionViewsDSL)
	code := renderServiceFile(t, root, NewServicesData(root))
	cases := []struct {
		function string
		call     string
	}{
		{"NewRTCollectionFromRTCollectionView", "res[i] = NewRTFromRTView(n)"},
		{"NewRTCollectionFromRTCollectionViewTiny", "res[i] = NewRTFromRTViewTiny(n)"},
		{"ProjectRTCollection", "vres[i] = ProjectRT(n)"},
		{"ProjectRTCollectionTiny", "vres[i] = ProjectRTTiny(n)"},
		{"NewParentFromParentView", "res.Child = NewRTFromRTViewTiny(vres.Child)"},
		{"ProjectParent", "vres.Child = ProjectRTTiny(res.Child)"},
	}
	for _, c := range cases {
		assert.Contains(t, generatedFunction(t, code, c.function), c.call, c.function)
	}
	assert.NotRegexp(t, `\bnew[A-Z]\w*\(`, code)
}

func namedUnionStreamingDSL() {
	leaf, other := namedUnionBranchTypes()
	choice := dsl.Type("Choice", dsl.OneOf(leaf, other))
	holder := dsl.Type("Holder", func() {
		dsl.Attribute("choice", choice)
	})
	dsl.Service("svc", func() {
		dsl.Method("talk", func() {
			dsl.StreamingPayload(choice)
			dsl.StreamingResult(holder)
		})
		dsl.Method("watch", func() {
			dsl.StreamingResult(choice)
		})
	})
}

func namedUnionViewsDSL() {
	leaf, other := namedUnionBranchTypes()
	choice := dsl.Type("Choice", dsl.OneOf(leaf, other))
	rt := dsl.ResultType("application/vnd.rt", "RT", func() {
		dsl.Attribute("id", dsl.String)
		dsl.Attribute("c", choice)
		dsl.Required("id")
		dsl.View("default", func() {
			dsl.Attribute("id")
			dsl.Attribute("c")
		})
		dsl.View("tiny", func() {
			dsl.Attribute("id")
		})
	})
	parent := dsl.ResultType("application/vnd.parent", "Parent", func() {
		dsl.Attribute("name", dsl.String)
		dsl.Attribute("child", rt, func() {
			dsl.View("tiny")
		})
		dsl.View("default", func() {
			dsl.Attribute("name")
			dsl.Attribute("child")
		})
	})
	dsl.Service("svc", func() {
		dsl.Method("list", func() {
			dsl.Result(dsl.CollectionOf(rt))
		})
		dsl.Method("parent", func() {
			dsl.Result(parent)
		})
	})
}

func namedUnionBranchTypes() (expr.UserType, expr.UserType) {
	leaf := dsl.Type("Leaf", func() {
		dsl.Attribute("name", dsl.String)
		dsl.Required("name")
	})
	other := dsl.Type("Other", func() {
		dsl.Attribute("count", dsl.Int)
	})
	return leaf, other
}

func renderServiceFile(t *testing.T, root *expr.RootExpr, services *ServicesData) string {
	t.Helper()
	files := Files("github.com/CaliLuke/loom/example", root.Services[0], services, make(map[string][]string))
	require.NotEmpty(t, files)
	return renderFileSections(t, files[0])
}

func renderViewsFile(t *testing.T, root *expr.RootExpr, services *ServicesData) string {
	t.Helper()
	file := ViewsFile("github.com/CaliLuke/loom/example", root.Services[0], services)
	require.NotNil(t, file)
	return renderFileSections(t, file)
}

func renderFileSections(t *testing.T, file *codegen.File) string {
	t.Helper()
	var buf bytes.Buffer
	for _, section := range file.AllSections()[1:] {
		require.NoError(t, section.Write(&buf))
	}
	code, err := format.Source(buf.Bytes())
	require.NoError(t, err, buf.String())
	return strings.ReplaceAll(string(code), "\r\n", "\n")
}
