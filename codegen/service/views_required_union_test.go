package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
)

// TestViewsRequiredNamedUnion checks that the conversions between a result
// type with views and its projected type treat the field of a required
// named union as the value that the projected type declares, and still check
// the pointer of a required primitive for nil.
func TestViewsRequiredNamedUnion(t *testing.T) {
	root := codegen.RunDSL(t, requiredNamedUnionViewsDSL)
	services := NewServicesData(root)
	views := renderViewsFile(t, root, services)
	assert.Contains(t, views, "C  ChoiceView")
	assert.NotContains(t, views, "C  *ChoiceView")

	code := renderServiceFile(t, root, services)
	cases := []struct {
		function string
		contains []string
	}{
		{
			function: "NewRTFromRTView",
			contains: []string{"if vres.ID != nil {", `if vres.C.Kind() != "" {`, "actual, _ := vres.C.AsLeaf()", "u := res.C"},
		},
		{
			function: "ProjectRT",
			contains: []string{`if res.C.Kind() != "" {`, "u := vres.C", "vres.C = u"},
		},
	}
	for _, c := range cases {
		function := generatedFunction(t, code, c.function)
		for _, s := range c.contains {
			assert.Contains(t, function, s, c.function)
		}
		assert.NotContains(t, function, "vres.C != nil", c.function)
		assert.NotContains(t, function, "*vres.C", c.function)
		assert.NotContains(t, function, "&u", c.function)
	}
	for _, function := range []string{"NewRTFromRTViewTiny", "ProjectRTTiny"} {
		assert.NotContains(t, generatedFunction(t, code, function), ".C", function)
	}
}

func requiredNamedUnionViewsDSL() {
	leaf, other := namedUnionBranchTypes()
	choice := dsl.Type("Choice", dsl.OneOf(leaf, other))
	rt := dsl.ResultType("application/vnd.rt", "RT", func() {
		dsl.Attribute("id", dsl.String)
		dsl.Attribute("c", choice)
		dsl.Required("id", "c")
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
		dsl.Attribute("child", rt)
		dsl.View("default", func() {
			dsl.Attribute("name")
			dsl.Attribute("child")
		})
	})
	dsl.Service("svc", func() {
		dsl.Method("get", func() {
			dsl.Result(rt)
		})
		dsl.Method("list", func() {
			dsl.Result(dsl.CollectionOf(rt))
		})
		dsl.Method("parent", func() {
			dsl.Result(parent)
		})
	})
}
