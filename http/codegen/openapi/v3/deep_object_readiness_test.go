package openapiv3_test

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestDeepObjectReadinessQueryExport(t *testing.T) {
	for _, test := range []struct {
		name string
		dsl  func()
		path string
	}{
		{"existing map fixture", testdata.PayloadQueryMapStringStringDSL, "/"},
		{"aliased map", func() {
			filter := Type("Filter", MapOf(String, String))
			Service("search", func() {
				Method("read", func() {
					Payload(func() {
						Attribute("filter", filter)
					})
					HTTP(func() {
						GET("/")
						Param("filter:filters")
					})
				})
			})
		}, "/"},
	} {
		t.Run(test.name, func(t *testing.T) {
			spec := decodeOpenAPIJSON(t, renderOpenAPIArtifacts(t, test.dsl).JSON)
			operation := requireOperation(t, spec, test.path, "get")
			params := requireSlice(t, operation["parameters"], "parameters")
			parameter := requireMap(t, params[0], "parameter")
			content := requireMap(t, parameter["content"], "parameter content")
			media := requireMap(t, content["application/json"], "JSON media")
			require.NotEmpty(t, media["schema"])
			require.NotContains(t, parameter, "schema")
			require.NotContains(t, parameter, "style")
			require.NotContains(t, parameter, "explode")
		})
	}
}

// Candidate document validates the accepted JSON representation before production.
func TestUniformJSONReadinessNativeContract(t *testing.T) {
	for _, version := range []string{"3.1.1", "3.2.0"} {
		t.Run(version, func(t *testing.T) {
			object := map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}}}
			spec := map[string]any{
				"openapi": version,
				"info":    map[string]any{"title": "JSON fields", "version": "1"},
				"paths": map[string]any{"/search": map[string]any{"post": map[string]any{
					"operationId": "search",
					"parameters": []any{
						map[string]any{"in": "query", "name": "filter", "content": map[string]any{"application/json": map[string]any{"schema": object}}},
						map[string]any{"in": "query", "name": "page", "schema": map[string]any{"type": "integer"}},
					},
					"requestBody": map[string]any{"content": map[string]any{"application/x-www-form-urlencoded": map[string]any{
						"schema":   map[string]any{"type": "object", "properties": map[string]any{"filter": object}},
						"encoding": map[string]any{"filter": map[string]any{"contentType": "application/json"}},
					}}},
					"responses": map[string]any{"204": map[string]any{"description": "Done"}},
				}}},
			}
			data, err := json.Marshal(spec, json.Deterministic(true))
			require.NoError(t, err)
			validateOpenAPIVersion(t, data, version)
			if os.Getenv("LOOM_OPENAPI_CONTRACT") != "" {
				path := filepath.Join(t.TempDir(), "json-fields.json")
				require.NoError(t, os.WriteFile(path, data, 0o600))
				require.NoError(t, installContractVacuum(t).lint(path))
			}
		})
	}
}

func TestDeepObjectReadinessFormExport(t *testing.T) {
	spec := decodeOpenAPIJSON(t, renderOpenAPIArtifacts(t, uniformJSONFormDSL).JSON)
	operation := requireOperation(t, spec, "/", "post")
	body := requireMap(t, operation["requestBody"], "body")
	content := requireMap(t, body["content"], "content")
	media := requireMap(t, content["application/x-www-form-urlencoded"], "form media")
	encoding := requireMap(t, media["encoding"], "encoding")
	for _, name := range []string{"address", "labels"} {
		t.Run(name, func(t *testing.T) {
			field := requireMap(t, encoding[name], "field encoding")
			require.Equal(t, "application/json", field["contentType"])
			require.NotContains(t, field, "style")
			require.NotContains(t, field, "explode")
			require.NotContains(t, field, "allowReserved")
		})
	}
}

func TestDeepObjectReadinessExportControls(t *testing.T) {
	for _, design := range []func(){testdata.PayloadQueryStringDSL, testdata.PayloadQueryArrayStringDSL} {
		spec := decodeOpenAPIJSON(t, renderOpenAPIArtifacts(t, design).JSON)
		operation := requireOperation(t, spec, "/", "get")
		parameters := requireSlice(t, operation["parameters"], "parameters")
		parameter := requireMap(t, parameters[0], "parameter")
		require.NotContains(t, parameter, "style", "existing form defaults are valid for scalar/repeated scalar fields")
	}
}

