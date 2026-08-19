package main

import (
	json "encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/openapiimport"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestOpenAPIImportSemanticRoundTrip(t *testing.T) {
	if os.Getenv("LOOM_OPENAPI_CONTRACT") == "" {
		t.Skip("set LOOM_OPENAPI_CONTRACT=1 to run the OpenAPI import contract")
	}

	repoRoot := testingx.RepoRoot()
	loomBin := filepath.Join(t.TempDir(), "loom")
	output, err := testingx.RunCmd(repoRoot, "go", "build", "-o", loomBin, "./cmd/loom")
	require.NoError(t, err, output)
	fixture, err := os.ReadFile(filepath.Join(repoRoot, "internal", "openapiimport", "testdata", "supported.yaml"))
	require.NoError(t, err)

	for _, version := range []string{"3.1.1", "3.2.0"} {
		t.Run(version, func(t *testing.T) {
			source := replaceOpenAPIVersion(t, fixture, version)
			sourceDocument := analyzeOpenAPIContract(t, source)
			moduleDir := t.TempDir()
			modulePath := "example.com/openapi-roundtrip"
			writeRoundTripModule(t, moduleDir, modulePath, repoRoot)
			input := filepath.Join(moduleDir, "openapi.yaml")
			require.NoError(t, os.WriteFile(input, source, 0o600))

			output, err := testingx.RunCmd(moduleDir, loomBin, "import", "openapi", input, "-o", "design")
			require.NoError(t, err, output)
			output, err = testingx.RunCmd(moduleDir, "go", "mod", "tidy")
			require.NoError(t, err, output)
			output, err = testingx.RunCmd(moduleDir, loomBin, "gen", modulePath+"/design", "-o", ".")
			require.NoError(t, err, output)
			output, err = testingx.RunCmd(moduleDir, "go", "mod", "tidy")
			require.NoError(t, err, output)

			for _, path := range []string{
				"gen/http/pet_api/client/client.go",
				"gen/http/pet_api/server/server.go",
				"gen/http/openapi.json",
				"gen/http/openapi.yaml",
			} {
				require.FileExists(t, filepath.Join(moduleDir, path))
			}
			output, err = testingx.RunCmd(moduleDir, "go", "test", "./...")
			require.NoError(t, err, output)

			regenerated, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
			require.NoError(t, err)
			regenerated = removeGeneratedDocumentDefaults(t, regenerated, sourceDocument.Title)
			regeneratedDocument := analyzeOpenAPIContract(t, regenerated)
			require.NoError(t, compareOpenAPIContracts(sourceDocument, regeneratedDocument))
		})
	}
}

func TestOpenAPIImportSharedStatusRoundTrip(t *testing.T) {
	if os.Getenv("LOOM_OPENAPI_CONTRACT") == "" {
		t.Skip("set LOOM_OPENAPI_CONTRACT=1 to run the OpenAPI import contract")
	}

	repoRoot := testingx.RepoRoot()
	loomBin := filepath.Join(t.TempDir(), "loom")
	output, err := testingx.RunCmd(repoRoot, "go", "build", "-o", loomBin, "./cmd/loom")
	require.NoError(t, err, output)
	source, err := os.ReadFile(filepath.Join(
		repoRoot,
		"internal",
		"openapiimport",
		"testdata",
		"shared_status_different_schemas.yaml",
	))
	require.NoError(t, err)
	sourceDocument := analyzeOpenAPIContract(t, source)

	for _, allowLossy := range []bool{false, true} {
		name := "strict"
		if allowLossy {
			name = "lossy"
		}
		t.Run(name, func(t *testing.T) {
			moduleDir := t.TempDir()
			modulePath := "example.com/shared-status-roundtrip"
			writeRoundTripModule(t, moduleDir, modulePath, repoRoot)
			input := filepath.Join(moduleDir, "openapi.yaml")
			require.NoError(t, os.WriteFile(input, source, 0o600))

			args := []string{"import", "openapi", input, "-o", "design"}
			if allowLossy {
				args = append(args, "--allow-lossy")
			}
			output, err := testingx.RunCmd(moduleDir, loomBin, args...)
			require.NoError(t, err, output)
			output, err = testingx.RunCmd(moduleDir, "go", "mod", "tidy")
			require.NoError(t, err, output)
			output, err = testingx.RunCmd(moduleDir, loomBin, "gen", modulePath+"/design", "-o", ".")
			require.NoError(t, err, output)
			output, err = testingx.RunCmd(moduleDir, "go", "mod", "tidy")
			require.NoError(t, err, output)
			output, err = testingx.RunCmd(moduleDir, "go", "test", "./gen/...")
			require.NoError(t, err, output)

			regenerated, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
			require.NoError(t, err)
			regenerated = removeGeneratedDocumentDefaults(t, regenerated, sourceDocument.Title)
			regeneratedDocument := analyzeOpenAPIContract(t, regenerated)
			require.NoError(t, compareOpenAPIContracts(sourceDocument, regeneratedDocument))
		})
	}
}

func TestOpenAPIImportExporterSymmetryRoundTrip(t *testing.T) {
	if os.Getenv("LOOM_OPENAPI_CONTRACT") == "" {
		t.Skip("set LOOM_OPENAPI_CONTRACT=1 to run the OpenAPI import contract")
	}

	repoRoot := testingx.RepoRoot()
	loomBin := filepath.Join(t.TempDir(), "loom")
	output, err := testingx.RunCmd(repoRoot, "go", "build", "-o", loomBin, "./cmd/loom")
	require.NoError(t, err, output)
	source, err := os.ReadFile(filepath.Join(repoRoot, "internal", "openapiimport", "testdata", "symmetry.yaml"))
	require.NoError(t, err)
	sourceDocument := analyzeOpenAPIContract(t, source)

	moduleDir := t.TempDir()
	modulePath := "example.com/openapi-symmetry-roundtrip"
	writeRoundTripModule(t, moduleDir, modulePath, repoRoot)
	input := filepath.Join(moduleDir, "openapi.yaml")
	require.NoError(t, os.WriteFile(input, source, 0o600))

	output, err = testingx.RunCmd(moduleDir, loomBin, "import", "openapi", input, "-o", "design")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(moduleDir, "go", "mod", "tidy")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(moduleDir, loomBin, "gen", modulePath+"/design", "-o", ".")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(moduleDir, "go", "mod", "tidy")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(moduleDir, "go", "test", "./...")
	require.NoError(t, err, output)

	regenerated, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	regenerated = removeGeneratedDocumentDefaults(t, regenerated, sourceDocument.Title)
	regeneratedDocument := analyzeOpenAPIContract(t, regenerated)
	require.NoError(t, compareOpenAPIContracts(sourceDocument, regeneratedDocument))
}

func TestCompareOpenAPIContractsRejectsLoss(t *testing.T) {
	source := &openapiimport.Document{
		OpenAPIVersion: "3.2.0",
		Title:          "Pets",
		Tags:           []string{"pets"},
		Components: openapiimport.Components{Schemas: []openapiimport.NamedSchema{{
			Name: "Pet", GoName: "Pet", Schema: &openapiimport.Schema{
				Type: "object",
				Properties: []openapiimport.NamedProperty{{
					Name: "name", Schema: &openapiimport.Schema{Type: "string"},
				}},
			},
		}}},
		Operations: []openapiimport.Operation{{
			Method: "POST", Path: "/pets", OperationID: "pets.create", Tags: []string{"pets"},
			RequestBody: &openapiimport.RequestBody{Description: "Pet to create.", Required: true},
			Responses: []openapiimport.StatusResponse{{
				Status: "201", Response: openapiimport.Response{Description: "Created pet."},
			}},
		}},
	}

	tests := []struct {
		name string
		edit func(*openapiimport.Document)
	}{
		{name: "missing operation tag", edit: func(document *openapiimport.Document) {
			document.Operations[0].Tags = nil
		}},
		{name: "missing request body description", edit: func(document *openapiimport.Document) {
			document.Operations[0].RequestBody.Description = ""
		}},
		{name: "changed response description", edit: func(document *openapiimport.Document) {
			document.Operations[0].Responses[0].Response.Description = "Status201: Created pet."
		}},
		{name: "extra response status", edit: func(document *openapiimport.Document) {
			document.Operations[0].Responses = append(document.Operations[0].Responses, openapiimport.StatusResponse{Status: "400"})
		}},
		{name: "extra schema property", edit: func(document *openapiimport.Document) {
			document.Components.Schemas[0].Schema.Properties = append(
				document.Components.Schemas[0].Schema.Properties,
				openapiimport.NamedProperty{Name: "id", Schema: &openapiimport.Schema{Type: "integer", Format: "int64"}},
			)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := cloneOpenAPIContract(t, source)
			test.edit(actual)
			require.Error(t, compareOpenAPIContracts(source, actual))
		})
	}
}

func replaceOpenAPIVersion(t *testing.T, source []byte, version string) []byte {
	t.Helper()
	updated := strings.Replace(string(source), "openapi: 3.1.1", "openapi: "+version, 1)
	if version != "3.1.1" {
		require.NotEqual(t, string(source), updated)
	}
	return []byte(updated)
}

func writeRoundTripModule(t *testing.T, moduleDir, modulePath, repoRoot string) {
	t.Helper()
	goMod := fmt.Sprintf(
		"module %s\n\ngo 1.27rc3\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n",
		modulePath,
		repoRoot,
	)
	require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte(goMod), 0o600))
}

