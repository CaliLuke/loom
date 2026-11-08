package openapiimport

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
)

func TestDiscriminatorImportedRoundtrip(t *testing.T) {
	source := readFixture(t, "discriminator/roundtrip.yaml")
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	directory := requireRenderedDesignGenerates(t, design)
	generated, err := os.ReadFile(filepath.Join(directory, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	var output map[string]any
	require.NoError(t, json.Unmarshal(generated, &output))
	// Existing importer roundtrips exclude unsupported publication metadata.
	delete(output, "jsonSchemaDialect")
	delete(output, "servers")
	roundtrip, err := json.Marshal(output, json.Deterministic(true))
	require.NoError(t, err)
	regenerated, diagnostics, err := Analyze(roundtrip)
	require.NoError(t, err)
	if len(diagnostics) > 0 {
		t.Logf("generated contract: %s", roundtrip)
	}
	require.Empty(t, diagnostics)
	second, err := Render(regenerated, Options{PackageName: "design"})
	require.NoError(t, err)
	require.NotEmpty(t, second)
	requireIndependentRegeneration(t, directory)
	requireDiscriminatorSpecifications(t, directory)
	components := output["components"].(map[string]any)["schemas"].(map[string]any)
	for _, name := range []string{"Event", "Pet", "DuplicatePet", "Parent"} {
		schema := components[name].(map[string]any)
		require.Contains(t, schema, "discriminator", name)
	}
	event := components["Event"].(map[string]any)
	require.Equal(t, false, event["unevaluatedProperties"])
	hint := event["discriminator"].(map[string]any)
	require.Equal(t, "#/components/schemas/EventData", hint["defaultMapping"])
	require.Equal(t, map[string]any{"created": "#/components/schemas/Created", "made": "#/components/schemas/Created", "deleted": "#/components/schemas/Deleted"}, hint["mapping"])
	require.Equal(t, map[string]any{"a": true, "z": float64(1)}, hint["x-audit"])
	require.Len(t, components["DuplicatePet"].(map[string]any)["oneOf"], 3)
	inline := components["InlinePet"].(map[string]any)
	inlineBranches := inline["oneOf"].([]any)
	require.Len(t, inlineBranches, 2)
	firstTarget := inlineBranches[0].(map[string]any)["$ref"]
	secondTarget := inlineBranches[1].(map[string]any)["$ref"]
	require.NotEqual(t, firstTarget, secondTarget, "equal inline schemas retain authored identities")
	require.Equal(t, map[string]any{"first": firstTarget, "second": secondTarget}, inline["discriminator"].(map[string]any)["mapping"])
	require.Equal(t, map[string]any{"propertyName": "kind"}, components["ImplicitPet"].(map[string]any)["discriminator"], "promoted names must not invent implicit mappings")
	require.Equal(t, map[string]any{"propertyName": "kind"}, components["ImplicitParent"].(map[string]any)["discriminator"])
	implicitChild := components["ImplicitChild"].(map[string]any)
	require.Contains(t, implicitChild, "allOf", "otherwise unused implicit child is exported")
	require.Equal(t, "#/components/schemas/ImplicitParent", implicitChild["allOf"].([]any)[0].(map[string]any)["$ref"])
	// Hints never narrow optional tags or route exact-one matching. Compile each
	// source schema and exported schema over the same representative JSON values.
	discriminatorSemanticValues(t, source, output)
	discriminatorHintValues(t, source, output)
	discriminatorClosureValues(t, source, output)
}

func discriminatorSemanticValues(t *testing.T, source []byte, output map[string]any) {
	t.Helper()
	var original map[string]any
	require.NoError(t, yaml.Unmarshal(source, &original))
	values := []any{
		nil,
		"cat",
		map[string]any{},
		map[string]any{"kind": "cat", "name": "M"},
		map[string]any{"kind": "dog", "age": 1},
		map[string]any{"age": 1},
		map[string]any{"kind": "other", "age": 1},
		map[string]any{"kind": "created", "data": map[string]any{"name": "N"}},
		map[string]any{"kind": "deleted", "data": 2},
		map[string]any{"kind": "created", "data": "N"},
		map[string]any{"kind": "first", "code": 1},
		map[string]any{"kind": "created", "data": map[string]any{"name": "N", "child": map[string]any{"kind": "deleted", "data": 2}}},
		map[string]any{"kind": "created", "data": map[string]any{"name": "N", "child": map[string]any{"kind": "made", "data": map[string]any{"name": "C"}}}},
	}
	for _, name := range []string{"Pet", "DuplicatePet", "Parent", "Child", "Event", "InlinePet", "ImplicitPet", "QueryEvent"} {
		var expected []bool
		for _, spec := range []map[string]any{original, output} {
			compiled, err := gojsonschema.NewSchema(gojsonschema.NewGoLoader(map[string]any{"$ref": "#/components/schemas/" + name, "components": spec["components"]}))
			require.NoError(t, err)
			actual := make([]bool, len(values))
			for i, value := range values {
				result, err := compiled.Validate(gojsonschema.NewGoLoader(value))
				require.NoError(t, err)
				actual[i] = result.Valid()
			}
			if expected == nil {
				expected = actual
			} else {
				require.Equal(t, expected, actual, name)
			}
		}
	}
}

func TestDiscriminatorImportedMediaRoundtrip(t *testing.T) {
	source := readFixture(t, "discriminator/media.yaml")
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(design), `Meta("oneof:value:field", "data")`)
	require.NotContains(t, string(design), `var ImportedCreated =`)
	directory := requireRenderedDesignGenerates(t, design)
	requireDiscriminatorSpecifications(t, directory)
	output := readGeneratedOpenAPIContract(t, directory)
	components := output["components"].(map[string]any)["schemas"].(map[string]any)
	event := components["Event"].(map[string]any)
	require.Equal(t, "#/components/schemas/Created", event["discriminator"].(map[string]any)["defaultMapping"])
	require.Contains(t, components, "Created", "default target remains emitted by the envelope owner")
	for _, path := range []string{"/form", "/multipart"} {
		mediaName := "application/x-www-form-urlencoded"
		if path == "/multipart" {
			mediaName = "multipart/form-data"
		}
		operation := operationFromImportedSpec(t, output, path, "post")
		body := operation["requestBody"].(map[string]any)
		if ref, ok := body["$ref"].(string); ok {
			const prefix = "#/components/requestBodies/"
			require.True(t, strings.HasPrefix(ref, prefix))
			body = output["components"].(map[string]any)["requestBodies"].(map[string]any)[strings.TrimPrefix(ref, prefix)].(map[string]any)
		}
		require.NotEqual(t, true, body["required"], "the fixture's optional request body must remain optional")
		media := body["content"].(map[string]any)[mediaName].(map[string]any)
		encoding := media["encoding"].(map[string]any)
		require.Equal(t, "application/json", encoding["event"].(map[string]any)["contentType"])
		if path == "/form" {
			require.Equal(t, "deepObject", encoding["details"].(map[string]any)["style"])
		}
	}
	query := operationFromImportedSpec(t, output, "/query-json", "get")
	parameters := query["parameters"].([]any)
	require.Len(t, parameters, 3)
	for _, value := range parameters {
		parameter := value.(map[string]any)
		require.Contains(t, parameter["content"].(map[string]any), "application/json")
		require.NotContains(t, parameter, "style")
		require.NotContains(t, parameter, "explode")
	}
	delete(output, "jsonSchemaDialect")
	delete(output, "servers")
	roundtrip, err := json.Marshal(output, json.Deterministic(true))
	require.NoError(t, err)
	regenerated, diagnostics, err := Analyze(roundtrip)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	second, err := Render(regenerated, Options{PackageName: "design"})
	require.NoError(t, err)
	requireRenderedDesignEvaluates(t, second, 6)
	requireIndependentRegeneration(t, directory)
}

func TestDiscriminatorImportedOptional31(t *testing.T) {
	source := strings.Replace(string(readFixture(t, "discriminator/roundtrip.yaml")), "openapi: 3.2.0", "openapi: 3.1.1", 1)
	lines := strings.Split(source, "\n")
	lines = slices.DeleteFunc(lines, func(line string) bool {
		return strings.Contains(line, "defaultMapping:")
	})
	document, diagnostics, err := Analyze([]byte(strings.Join(lines, "\n")))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	directory := requireRenderedDesignGenerates(t, design)
	requireDiscriminatorSpecifications(t, directory)
	output := readGeneratedOpenAPIContract(t, directory)
	require.Equal(t, "3.1.1", output["openapi"])
	schemas := output["components"].(map[string]any)["schemas"].(map[string]any)
	pet := schemas["Pet"].(map[string]any)
	require.NotContains(t, pet, "required", "hint projection must not narrow an optional tag")
	require.NotContains(t, pet["discriminator"].(map[string]any), "defaultMapping")
	requireIndependentRegeneration(t, directory)
}

func discriminatorClosureValues(t *testing.T, source []byte, output map[string]any) {
	t.Helper()
	var original map[string]any
	require.NoError(t, yaml.Unmarshal(source, &original))
	for _, spec := range []map[string]any{original, output} {
		schema := spec["components"].(map[string]any)["schemas"].(map[string]any)["Event"].(map[string]any)
		for _, tc := range []struct {
			wire  string
			valid bool
		}{
			{`{"kind":"created","data":{"name":"N"}}`, true},
			{`{"kind":"created","data":{"name":"N"},"extra":1}`, false},
			{`{"kind":"created","data":{}}`, false},
			{`{"kind":"made","data":{"name":"N"}}`, false},
			{`{"data":{"name":"N"}}`, false},
		} {
			var value any
			require.NoError(t, json.Unmarshal([]byte(tc.wire), &value))
			valid, _ := evaluateOneOfClosureSchema(t, spec, schema, value)
			require.Equal(t, tc.valid, valid, tc.wire)
		}
	}
}

func discriminatorHintValues(t *testing.T, source []byte, output map[string]any) {
	t.Helper()
	var original map[string]any
	require.NoError(t, yaml.Unmarshal(source, &original))
	sourceSchemas := original["components"].(map[string]any)["schemas"].(map[string]any)
	generatedSchemas := output["components"].(map[string]any)["schemas"].(map[string]any)
	for _, name := range []string{"Event", "Pet", "DuplicatePet", "Parent"} {
		expected := sourceSchemas[name].(map[string]any)["discriminator"].(map[string]any)
		actual := generatedSchemas[name].(map[string]any)["discriminator"].(map[string]any)
		mappings := expected["mapping"].(map[string]any)
		for key, value := range mappings {
			if !strings.Contains(value.(string), ":") && !strings.HasPrefix(value.(string), "#") {
				mappings[key] = "#/components/schemas/" + value.(string)
			}
		}
		if value, ok := expected["defaultMapping"].(string); ok && !strings.Contains(value, ":") && !strings.HasPrefix(value, "#") {
			expected["defaultMapping"] = "#/components/schemas/" + value
		}
		want, err := json.Marshal(expected, json.Deterministic(true))
		require.NoError(t, err)
		got, err := json.Marshal(actual, json.Deterministic(true))
		require.NoError(t, err)
		require.Equal(t, string(want), string(got), "complete hint contract %s", name)
	}
}

func requireDiscriminatorSpecifications(t *testing.T, directory string) {
	t.Helper()
	for _, name := range []string{"openapi.json", "openapi.yaml"} {
		data, err := os.ReadFile(filepath.Join(directory, "gen", "http", name))
		require.NoError(t, err)
		parsed, err := libopenapi.NewDocument(data)
		require.NoError(t, err)
		_, err = parsed.BuildV3Model()
		require.NoError(t, err)
	}
}

func TestDiscriminatorPartialParentRoundtrip(t *testing.T) {
	source := readFixture(t, "discriminator/partial_parent.yaml")
	partial, _, err := AnalyzePartial(Source{Bytes: source}, Selection{}, false)
	require.NoError(t, err)
	require.Empty(t, partial.Skipped)
	require.Len(t, partial.OperationOmissions, 1)
	requireDiagnosticCode(t, partial.OperationOmissions[0].Diagnostics, "schema-discriminator-omission")
	design, err := Render(partial.Document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.NotContains(t, string(design), "openapi:discriminator")
	directory := requireRenderedDesignGenerates(t, design)
	requireDiscriminatorSpecifications(t, directory)
	output := readGeneratedOpenAPIContract(t, directory)
	components := output["components"].(map[string]any)["schemas"].(map[string]any)
	require.NotContains(t, components, "UnsupportedChild")
	parent := components["Parent"].(map[string]any)
	require.NotContains(t, parent, "discriminator")
	require.Equal(t, []any{"kind"}, parent["required"])
	var original map[string]any
	require.NoError(t, yaml.Unmarshal(source, &original))
	for _, value := range []any{nil, map[string]any{}, map[string]any{"kind": "other"}, map[string]any{"kind": 1}, map[string]any{"kind": "child", "extra": true}} {
		membership := make([]bool, 0, 2)
		for _, contract := range []map[string]any{original, output} {
			compiled, err := gojsonschema.NewSchema(gojsonschema.NewGoLoader(map[string]any{"$ref": "#/components/schemas/Parent", "components": contract["components"]}))
			require.NoError(t, err)
			result, err := compiled.Validate(gojsonschema.NewGoLoader(value))
			require.NoError(t, err)
			membership = append(membership, result.Valid())
		}
		require.Equal(t, membership[0], membership[1], "%v", value)
	}
	requireIndependentRegeneration(t, directory)
}

func TestDiscriminatorImplicitParentRoundtrip(t *testing.T) {
	source := Source{Bytes: readFixture(t, "discriminator/roundtrip.yaml")}
	document, diagnostics, _, err := AnalyzeSelected(source, Selection{Paths: []string{"/implicit-parent"}})
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.ElementsMatch(t, []string{"ImplicitParent", "ImplicitChild"}, discriminatorSelectionNames(document))
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	directory := requireRenderedDesignGenerates(t, design)
	requireDiscriminatorSpecifications(t, directory)
	output := readGeneratedOpenAPIContract(t, directory)
	components := output["components"].(map[string]any)["schemas"].(map[string]any)
	require.Equal(t, map[string]any{"propertyName": "kind"}, components["ImplicitParent"].(map[string]any)["discriminator"])
	require.Contains(t, components, "ImplicitChild", "retained implicit child declaration must be emitted")
	child := components["ImplicitChild"].(map[string]any)
	require.Equal(t, "#/components/schemas/ImplicitParent", child["allOf"].([]any)[0].(map[string]any)["$ref"])
	requireIndependentRegeneration(t, directory)
}