func TestDeepObjectReadinessWholeQuery31(t *testing.T) {
	spec := decodeOpenAPIJSON(t, renderOpenAPIArtifactsForVersion(t, func() {
		API("maps", func() {
			Meta("openapi:version", "3.1")
		})
		Service("maps", func() {
			Method("read", func() {
				Payload(MapOf(String, String))
				HTTP(func() {
					GET("/")
					MapParams()
				})
			})
		})
	}, "3.1", "3.1.1").JSON)
	operation := requireOperation(t, spec, "/", "get")
	parameters := requireSlice(t, operation["parameters"], "parameters")
	parameter := requireMap(t, parameters[0], "parameter")
	require.Equal(t, "form", parameter["style"], "whole-query maps use unprefixed dynamic names, not payload[key]")
}

func TestJSONFieldMetadataAdmission(t *testing.T) {
	for _, location := range []string{"query", "form"} {
		for _, key := range []string{"style", "explode", "allowReserved"} {
			t.Run(location+"/"+key, func(t *testing.T) {
				root := httpgen.RunHTTPDSL(t, func() {
					Service("encoding", func() {
						Method("read", func() {
							Payload(func() { Attribute("filter", MapOf(String, String)) })
							HTTP(func() {
								POST("/")
								value := "true"
								if key == "style" {
									value = "deepObject"
								}
								if location == "query" {
									Param("filter", func() { Meta("openapi:"+key, value) })
								} else {
									FormRequest()
									Body(func() { Attribute("filter"); Meta("openapi:encoding:filter:"+key, value) })
								}
							})
						})
					})
				})
				_, err := openapiv3.Files(root)
				require.ErrorContains(t, err, "encoding.read")
				require.ErrorContains(t, err, "filter")
			})
		}
	}
}

func uniformJSONFormDSL() {
	address := Type("Address", func() {
		Attribute("street", String)
	})
	Service("form", func() {
		Method("submit", func() {
			Payload(func() {
				Attribute("address", address)
				Attribute("labels", MapOf(String, String))
				Attribute("tags", ArrayOf(String))
				Attribute("title", String)
			})
			HTTP(func() {
				POST("/")
				FormRequest()
			})
		})
	})
}

func TestJSONFieldRenderedContracts(t *testing.T) {
	for _, version := range []struct{ target, want string }{{"3.1", "3.1.1"}, {"3.2", "3.2.0"}} {
		for _, design := range []struct {
			name string
			dsl  func()
		}{{"scalar-map-keys", testdata.OpenAPIScalarMapKeysDSL}, {"form", uniformJSONFormDSL}} {
			t.Run(version.target+"/"+design.name, func(t *testing.T) {
				artifacts := renderOpenAPIArtifactsForVersion(t, design.dsl, version.target, version.want)
				if os.Getenv("LOOM_OPENAPI_CONTRACT") == "" {
					return
				}
				vacuum := installContractVacuum(t)
				for format, data := range map[string][]byte{"json": artifacts.JSON, "yaml": artifacts.YAML} {
					path := filepath.Join(t.TempDir(), "contract."+format)
					require.NoError(t, os.WriteFile(path, data, 0o600))
					require.NoError(t, vacuum.lint(path))
				}
			})
		}
	}
}

func TestJSONFieldReusableParameterSchema(t *testing.T) {
	spec := decodeOpenAPIJSON(t, renderOpenAPIArtifacts(t, func() {
		filter := Type("Filter", MapOf(String, String))
		Service("filters", func() {
			for _, name := range []string{"first", "second"} {
				Method(name, func() {
					Payload(func() { Attribute("filter", filter) })
					HTTP(func() {
						GET("/" + name)
						Param("filter", func() { Meta("openapi:component:parameter", "SharedFilter") })
					})
				})
			}
		})
	}).JSON)
	components := requireMap(t, spec["components"], "components")
	parameters := requireMap(t, components["parameters"], "parameters")
	parameter := requireMap(t, parameters["SharedFilter"], "shared parameter")
	media := jsonParameterMedia(t, parameter)
	require.NotEmpty(t, media["schema"])
	require.NotEmpty(t, components["schemas"])
}