func analyzeOpenAPIContract(t *testing.T, source []byte) *openapiimport.Document {
	t.Helper()
	document, diagnostics, err := openapiimport.Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	return document
}

func removeGeneratedDocumentDefaults(t *testing.T, source []byte, title string) []byte {
	t.Helper()
	var document map[string]any
	require.NoError(t, json.Unmarshal(source, &document))
	require.Equal(t, "https://spec.openapis.org/oas/3.1/dialect/base", document["jsonSchemaDialect"])
	delete(document, "jsonSchemaDialect")
	servers, ok := document["servers"].([]any)
	require.True(t, ok)
	wantServer := map[string]any{
		"url":         "http://localhost:80",
		"description": "Default server for " + title,
	}
	if version, ok := document["openapi"].(string); ok && strings.HasPrefix(version, "3.2.") {
		wantServer["name"] = title
	}
	require.Equal(t, []any{wantServer}, servers)
	delete(document, "servers")
	normalized, err := json.Marshal(document)
	require.NoError(t, err)
	return normalized
}

func compareOpenAPIContracts(expected, actual *openapiimport.Document) error {
	expectedNormalized, err := cloneOpenAPIContractValue(expected)
	if err != nil {
		return err
	}
	normalized, err := normalizeGeneratedOpenAPIContract(expected, actual)
	if err != nil {
		return err
	}
	normalizeEmptyOpenAPICollections(expectedNormalized)
	normalizeEmptyOpenAPICollections(normalized)
	expectedJSON, err := json.Marshal(expectedNormalized)
	if err != nil {
		return fmt.Errorf("marshal expected OpenAPI semantic contract: %w", err)
	}
	actualJSON, err := json.Marshal(normalized)
	if err != nil {
		return fmt.Errorf("marshal actual OpenAPI semantic contract: %w", err)
	}
	if string(expectedJSON) != string(actualJSON) {
		return fmt.Errorf("OpenAPI semantic contract differs:\nexpected: %s\nactual:   %s", expectedJSON, actualJSON)
	}
	return nil
}

