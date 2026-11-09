package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestResultTypeLocalizationCompletesDeclaration(t *testing.T) {
	for _, shape := range []string{"direct", "array", "map"} {
		for _, order := range []string{"result-first", "holder-first"} {
			t.Run(shape+"/"+order, func(t *testing.T) {
				var record *expr.ResultTypeExpr
				calls := 0
				root := expr.RunDSL(t, func() {
					result := func() {
						record = ResultType("application/vnd.localized.record", func() {
							calls++
							TypeName("LocalizedRecord")
							Attributes(func() {
								Attribute("id", String)
								Attribute("name", String)
								Required("id")
							})
							View("default", func() {
								Attribute("id")
								Attribute("name")
							})
							View("tiny", func() {
								Attribute("id")
							})
						})
					}
					holder := func() {
						Type("LocalizedHolder", func() {
							var typ expr.DataType = record
							switch shape {
							case "array":
								typ = ArrayOf(record)
							case "map":
								typ = MapOf(String, record)
							}
							Attribute("custom", typ, func() {
								Meta("probe:key", "value")
							})
							Attribute("plain", record)
						})
					}
					if order == "result-first" {
						result()
						holder()
					} else {
						holder()
						result()
					}
				})
				require.Equal(t, 1, calls)
				custom := root.UserType("LocalizedHolder").Attribute().Find("custom")
				require.Equal(t, []string{"value"}, custom.Meta["probe:key"])
				typ := custom.Type
				switch shape {
				case "array":
					typ = expr.AsArray(typ).ElemType.Type
				case "map":
					typ = expr.AsMap(typ).ElemType.Type
				}
				local, ok := typ.(*expr.ResultTypeExpr)
				require.True(t, ok)
				require.NotSame(t, record, local)
				require.Equal(t, record.Name(), local.Name())
				require.Len(t, *expr.AsObject(local), 2)
				require.True(t, local.IsRequired("id"))
				for _, view := range []string{"default", "tiny"} {
					projected, err := expr.Project(local, view)
					require.NoError(t, err)
					require.NotNil(t, projected.Find("id"))
					require.Equal(t, view == "default", projected.Find("name") != nil)
				}
				require.Same(t, record, root.UserType("LocalizedHolder").Attribute().Find("plain").Type)
				require.NotContains(t, record.Meta, "probe:key")
			})
		}
	}
}

func TestResultTypeLocalizationRetainsRecursiveOwner(t *testing.T) {
	var record *expr.ResultTypeExpr
	root := expr.RunDSL(t, func() {
		holder := Type("RecursiveHolder", func() {
			Attribute("record", record, func() {
				Description("Localized recursive record")
			})
		})
		record = ResultType("application/vnd.localized.recursive", func() {
			Attributes(func() {
				Attribute("id", String)
				Attribute("holder", holder)
			})
			View("default", func() {
				Attribute("id")
				Attribute("holder")
			})
		})
	})
	holder := root.UserType("RecursiveHolder")
	local := holder.Attribute().Find("record").Type.(*expr.ResultTypeExpr)
	require.Len(t, *expr.AsObject(local), 2)
	require.Same(t, holder, local.Find("holder").Type)
	require.Len(t, *expr.AsObject(local.View("default").Type), 2)
}
