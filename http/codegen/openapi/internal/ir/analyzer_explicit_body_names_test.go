package ir

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

// TestBuildBodyTypesNamesInlineResultBodySelections checks that an inline
// response Body that selects attributes of a result type gets a component of
// its own instead of claiming the component name of the result type. The Body
// DSL documents such a body as equivalent to the implicit body, so when the
// design has an implicit equivalent, both get the same component. A result
// type with views has none: a header must map an attribute of every view.
func TestBuildBodyTypesNamesInlineResultBodySelections(t *testing.T) {
	cases := []struct {
		name     string
		explicit func()
		implicit func()
		service  string
		method   string
		wantName string
	}{
		{
			name:     "result type",
			explicit: testdata.ExplicitBodyUserResultObjectDSL,
			implicit: implicitResultBodyDSL(),
			service:  "ServiceExplicitBodyUserResultObject",
			method:   "MethodExplicitBodyUserResultObject",
			wantName: "Resulttype_e30f7140cbd99e86",
		},
		{
			name:     "result type with views",
			explicit: testdata.ExplicitBodyUserResultObjectMultipleViewDSL,
			service:  "ServiceExplicitBodyUserResultObjectMultipleView",
			method:   "MethodExplicitBodyUserResultObjectMultipleView",
			wantName: "Resulttypemultipleviews_e30f7140cbd99e86",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			explicitRoot := codegen.RunDSL(t, tc.explicit)
			var explicit *BodyTypes
			require.NotPanics(t, func() {
				explicit = BuildBodyTypes(explicitRoot.API, explicitRoot.Types, explicitRoot.ResultTypes)
			})
			ref := explicit.Services[tc.service][tc.method].ResponseBodies[200][0].Ref
			assert.Equal(t, toRef(tc.wantName), ref)
			component := explicit.Components[tc.wantName]
			require.NotNil(t, component)
			require.Len(t, component.Properties, 1)
			assert.Equal(t, toRef("UserType"), component.Properties["a"].Ref)
			if tc.implicit == nil {
				return
			}

			implicitRoot := codegen.RunDSL(t, tc.implicit)
			implicit := BuildBodyTypes(implicitRoot.API, implicitRoot.Types, implicitRoot.ResultTypes)
			assert.Equal(t, ref, implicit.Services[tc.service][tc.method].ResponseBodies[200][0].Ref)
			assert.Equal(t, component, implicit.Components[tc.wantName])
		})
	}
}

// TestBuildBodyTypesKeepsInlineBodyNameAndReuse checks that an inline
// response Body keeps a name declared on the Body itself, and that an inline
// Body that selects every attribute of the result type still reuses its
// component.
func TestBuildBodyTypesKeepsInlineBodyNameAndReuse(t *testing.T) {
	cases := []struct {
		name     string
		body     func()
		wantName string
		want     []string
	}{
		{
			name: "declared name",
			body: func() {
				dsl.Body(func() {
					dsl.Meta("openapi:typename", "Selected")
					dsl.Attribute("a")
				})
			},
			wantName: "Selected",
			want:     []string{"a"},
		},
		{
			name: "every attribute",
			body: func() {
				dsl.Body(func() {
					dsl.Attribute("a")
					dsl.Attribute("b")
				})
			},
			wantName: "RT",
			want:     []string{"a", "b"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				rt := dsl.ResultType("application/vnd.rt", "RT", func() {
					dsl.Attribute("a", dsl.String)
					dsl.Attribute("b", dsl.String)
				})
				dsl.Service("Svc", func() {
					dsl.Method("do", func() {
						dsl.Result(rt)
						dsl.HTTP(func() {
							dsl.POST("/do")
							dsl.Response(dsl.StatusOK, tc.body)
						})
					})
				})
			})
			var bodies *BodyTypes
			require.NotPanics(t, func() {
				bodies = BuildBodyTypes(root.API, root.Types, root.ResultTypes)
			})
			assert.Equal(t, toRef(tc.wantName), bodies.Services["Svc"]["do"].ResponseBodies[200][0].Ref)
			component := bodies.Components[tc.wantName]
			require.NotNil(t, component)
			assert.Len(t, component.Properties, len(tc.want))
			for _, name := range tc.want {
				assert.Contains(t, component.Properties, name)
			}
		})
	}
}

// implicitResultBodyDSL returns ExplicitBodyUserResultObjectDSL with the
// same headers and no Body, which the Body DSL documents as equivalent.
func implicitResultBodyDSL() func() {
	return func() {
		userType := dsl.Type("UserType", func() {
			dsl.Attribute("x", dsl.String)
			dsl.Attribute("y", dsl.Int)
		})
		result := dsl.ResultType("ResultType", func() {
			dsl.Attribute("a", userType)
			dsl.Attribute("b", dsl.String)
			dsl.Attribute("c", dsl.String)
		})
		dsl.Service("ServiceExplicitBodyUserResultObject", func() {
			dsl.Method("MethodExplicitBodyUserResultObject", func() {
				dsl.Result(result)
				dsl.HTTP(func() {
					dsl.POST("/")
					dsl.Response(dsl.StatusOK, func() {
						dsl.Header("c:Location")
						dsl.Header("b:Content-Type")
					})
				})
			})
		})
	}
}
