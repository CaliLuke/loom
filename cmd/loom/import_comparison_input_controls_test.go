package main

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/openapiimport"
)

func TestComparisonInputSchemaPositions(t *testing.T) {
	source := []byte(`{
		"openapi":"3.2.0","info":{"title":"Positions","version":"1"},
		"components":{"schemas":{"Text":{"type":"string"},"Nested":{"type":"object","properties":{"child":{"$ref":"#/components/schemas/Text","description":"nested"}}}},
		"parameters":{"Param":{"name":"q","in":"query","schema":{"$ref":"#/components/schemas/Text","description":"parameter"}}},
		"headers":{"Header":{"schema":{"$ref":"#/components/schemas/Text","description":"header"}}},
		"requestBodies":{"Body":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Text","description":"body"}}}}},
		"responses":{"Response":{"description":"ok","content":{"application/json":{"schema":{"$ref":"#/components/schemas/Text","description":"response"}}}}}},
		"paths":{"/value":{"get":{"responses":{"200":{"description":"ok","content":{"application/json":{"schema":{"$ref":"#/components/schemas/Text","description":"operation"}}}}}}}}
	}`)
	before := bytes.Clone(source)
	got, err := rewriteComparisonObject(source, "document")
	require.NoError(t, err)
	require.Equal(t, before, source)
	require.Equal(t, 6, bytes.Count(got, []byte(`"allOf"`)))
	_, diagnostics, err := openapiimport.Analyze(got)
	require.NoError(t, err)
	require.Len(t, diagnostics, 4)
	for _, diagnostic := range diagnostics {
		require.Contains(t, []string{"component-header", "unused-component-parameter", "component-request-body", "component-response"}, diagnostic.Code)
	}
	repeated, err := rewriteComparisonObject(got, "document")
	require.NoError(t, err)
	require.Equal(t, got, repeated)
}

func TestComparisonInputPreservesLiteralDataAndNumbers(t *testing.T) {
	for _, token := range []string{"18446744073709551615", "-9007199254740993", "1e-400", "1e400"} {
		t.Run(token, func(t *testing.T) {
			literal := fmt.Sprintf(`{"$ref":"literal","allOf":["literal"],"number":%s}`, token)
			source := []byte(fmt.Sprintf(`{"type":"number","minimum":%s,"default":%s,"example":%s,"x-data":%s}`, token, literal, literal, literal))
			got, err := rewriteComparisonObject(source, "schema")
			require.NoError(t, err)
			var values map[string]jsontext.Value
			require.NoError(t, json.Unmarshal(got, &values))
			require.Equal(t, token, string(values["minimum"]))
			for _, key := range []string{"default", "example", "x-data"} {
				require.Equal(t, literal, string(values[key]))
			}
		})
	}
}

func TestComparisonInputRejectsUnsupportedReferenceForms(t *testing.T) {
	for _, sibling := range []string{`"pattern":"x"`, `"maxLength":2`, `"unevaluatedProperties":false`} {
		t.Run(sibling, func(t *testing.T) {
			source := []byte(fmt.Sprintf(`{"openapi":"3.2.0","info":{"title":"Rejected","version":"1"},"paths":{},"components":{"schemas":{"Text":{"type":"string"},"Occurrence":{"$ref":"#/components/schemas/Text",%s}}}}`, sibling))
			normalized, err := rewriteComparisonObject(source, "document")
			require.NoError(t, err)
			_, diagnostics, err := openapiimport.Analyze(normalized)
			require.NoError(t, err)
			require.NotEmpty(t, diagnostics, "normalization must not broaden strict schema admission")
		})
	}
	_, err := rewriteComparisonObject([]byte(`{"$ref":"#/components/schemas/Text","allOf":[{"type":"string"}]}`), "schema")
	require.ErrorContains(t, err, "together with allOf")
}

func replaceComparisonJSONField(raw jsontext.Value, path []string, replacement jsontext.Value) (jsontext.Value, error) {
	if len(path) == 0 {
		return replacement, nil
	}
	var object map[string]jsontext.Value
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, err
	}
	child, ok := object[path[0]]
	if !ok {
		return nil, fmt.Errorf("missing comparison path %q", path[0])
	}
	updated, err := replaceComparisonJSONField(child, path[1:], replacement)
	if err != nil {
		return nil, err
	}
	object[path[0]] = updated
	return json.Marshal(object, json.Deterministic(true))
}
