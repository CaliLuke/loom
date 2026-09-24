package expr_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestStructMetaValidation(t *testing.T) {
	cases := []struct {
		Name    string
		Key     string
		Value   string
		OnField bool
		Error   string
	}{
		{Name: "ascii package path", Key: "struct:pkg:path", Value: "shared/types"},
		{Name: "non-ascii package path", Key: "struct:pkg:path", Value: "tipos/menü"},
		{Name: "cjk package path", Key: "struct:pkg:path", Value: "型"},
		{Name: "dotted package path", Key: "struct:pkg:path", Value: "types.v2"},
		{Name: "absolute package path", Key: "struct:pkg:path", Value: "/types", Error: `metadata "struct:pkg:path" value "/types" is not a valid relative Go import path: empty path element`},
		{Name: "parent package path", Key: "struct:pkg:path", Value: "../types", Error: `metadata "struct:pkg:path" value "../types" is not a valid relative Go import path: invalid path element ".."`},
		{Name: "empty element", Key: "struct:pkg:path", Value: "a//b", Error: `metadata "struct:pkg:path" value "a//b" is not a valid relative Go import path: double slash`},
		{Name: "space in package path", Key: "struct:pkg:path", Value: "my types", Error: `metadata "struct:pkg:path" value "my types" is not a valid relative Go import path: invalid char ' '`},
		{Name: "trailing slash", Key: "struct:pkg:path", Value: "types/", Error: `metadata "struct:pkg:path" value "types/" is not a valid relative Go import path: trailing slash`},
		{Name: "field package path", Key: "struct:pkg:path", Value: "a b", OnField: true, Error: `metadata "struct:pkg:path" value "a b" is not a valid relative Go import path: invalid char ' '`},
		{Name: "ascii proto name", Key: "struct:name:proto", Value: "CustomType"},
		{Name: "underscore proto name", Key: "struct:name:proto", Value: "Custom_Type"},
		{Name: "non-ascii proto name", Key: "struct:name:proto", Value: "MenüProto"},
		{Name: "non-ascii digit-leading proto name", Key: "struct:name:proto", Value: "é1"},
		{Name: "empty proto name", Key: "struct:name:proto", Value: "", Error: `metadata "struct:name:proto" value "" is not a valid protocol buffer message name`},
		{Name: "digit-leading proto name", Key: "struct:name:proto", Value: "1Custom", Error: `metadata "struct:name:proto" value "1Custom" is not a valid protocol buffer message name`},
		{Name: "dash in proto name", Key: "struct:name:proto", Value: "Custom-Type", Error: `metadata "struct:name:proto" value "Custom-Type" is not a valid protocol buffer message name`},
		{Name: "qualified proto name", Key: "struct:name:proto", Value: "pkg.Custom", Error: `metadata "struct:name:proto" value "pkg.Custom" is not a valid protocol buffer message name`},
		{Name: "dash in non-ascii proto name", Key: "struct:name:proto", Value: "Menü-Proto", Error: `metadata "struct:name:proto" value "Menü-Proto" is not a valid protocol buffer message name`},
		{Name: "field proto name", Key: "struct:name:proto", Value: "a-b", OnField: true, Error: `metadata "struct:name:proto" value "a-b" is not a valid protocol buffer message name`},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			design := structMetaDSL(c.Key, c.Value, c.OnField)
			if c.Error == "" {
				expr.RunDSL(t, design)
				return
			}
			err := expr.RunInvalidDSL(t, design)
			require.Error(t, err)
			assert.Contains(t, err.Error(), c.Error)
		})
	}
}

func structMetaDSL(key, value string, onField bool) func() {
	return func() {
		API("meta", func() {})
		var Leaf = Type("Leaf", func() {
			Attribute("name", String)
		})
		var Holder = Type("Holder", func() {
			if !onField {
				Meta(key, value)
			}
			Attribute("leaf", Leaf, func() {
				if onField {
					Meta(key, value)
				}
			})
		})
		Service("svc", func() {
			Method("show", func() {
				Payload(Holder)
			})
		})
	}
}