func normalizeEmptyOpenAPICollections(document *openapiimport.Document) {
	if len(document.Tags) == 0 {
		document.Tags = nil
	} else {
		sort.Strings(document.Tags)
	}
	if len(document.TagMetadata) == 0 {
		document.TagMetadata = nil
	} else {
		sort.Slice(document.TagMetadata, func(i, j int) bool {
			return document.TagMetadata[i].Name < document.TagMetadata[j].Name
		})
	}
	for index := range document.Components.Schemas {
		normalizeOpenAPISchema(document.Components.Schemas[index].Schema)
	}
	if len(document.Components.Schemas) == 0 {
		document.Components.Schemas = nil
	}
	if len(document.Components.Parameters) == 0 {
		document.Components.Parameters = nil
	}
	for index := range document.Components.Parameters {
		normalizeOpenAPIParameter(&document.Components.Parameters[index].Parameter)
	}
	if len(document.Components.RequestBodies) == 0 {
		document.Components.RequestBodies = nil
	}
	for index := range document.Components.RequestBodies {
		normalizeOpenAPIRequestBody(&document.Components.RequestBodies[index].RequestBody)
	}
	if len(document.Components.Responses) == 0 {
		document.Components.Responses = nil
	}
	for index := range document.Components.Responses {
		normalizeOpenAPIResponse(&document.Components.Responses[index].Response)
	}
	if len(document.Components.Headers) == 0 {
		document.Components.Headers = nil
	}
	for index := range document.Components.Headers {
		normalizeOpenAPIHeader(&document.Components.Headers[index].Header)
	}
	if len(document.Operations) == 0 {
		document.Operations = nil
	}
	for index := range document.Operations {
		operation := &document.Operations[index]
		if len(operation.Tags) == 0 {
			operation.Tags = nil
		} else {
			sort.Strings(operation.Tags)
		}
		if len(operation.Parameters) == 0 {
			operation.Parameters = nil
		}
		for parameterIndex := range operation.Parameters {
			normalizeOpenAPIParameter(&operation.Parameters[parameterIndex])
		}
		if operation.RequestBody != nil {
			normalizeOpenAPIRequestBody(operation.RequestBody)
		}
		if len(operation.Responses) == 0 {
			operation.Responses = nil
		}
		for responseIndex := range operation.Responses {
			normalizeOpenAPIResponse(&operation.Responses[responseIndex].Response)
		}
	}
}

