package codegen

import (
	"encoding/json/v2"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

// TestMappedNamesOpenAPIElementNames checks that the OpenAPI document of
// MappedNamesDSL parses and names the properties, required fields and
// examples of attributes declared with an element name suffix, such as
// "n:m", after the element name that the HTTP bodies use.
func TestMappedNamesOpenAPIElementNames(t *testing.T) {
	root := RunHTTPDSL(t, testdata.MappedNamesDSL)
	spec := renderOpenAPIJSON(t, OpenAPIFiles, root)
	parseOpenAPIV3Document(t, spec)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(spec, &doc))
	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	for name, want := range map[string]struct {
		properties []string
		required   []string
	}{
		"Envelope":   {[]string{"m", "r", "d", "p", "o", "ls", "ix", "ch"}, []string{"r", "p", "o"}},
		"Leaf":       {[]string{"l", "c"}, []string{"c"}},
		"DataResult": {[]string{"dt", "sz"}, []string{"dt"}},
		"FailResult": {[]string{"rs"}, []string{"rs"}},
	} {
		schema, ok := schemas[name].(map[string]any)
		if !assert.True(t, ok, name) {
			continue
		}
		assert.ElementsMatch(t, want.properties, mapKeys(schema["properties"]), name)
		assert.ElementsMatch(t, want.required, schema["required"], name)
		example, ok := schema["example"].(map[string]any)
		if assert.True(t, ok, "%s example", name) {
			assert.Subset(t, want.properties, mapKeys(example), "%s example", name)
			for _, required := range want.required {
				assert.Contains(t, example, required, "%s example", name)
			}
		}
	}
	echo := doc["paths"].(map[string]any)["/echo"].(map[string]any)["post"].(map[string]any)
	body := echo["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)
	assert.Subset(t, []string{"m", "r", "d", "p", "o", "ls", "ix", "ch"}, mapKeys(body["example"]))
	assert.Empty(t, suffixedKeys(doc), "keys with an element name suffix")
}

// TestMappedNamesCLIBodyExamples checks that the client CLI body examples of
// MappedNamesDSL use the element names of suffixed attributes.
func TestMappedNamesCLIBodyExamples(t *testing.T) {
	root := RunHTTPDSL(t, testdata.MappedNamesDSL)
	cli := filesCode(t, ClientCLIFiles("gen", CreateHTTPServices(root)))

	for _, want := range []string{`mappednames echo --body '{\n      \"ch\": {`, `\"m\": `, `\"i\": `, `mappednames lookup --body '{\n      \"rs\": `} {
		assert.Contains(t, cli, want)
	}
	assert.Empty(t, regexp.MustCompile(`\\"[a-z_]+:[a-z]+\\"`).FindAllString(cli, -1), "keys with an element name suffix")
}

func mapKeys(value any) []string {
	object, _ := value.(map[string]any)
	names := make([]string, 0, len(object))
	for name := range object {
		names = append(names, name)
	}
	return names
}

// suffixedKeys returns the object keys and required names in value that
// contain a colon, such as "n:m".
func suffixedKeys(value any) []string {
	var found []string
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			if strings.Contains(key, ":") {
				found = append(found, key)
			}
			if names, ok := item.([]any); ok && key == "required" {
				for _, name := range names {
					if s, ok := name.(string); ok && strings.Contains(s, ":") {
						found = append(found, s)
					}
				}
			}
			found = append(found, suffixedKeys(item)...)
		}
	case []any:
		for _, item := range v {
			found = append(found, suffixedKeys(item)...)
		}
	}
	return found
}
