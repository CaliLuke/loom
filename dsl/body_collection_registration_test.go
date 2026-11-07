package dsl_test

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestBodyCollectionDeclarationAdmission(t *testing.T) {
	for _, reused := range []bool{false, true} {
		name := "created_in_body"
		if reused {
			name = "registered_before_body"
		}
		t.Run(name, func(t *testing.T) {
			var collection *expr.ResultTypeExpr
			var atRegistration, atCallback int
			var calls int
			root := expr.RunDSL(t, func() {
				item := ResultType("application/vnd.body-item", func() {
					Attributes(func() {
						Attribute("name", String)
					})
					View("default", func() {
						Attribute("name")
					})
				})
				if reused {
					collection = CollectionOf(item)
				}
				Service("body_collection", func() {
					Method("create", func() {
						Payload(func() {
							Attribute("items", ArrayOf(item))
						})
						HTTP(func() {
							POST("/")
							atRegistration = len(*expr.GeneratedResultTypes)
							Body(func() {
								calls++
								collection = CollectionOf(item)
								atCallback = len(*expr.GeneratedResultTypes)
								Attribute("items", collection)
							})
						})
					})
				})
			})
			require.Equal(t, 1, calls)
			if reused {
				require.Equal(t, atRegistration, atCallback)
			} else {
				require.Equal(t, atRegistration+1, atCallback)
			}
			require.NotNil(t, expr.AsArray(collection.Type))
			endpoint := root.API.HTTP.Service("body_collection").Endpoint("create")
			_, source := endpoint.MethodExpr.Payload.FindAttribute("items")
			_, selected := endpoint.Body.FindAttribute("items")
			require.NotNil(t, source)
			require.NotNil(t, selected)
			require.NotNil(t, expr.AsArray(source.Type))
			require.NotNil(t, expr.AsArray(selected.Type))
			t.Logf("callback_count=%d generated_roots_before=%d after=%d source=%s body=%s collection=%s", calls, atRegistration, atCallback, source.Type.Name(), selected.Type.Name(), collection.Name())
		})
	}
}

func TestBodyTemplateLateCollectionSource(t *testing.T) {
	var collection *expr.ResultTypeExpr
	var sourcePending bool
	calls := 0
	root := expr.RunDSL(t, func() {
		item := ResultType("application/vnd.late-template-item", func() {
			Attributes(func() {
				Attribute("count", Int, func() {
					Maximum(10)
					Default(7)
				})
			})
			View("default", func() {
				Attribute("count")
			})
		})
		Service("late", func() {
			Method("create", func() {
				Payload(func() {
					collection = CollectionOf(item)
					Attribute("items", collection)
				})
				HTTP(func() {
					POST("/")
					sourcePending = expr.AsArray(collection.Type) == nil
					Body(func() {
						calls++
						Attribute("items")
					})
				})
			})
		})
	})
	require.True(t, sourcePending, "the source collection must still await its generated-root DSL")
	require.Equal(t, 1, calls)
	endpoint := root.API.HTTP.Service("late").Endpoint("create")
	selected := bodyChild(t, endpoint.Body, "items")
	array := expr.AsArray(selected.Type)
	require.NotNil(t, array)
	count := bodyChild(t, array.ElemType, "count")
	require.Equal(t, expr.Int, count.Type)
	require.Equal(t, 7, count.DefaultValue)
	require.NotNil(t, count.Validation.Maximum)
	bound, err := expr.NewNumericBound(10)
	require.NoError(t, err)
	require.Equal(t, bound, *count.Validation.Maximum)
}
