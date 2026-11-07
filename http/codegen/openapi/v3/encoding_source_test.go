package openapiv3_test

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/CaliLuke/loom/internal/openapiimport"
)

func TestEncodingSourceRenderedContract(t *testing.T) {
	artifacts := renderOpenAPIArtifacts(t, testdata.EncodingSourceDSL)
	validateOpenAPI(t, artifacts.JSON)
	validateOpenAPI(t, artifacts.YAML)
	spec := decodeOpenAPIJSON(t, artifacts.JSON)
	components := requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
	resolve := func(value map[string]any) map[string]any {
		for {
			ref, ok := value["$ref"].(string)
			if !ok {
				return value
			}
			value = requireMap(t, components[strings.TrimPrefix(ref, "#/components/schemas/")], ref)
		}
	}
	for _, pair := range []struct{ path, media string }{{"/form", "application/x-www-form-urlencoded"}, {"/multipart", "multipart/form-data"}} {
		t.Run(pair.path, func(t *testing.T) {
			operation := requireOperation(t, spec, pair.path+"/fields", "post")
			body := requireMap(t, operation["requestBody"], "requestBody")
			media := requireMap(t, requireMap(t, body["content"], "content")[pair.media], "media")
			encoding := requireMap(t, media["encoding"], "encoding")
			require.NotContains(t, encoding, "mixed")
			if pair.media == "multipart/form-data" {
				require.NotContains(t, encoding, "nested")
			} else {
				require.Equal(t, "application/json", requireMap(t, encoding["nested"], "nested")["contentType"])
			}
			require.Equal(t, "application/example+json; charset=utf-8", requireMap(t, encoding["whole"], "whole")["contentType"])
			require.Equal(t, "text/plain; charset=utf-8", requireMap(t, encoding["wire"], "wire")["contentType"])
			require.NotContains(t, encoding, "renamed")
			styled := requireMap(t, encoding["styled"], "styled")
			require.Equal(t, "form", styled["style"])
			require.NotContains(t, styled, "contentType")
			schema := resolve(requireMap(t, media["schema"], "schema"))
			properties := requireMap(t, schema["properties"], "properties")
			mixed := resolve(requireMap(t, properties["mixed"], "mixed"))
			require.Equal(t, "array", mixed["type"])
			require.Len(t, requireSlice(t, mixed["oneOf"], "oneOf"), 2)
			whole := resolve(requireMap(t, properties["whole"], "whole"))
			require.NotContains(t, whole, "type")
			singleton := resolve(requireMap(t, properties["singleton"], "singleton"))
			require.NotContains(t, singleton, "type")
			branches := requireSlice(t, singleton["oneOf"], "singleton.oneOf")
			require.Len(t, branches, 1)
			require.Equal(t, "array", resolve(requireMap(t, branches[0], "singleton branch"))["type"])
			roots := requireOperation(t, spec, pair.path+"/roots", "post")
			rootsMedia := requireMap(t, requireMap(t, requireMap(t, roots["requestBody"], "requestBody")["content"], "content")[pair.media], "roots media")
			require.NotContains(t, rootsMedia, "encoding", "each root alternative keeps its own item default")
		})
	}
	// This round trip covers body contracts. Document identity and generated
	// server metadata are outside the strict importer's admitted subset.
	contract, err := json.Marshal(map[string]any{"openapi": spec["openapi"], "info": spec["info"], "paths": spec["paths"], "components": spec["components"]}, json.Deterministic(true))
	require.NoError(t, err)
	document, diagnostics, err := openapiimport.Analyze(contract)
	require.NoError(t, err)
	fatal, _ := diagnostics.Classify(false)
	require.Empty(t, fatal)
	design, err := openapiimport.Render(document, openapiimport.Options{})
	require.NoError(t, err)
	require.Contains(t, string(design), `Meta("http:encoding:sourceType", "array")`)
	require.Contains(t, string(design), `Meta("http:encoding:sourceType", "absent")`)
	require.Contains(t, string(design), `application/example+json; charset=utf-8`)
	if os.Getenv("LOOM_OPENAPI_CONTRACT") != "" {
		path := filepath.Join(t.TempDir(), "encoding-source.json")
		require.NoError(t, os.WriteFile(path, artifacts.JSON, 0o600))
		require.NoError(t, installContractVacuum(t).lint(path))
	}
}

func TestEncodingSourceConflictReportsField(t *testing.T) {
	for _, multipart := range []bool{false, true} {
		root := httpgen.RunHTTPDSL(t, func() {
			left := Type("EncodingConflictLeft", func() {
				Attribute("kind", String, func() { Enum("left") })
				Attribute("value", String)
				Required("kind")
				Meta("openapi:encoding:value:contentType", "application/json")
			})
			right := Type("EncodingConflictRight", func() {
				Attribute("kind", String, func() { Enum("right") })
				Attribute("value", String)
				Required("kind")
				Meta("openapi:encoding:value:contentType", "text/plain")
			})
			body := Type("EncodingConflict", OneOf(func() {
				Untagged()
				Attribute("left", left)
				Attribute("right", right)
			}))
			Service("conflict", func() {
				Method("send", func() {
					Payload(body)
					HTTP(func() {
						POST("/conflict")
						if multipart {
							MultipartRequest()
						} else {
							FormRequest()
						}
					})
				})
			})
		})
		_, err := openapiv3.Files(root)
		require.ErrorContains(t, err, `field "value"`)
		require.ErrorContains(t, err, "incompatible Encoding Objects")
	}
}
