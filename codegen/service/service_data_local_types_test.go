package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestServiceTypesKeepLaterDeclaredTypesRefinedByAttributeDSL(t *testing.T) {
	root := expr.RunDSL(t, func() {
		var byVariable expr.UserType
		holder := dsl.Type("LocalHolder", func() {
			dsl.Attribute("direct", "LocalLater", func() {
				dsl.Description("described locally")
			})
			dsl.Attribute("variable", byVariable, func() {
				dsl.Description("described locally")
			})
			dsl.Attribute("list", dsl.ArrayOf("LocalLaterElem"), func() {
				dsl.MinLength(1)
			})
			dsl.Attribute("index", dsl.MapOf(dsl.String, "LocalLaterValue"), func() {
				dsl.MinLength(1)
			})
		})
		dsl.Type("LocalLater", func() {
			dsl.Attribute("name", dsl.String)
		})
		byVariable = dsl.Type("LocalLaterVariable", func() {
			dsl.Attribute("label", dsl.String)
		})
		dsl.Type("LocalLaterElem", func() {
			dsl.Attribute("age", dsl.Int)
		})
		dsl.Type("LocalLaterValue", func() {
			dsl.Attribute("weight", dsl.Float64)
		})
		dsl.Service("LocalService", func() {
			dsl.Method("Show", func() {
				dsl.Payload(holder)
				dsl.Result(holder)
			})
		})
	})

	data := NewServicesData(root).Get("LocalService")
	defs := make(map[string]string, len(data.userTypes))
	for _, userType := range data.userTypes {
		defs[userType.Name] = userType.Def
	}
	for name, field := range map[string]string{
		"LocalLater":         "Name",
		"LocalLaterVariable": "Label",
		"LocalLaterElem":     "Age",
		"LocalLaterValue":    "Weight",
	} {
		require.Contains(t, defs, name)
		require.Contains(t, defs[name], field, name)
	}
}