func normalizeOpenAPIParameter(parameter *openapiimport.Parameter) {
	normalizeOpenAPISchema(parameter.Schema)
}

func normalizeOpenAPIRequestBody(body *openapiimport.RequestBody) {
	normalizeOpenAPISchema(body.Schema)
}

func normalizeOpenAPIResponse(response *openapiimport.Response) {
	normalizeOpenAPISchema(response.Schema)
	if len(response.Examples) == 0 {
		response.Examples = nil
	} else {
		sort.Slice(response.Examples, func(i, j int) bool {
			return response.Examples[i].Name < response.Examples[j].Name
		})
	}
	if len(response.Headers) == 0 {
		response.Headers = nil
	}
	for index := range response.Headers {
		normalizeOpenAPIHeader(&response.Headers[index].Header)
	}
}

func normalizeOpenAPIHeader(header *openapiimport.Header) {
	normalizeOpenAPISchema(header.Schema)
}

func normalizeOpenAPISchema(schema *openapiimport.Schema) {
	if schema == nil {
		return
	}
	if len(schema.Properties) == 0 {
		schema.Properties = nil
	}
	for index := range schema.Properties {
		normalizeOpenAPISchema(schema.Properties[index].Schema)
	}
	if len(schema.Required) == 0 {
		schema.Required = nil
	}
	if len(schema.Enum) == 0 {
		schema.Enum = nil
	}
	normalizeOpenAPISchema(schema.Items)
	if schema.AdditionalProperties != nil {
		normalizeOpenAPISchema(schema.AdditionalProperties.Schema)
	}
}

func normalizeGeneratedOpenAPIContract(expected, actual *openapiimport.Document) (*openapiimport.Document, error) {
	normalized, err := cloneOpenAPIContractValue(actual)
	if err != nil {
		return nil, err
	}
	authoredTags := make(map[string]struct{})
	for _, operation := range expected.Operations {
		for _, tag := range operation.Tags {
			authoredTags[tag] = struct{}{}
		}
	}
	expectedTags := make(map[string]struct{}, len(expected.Tags))
	for _, tag := range expected.Tags {
		expectedTags[tag] = struct{}{}
	}
	var retained []string
	for _, tag := range normalized.Tags {
		if _, ok := expectedTags[tag]; ok {
			retained = append(retained, tag)
			continue
		}
		if _, ok := authoredTags[tag]; !ok {
			return nil, fmt.Errorf("generated OpenAPI declared unexpected tag %q", tag)
		}
	}
	normalized.Tags = retained
	expectedTagMetadata := make(map[string]struct{}, len(expected.TagMetadata))
	for _, tag := range expected.TagMetadata {
		expectedTagMetadata[tag.Name] = struct{}{}
	}
	retainedMetadata := make([]openapiimport.Tag, 0, len(normalized.TagMetadata))
	for _, tag := range normalized.TagMetadata {
		if _, ok := expectedTagMetadata[tag.Name]; ok {
			retainedMetadata = append(retainedMetadata, tag)
			continue
		}
		if _, ok := authoredTags[tag.Name]; !ok || !emptyOpenAPITagMetadata(tag) {
			return nil, fmt.Errorf("generated OpenAPI declared unexpected tag metadata for %q", tag.Name)
		}
	}
	normalized.TagMetadata = retainedMetadata
	for operationIndex := range normalized.Operations {
		if operationIndex >= len(expected.Operations) {
			break
		}
		actualOperation := &normalized.Operations[operationIndex]
		expectedOperation := expected.Operations[operationIndex]
		if !expectedOperation.SecurityDefined && actualOperation.SecurityDefined &&
			reflect.DeepEqual(actualOperation.Security, normalized.Security) {
			actualOperation.SecurityDefined = false
			actualOperation.Security = nil
		}
		for responseIndex := range actualOperation.Responses {
			if responseIndex >= len(expected.Operations[operationIndex].Responses) {
				break
			}
			actualResponse := &actualOperation.Responses[responseIndex].Response
			expectedResponse := expectedOperation.Responses[responseIndex].Response
			inlineGeneratedResponseSchema(normalized, &expectedResponse, actualResponse)
			actualHeaders := actualResponse.Headers
			expectedHeaders := expectedResponse.Headers
			for headerIndex := range actualHeaders {
				if headerIndex >= len(expectedHeaders) {
					break
				}
				actualHeader := &actualHeaders[headerIndex].Header
				expectedHeader := expectedHeaders[headerIndex].Header
				if actualHeader.Schema != nil && expectedHeader.Schema != nil &&
					expectedHeader.Schema.Description == "" &&
					actualHeader.Schema.Description == actualHeader.Description {
					actualHeader.Schema.Description = ""
				}
			}
		}
		actualParameters := actualOperation.Parameters
		expectedParameters := expectedOperation.Parameters
		for parameterIndex := range actualParameters {
			if parameterIndex >= len(expectedParameters) {
				break
			}
			actualParameter := &actualParameters[parameterIndex]
			expectedParameter := expectedParameters[parameterIndex]
			if actualParameter.Schema != nil && expectedParameter.Schema != nil &&
				expectedParameter.Schema.Description == "" &&
				actualParameter.Schema.Description == actualParameter.Description {
				actualParameter.Schema.Description = ""
			}
		}
	}
	return normalized, nil
}

