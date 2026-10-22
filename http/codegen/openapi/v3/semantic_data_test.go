package openapiv3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestSharedSemanticsPreserveOpenAPI(t *testing.T) {
	for _, test := range []struct {
		name   string
		design func()
	}{
		{"parameters without body", semanticParameterDSL},
		{"selected defaults", testdata.SelectedBodyDefaultsDSL},
		{"recursive", testdata.PayloadBodyInlineRecursiveUserDSL},
		{"views", testdata.MultipleViewsDSL},
		{"streaming", testdata.AsyncSessionSecurityDSL},
		{"error projection", testdata.ErrorBodyProjectionDSL},
	} {
		for _, suppress := range []bool{false, true} {
			name := test.name
			if suppress {
				name += "/suppressed"
			}
			t.Run(name, func(t *testing.T) {
				root := httpgen.RunHTTPDSL(t, test.design)
				if suppress {
					if root.API.Meta == nil {
						root.API.Meta = make(expr.MetaExpr)
					}
					root.API.Meta["openapi:example"] = []string{"false"}
				}
				// Match the normal ordering: Go analysis normalizes raw method objects
				// before standalone OpenAPI captures them.
				standalone := service.NewServicesData(root)
				for _, svc := range root.Services {
					standalone.Get(svc.Name)
				}
				want, err := openapiv3.Files(root)
				require.NoError(t, err)

				root = httpgen.RunHTTPDSL(t, test.design)
				if suppress {
					if root.API.Meta == nil {
						root.API.Meta = make(expr.MetaExpr)
					}
					root.API.Meta["openapi:example"] = []string{"false"}
				}
				owner := service.NewSemanticData(root)
				backend := owner.Services()
				for _, svc := range root.Services {
					backend.Get(svc.Name)
				}
				got, err := openapiv3.Files(root, openapiv3.WithServiceSemantics(owner))
				require.NoError(t, err)
				require.Len(t, got, len(want))
				for i, file := range got {
					require.Equal(t, want[i].Path, file.Path)
					expected := renderSection(t, want[i].AllSections()[0])
					actual := renderSection(t, file.AllSections()[0])
					require.Equal(t, expected.String(), actual.String())
				}
			})
		}
	}
}

func semanticParameterDSL() {
	dsl.Service("parameters", func() {
		dsl.Method("show", func() {
			dsl.Payload(func() {
				dsl.Attribute("id", dsl.Int64, func() {
					dsl.Minimum(1)
				})
				dsl.Required("id")
			})
			dsl.HTTP(func() {
				dsl.GET("/{id}")
				dsl.Param("id")
			})
		})
	})
}

func TestSharedSemanticsPreserveCustomSamplingSequence(t *testing.T) {
	random := &documentRandomizer{Randomizer: expr.NewDeterministicRandomizer()}
	root := httpgen.RunHTTPDSL(t, semanticParameterDSL)
	root.API.ExampleGenerator = &expr.ExampleGenerator{Randomizer: random}
	for range 2 {
		service.NewServicesData(root).Get("parameters")
	}
	want, err := openapiv3.Files(root)
	require.NoError(t, err)
	draws := random.draws

	random.draws = 0
	root = httpgen.RunHTTPDSL(t, semanticParameterDSL)
	root.API.ExampleGenerator = &expr.ExampleGenerator{Randomizer: random}
	owner := service.NewSemanticData(root)
	for range 2 {
		owner.Services().Get("parameters")
	}
	got, err := openapiv3.Files(root, openapiv3.WithServiceSemantics(owner))
	require.NoError(t, err)
	require.Equal(t, draws, random.draws)
	require.Len(t, got, len(want))
	for i, file := range got {
		expected := renderSection(t, want[i].AllSections()[0])
		actual := renderSection(t, file.AllSections()[0])
		require.Equal(t, expected.String(), actual.String())
	}
}

type documentRandomizer struct {
	expr.Randomizer
	draws int64
}

func (r *documentRandomizer) Int64() int64 {
	r.draws++
	return r.draws
}
