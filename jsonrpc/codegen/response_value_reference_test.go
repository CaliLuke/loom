package codegen

import (
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/dsl"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

// TestJSONRPCResponseValueReference checks the body metadata and emitted local agree.
func TestJSONRPCResponseValueReference(t *testing.T) {
	for _, nullable := range []bool{false, true} {
		mode := "ordinary"
		if nullable {
			mode = "nullable"
		}
		t.Run(mode, func(t *testing.T) {
			root := RunJSONRPCDSL(t, responseValueReferenceDSL(nullable))
			services := CreateJSONRPCServices(root)
			for _, method := range []string{"object", "scalar", "array", "map"} {
				t.Run(method, func(t *testing.T) {
					endpoint := services.Get("probe").Endpoint(method)
					require.Len(t, endpoint.Result.Responses, 1)
					body := endpoint.Result.Responses[0].ClientBody
					require.NotEmpty(t, body.ValueRef)
					if nullable {
						if method == "object" {
							require.NotEqual(t, body.VarName, body.ValueRef)
						}
						require.Contains(t, body.ValueRef, "loom.Nullable[")
					} else {
						require.Equal(t, body.VarName, body.ValueRef)
					}
					source := renderGroupWriter(t, func(g *jen.Group) {
						writeSingleResponseDecode(g, endpoint.Result.Responses[0], "probe", endpoint.Method)
					})
					require.Equal(t, body.ValueRef, responseBodyLocalType(t, source))
				})
			}
		})
	}
}

// TestJSONRPCErrorResponseValueReference checks the shared error decoding consumer.
func TestJSONRPCErrorResponseValueReference(t *testing.T) {
	for _, reference := range []string{"FailureBody", "loom.Nullable[FailureBody]"} {
		t.Run(reference, func(t *testing.T) {
			response := &httpcodegen.ResponseData{
				ClientBody: &httpcodegen.TypeData{VarName: "FailureBody", ValueRef: reference},
			}
			source := renderGroupWriter(t, func(g *jen.Group) {
				writeSingleResponseDecode(g, response, "probe", &service.MethodData{Name: "show"})
			})
			require.Equal(t, reference, responseBodyLocalType(t, source))
			require.NotContains(t, source, "resp.Body =", "the candidate response already owns its detached data")
		})
	}
}

func responseBodyLocalType(t *testing.T, source string) string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "decoder.go", "package client\n"+source, 0)
	require.NoError(t, err)
	var bodyType string
	ast.Inspect(file, func(node ast.Node) bool {
		declaration, ok := node.(*ast.ValueSpec)
		if !ok || len(declaration.Names) != 1 || declaration.Names[0].Name != "body" {
			return true
		}
		var output strings.Builder
		require.NoError(t, format.Node(&output, token.NewFileSet(), declaration.Type))
		bodyType = output.String()
		return false
	})
	require.NotEmpty(t, bodyType)
	return bodyType
}

func responseValueReferenceDSL(nullable bool) func() {
	return func() {
		dsl.API("response-value-reference", func() {
			dsl.JSONRPC(func() {
			})
		})
		presence := func() {
			if nullable {
				dsl.Nullable()
			}
		}
		object := dsl.Type("Object", func() {
			presence()
			dsl.Attribute("text", dsl.String, func() {
				dsl.MinLength(1)
			})
			dsl.Required("text")
		})
		scalar := dsl.Type("Scalar", dsl.String, func() {
			presence()
			dsl.MinLength(1)
		})
		array := dsl.Type("List", dsl.ArrayOf(dsl.String), func() {
			presence()
			dsl.MinLength(1)
		})
		mapping := dsl.Type("Index", dsl.MapOf(dsl.String, dsl.String), func() {
			presence()
			dsl.MinLength(1)
		})
		failure := dsl.Type("Failure", func() {
			dsl.Attribute("message", dsl.String)
			dsl.Required("message")
		})
		dsl.Service("probe", func() {
			dsl.JSONRPC(func() {
				dsl.POST("/rpc")
			})
			for _, item := range []struct {
				name string
				kind any
			}{
				{"object", object},
				{"scalar", scalar},
				{"array", array},
				{"map", mapping},
			} {
				dsl.Method(item.name, func() {
					dsl.Result(item.kind)
					dsl.Error("failed", failure)
					dsl.JSONRPC(func() {
						dsl.Response("failed", func() {
							dsl.Code(4200)
						})
					})
				})
			}
		})
	}
}