func emptyOpenAPITagMetadata(tag openapiimport.Tag) bool {
	return tag.Summary == "" && tag.Description == "" && tag.Parent == "" && tag.Kind == "" &&
		tag.ExternalDocsURL == "" && tag.ExternalDocsDescription == "" && len(tag.Extensions) == 0
}

func inlineGeneratedResponseSchema(
	document *openapiimport.Document,
	expected *openapiimport.Response,
	actual *openapiimport.Response,
) {
	if expected.Schema == nil || expected.Schema.Ref != "" || actual.Schema == nil || actual.Schema.Ref == "" {
		return
	}
	const componentPrefix = "#/components/schemas/"
	if !strings.HasPrefix(actual.Schema.Ref, componentPrefix) {
		return
	}
	name := strings.TrimPrefix(actual.Schema.Ref, componentPrefix)
	for index := range document.Components.Schemas {
		component := document.Components.Schemas[index]
		if component.Name != name {
			continue
		}
		actual.Schema = component.Schema
		removeDuplicatedResponseSchemaExamples(expected, actual)
		document.Components.Schemas = append(
			document.Components.Schemas[:index],
			document.Components.Schemas[index+1:]...,
		)
		return
	}
}

func removeDuplicatedResponseSchemaExamples(expected, actual *openapiimport.Response) {
	if actual.Schema == nil || expected.Schema == nil || len(expected.Schema.Examples) > 0 ||
		len(actual.Schema.Examples) == 0 {
		return
	}
	for _, schemaExample := range actual.Schema.Examples {
		matched := false
		for _, responseExample := range actual.Examples {
			if reflect.DeepEqual(schemaExample.Value, responseExample.Value) {
				matched = true
				break
			}
		}
		if !matched {
			return
		}
	}
	actual.Schema.Examples = nil
}

func cloneOpenAPIContract(t *testing.T, source *openapiimport.Document) *openapiimport.Document {
	t.Helper()
	clone, err := cloneOpenAPIContractValue(source)
	require.NoError(t, err)
	return clone
}

func cloneOpenAPIContractValue(source *openapiimport.Document) (*openapiimport.Document, error) {
	encoded, err := json.Marshal(source)
	if err != nil {
		return nil, fmt.Errorf("marshal OpenAPI contract: %w", err)
	}
	var clone openapiimport.Document
	if err := json.Unmarshal(encoded, &clone); err != nil {
		return nil, fmt.Errorf("unmarshal OpenAPI contract: %w", err)
	}
	return &clone, nil
}
