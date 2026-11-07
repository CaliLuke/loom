package main

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/openapiimport"
)

func TestComparisonInputPreservesSchemasAndLiterals(t *testing.T) {
	for _, version := range []string{"3.1.1", "3.2.0"} {
		t.Run(version, func(t *testing.T) {
			for _, token := range []string{"18446744073709551615", "-9007199254740993", "1e-400", "1e400"} {
				components := fmt.Sprintf(`{"schemas":{"Text":{"type":"string"},"Occurrence":{"$ref":"#/components/schemas/Text","enum":["text"],"description":"occurrence","x-data":{"$ref":"literal","allOf":["literal"],"number":%s},"default":%s,"example":%s}}}`, token, token, token)
				source := comparisonGeneratedInput(version, components)
				prepared, err := prepareGeneratedOpenAPIContract(source, "Comparison")
				require.NoError(t, err)
				var got map[string]jsontext.Value
				require.NoError(t, json.Unmarshal(prepared, &got))
				require.Equal(t, components, string(got["components"]))
				require.Contains(t, string(got["components"]), `"number":`+token)
				require.Contains(t, string(got["components"]), `"default":`+token)
				require.Contains(t, string(got["components"]), `"example":`+token)
				require.NotContains(t, got, "servers")
				require.NotContains(t, got, "jsonSchemaDialect")
			}
		})
	}
}

func TestComparisonInputReferenceAdmission(t *testing.T) {
	for _, version := range []string{"3.1.1", "3.2.0"} {
		for _, tc := range []struct {
			sibling  string
			admitted bool
		}{
			{`"description":"occurrence"`, true},
			{`"enum":["text"]`, true},
			{`"pattern":"x"`, true},
			{`"maxLength":2`, true},
			{`"items":{"type":"string"}`, false},
			{`"allOf":[{"type":"string"}]`, false},
		} {
			t.Run(version+"/"+tc.sibling, func(t *testing.T) {
				source := comparisonGeneratedInput(version, fmt.Sprintf(`{"schemas":{"Text":{"type":"string"},"Occurrence":{"$ref":"#/components/schemas/Text",%s}}}`, tc.sibling))
				prepared, err := prepareGeneratedOpenAPIContract(source, "Comparison")
				require.NoError(t, err)
				_, diagnostics, err := openapiimport.Analyze(prepared)
				require.NoError(t, err)
				if tc.admitted {
					require.Empty(t, diagnostics)
				} else {
					require.NotEmpty(t, diagnostics)
				}
			})
		}
	}
}

func TestComparisonInputRejectsUnexpectedGeneratedAnnotations(t *testing.T) {
	source := comparisonGeneratedInput("3.2.0", `{}`)
	for _, tc := range []struct {
		field string
		value jsontext.Value
	}{
		{"servers", jsontext.Value(`[{"url":"https://unexpected.example"}]`)},
		{"jsonSchemaDialect", jsontext.Value(`"https://unexpected.example/dialect"`)},
	} {
		changed, err := replaceComparisonJSONField(source, []string{tc.field}, tc.value)
		require.NoError(t, err)
		_, err = prepareGeneratedOpenAPIContract(changed, "Comparison")
		require.Error(t, err)
	}
}

func comparisonGeneratedInput(version, components string) []byte {
	name := ""
	if version == "3.2.0" {
		name = `,"name":"Comparison"`
	}
	return []byte(fmt.Sprintf(`{"openapi":%q,"info":{"title":"Comparison","version":"1"},"paths":{},"jsonSchemaDialect":"https://spec.openapis.org/oas/3.1/dialect/base","servers":[{"url":"http://localhost:80","description":"Default server for Comparison"%s}],"components":%s}`, version, name, components))
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
