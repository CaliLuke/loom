package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestSelectedResponseStorageReadiness(t *testing.T) {
	for _, tc := range []struct {
		name                                  string
		viewed, required, nullable, defaulted bool
		kind                                  string
	}{
		{"required viewed scalar", true, true, false, false, "string"},
		{"optional viewed scalar", true, false, false, false, "string"},
		{"defaulted viewed scalar", true, false, false, true, "string"},
		{"required viewed named scalar", true, true, false, false, "named"},
		{"required viewed custom scalar", true, true, false, false, "custom"},
		{"required viewed nullable scalar", true, true, true, false, "string"},
		{"optional viewed nullable scalar", true, false, true, false, "string"},
		{"viewed array", true, true, false, false, "array"},
		{"viewed bytes", true, true, false, false, "bytes"},
		{"required service scalar", false, true, false, false, "string"},
		{"optional service scalar", false, false, false, false, "string"},
		{"optional service nullable scalar", false, false, true, false, "string"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				value := any(dsl.String)
				switch tc.kind {
				case "named":
					value = dsl.Type("Text", dsl.String)
				case "array":
					value = dsl.ArrayOf(dsl.String)
				case "bytes":
					value = dsl.Bytes
				}
				fields := func() {
					dsl.Attribute("value", value, func() {
						if tc.kind == "custom" {
							dsl.Meta("struct:field:type", "time.Time", "time")
						}
						if tc.nullable {
							dsl.Nullable()
						}
						if tc.defaulted {
							dsl.Default("fallback")
						}
					})
					if tc.required {
						dsl.Required("value")
					}
				}
				var result any = dsl.Type("Output", fields)
				if tc.viewed {
					result = dsl.ResultType("application/vnd.output", func() {
						fields()
						dsl.View("default", func() {
							dsl.Attribute("value")
						})
					})
				}
				dsl.Service("fields", func() {
					dsl.Method("show", func() {
						dsl.Result(result)
						dsl.HTTP(func() {
							dsl.GET("/")
							dsl.Response(dsl.StatusOK, func() {
								dsl.Body("value")
							})
						})
					})
				})
			})
			data := CreateHTTPServices(root).Get("fields")
			result := root.Service("fields").Method("show").Result
			attribute := result.Find("value")
			layout := expr.ServiceFieldLayout(result, "value", attribute, tc.viewed, true)
			wantAddress := expr.IsPrimitive(attribute.Type) && layout.Pointer
			init := data.Endpoints[0].Result.Responses[0].ResultInit
			require.NotNil(t, init)
			t.Logf("target=%s layoutPointer=%t primitive=%t transform=\n%s", init.ReturnTypeRef, layout.Pointer, expr.IsPrimitive(attribute.Type), init.ClientCode)
			require.Equal(t, wantAddress, init.ReturnIsPrimitivePointer,
				"constructor target %s must agree with declaration storage", init.ReturnTypeRef)
		})
	}
}

func TestSelectedResponseSourceStorage(t *testing.T) {
	root := RunHTTPDSL(t, testdata.SelectedResponseStorageDSL)
	data := CreateHTTPServices(root).Get("httpfields")
	checked := 0
	for _, endpoint := range data.Endpoints {
		if endpoint.Method.Name != "named" && endpoint.Method.Name != "namedoptional" && endpoint.Method.Name != "ordinarynamedoptional" {
			continue
		}
		checked++
		t.Run(endpoint.Method.Name, func(t *testing.T) {
			body := endpoint.Result.Responses[0].ServerBody[0]
			require.NotNil(t, body.Init)
			expected := "*Text"
			if endpoint.Method.Name == "namedoptional" {
				expected = "*OptionalText"
			}
			if endpoint.Method.Name == "ordinarynamedoptional" {
				expected = "*ServiceOptionalText"
			}
			require.Equal(t, expected, body.Init.ReturnTypeRef,
				"selected scalar conversion must preserve the source pointer's absence")
			require.Equal(t, body.Init.ReturnTypeRef, body.Ref)
		})
	}
	require.Equal(t, 3, checked)
}

func TestServiceFieldPrimitiveStorageContext(t *testing.T) {
	parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{
		Type: expr.String, DefaultValue: "fallback",
	}}}}
	for _, tc := range []struct {
		name                    string
		pointer, defaults, want bool
	}{
		{"materialized default", false, true, false},
		{"unmaterialized default", false, false, true},
		{"view default", true, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			context := serviceContext("", nil)
			context.Pointer, context.UseDefault = tc.pointer, tc.defaults
			require.Equal(t, tc.want, serviceFieldPrimitivePointer(parent, "value", context))
		})
	}
}
