package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestUntaggedAcceptsStringMapFields(t *testing.T) {
	root := expr.RunDSL(t, func() {
		key := Type("Key", String)
		item := Type("Item", func() {
			Attribute("id", String)
			Required("id")
		})
		lookup := Type("Lookup", MapOf(key, item))
		tree := Type("Tree", func() {
			Attribute("id", String)
			Attribute("children", MapOf(String, "Tree"))
			Required("id")
		})
		record := Type("Record", func() {
			Attribute("id", String)
			Attribute("properties", MapOf(String, Any))
			Attribute("labels", MapOf(String, String))
			Attribute("records", lookup)
			Attribute("arrays", ArrayOf(MapOf(String, ArrayOf(item))))
			Attribute("nested", MapOf(String, MapOf(String, item)))
			Attribute("tree", tree)
			Required("id")
		})
		absent := Type("Absent", func() {
			Attribute("missing", Boolean)
			Required("missing")
		})
		Service("maps", func() {
			Method("read", func() {
				Result(OneOf(record, absent), func() {
					Untagged()
				})
			})
		})
	})
	require.NotNil(t, root.Service("maps"))
}

func TestUntaggedRejectsNonStringMapKeys(t *testing.T) {
	for _, key := range []expr.DataType{Int, Boolean} {
		err := expr.RunInvalidDSL(t, func() {
			record := Type("Record", func() {
				Attribute("values", MapOf(key, String))
			})
			absent := Type("Absent", func() {
				Attribute("missing", Boolean)
			})
			Service("maps", func() {
				Method("read", func() {
					Result(OneOf(record, absent), func() {
						Untagged()
					})
				})
			})
		})
		require.ErrorContains(t, err, "map keys must be strings")
	}
}

func TestUntaggedRejectsUnsupportedMapValues(t *testing.T) {
	for _, tc := range []struct {
		name    string
		field   func()
		message string
	}{
		{"opaque key", func() {
			Attribute("values", MapOf(String, String), func() {
				Key(func() {
					Meta("struct:field:type", "Custom", "example.com/custom")
				})
			})
		}, "opaque codec"},
		{"opaque value", func() {
			Attribute("values", MapOf(String, String), func() {
				Elem(func() {
					Meta("openapi:contentEncoding", "hex")
				})
			})
		}, "schema encoding override"},
		{"inline object", func() {
			Attribute("values", MapOf(String, &expr.Object{{Name: "name", Attribute: &expr.AttributeExpr{Type: String}}}))
		}, "map values must be primitive"},
		{"nested union", func() {
			Attribute("values", MapOf(String, "Choice"))
		}, "map values must be primitive"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				first := Type("First", func() {
					Attribute("name", String)
				})
				second := Type("Second", func() {
					Attribute("id", String)
				})
				Type("Choice", OneOf(first, second), Untagged)
				record := Type("Record", tc.field)
				other := Type("Other", func() {
					Attribute("missing", Boolean)
				})
				Service("maps", func() {
					Method("read", func() {
						Result(OneOf(record, other), Untagged)
					})
				})
			})
			require.ErrorContains(t, err, tc.message)
		})
	}
}
