package openapiimport

import (
	"bytes"
	"cmp"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/httpencoding"
)

func TestImporterBranchShape(t *testing.T) {
	plain := &Schema{Type: "string"}
	integer := &Schema{Type: "integer"}
	unreserved := httpencoding.ValueDomain{Unreserved: true}
	typed := &Schema{Type: "object", Properties: []NamedProperty{{Name: "field", Schema: integer}}, AdditionalProperties: &AdditionalProperties{Schema: plain}}
	no := false
	planner := documentPlanner{schemas: map[string]NamedSchema{"Alias": {Name: "Alias", Schema: plain}}}
	for _, tc := range []struct {
		name   string
		schema *Schema
		want   httpencoding.BranchShape
	}{
		{"typed object", typed, httpencoding.BranchShape{Shape: httpencoding.UnsupportedShape}},
		{"inherited typed object", &Schema{Type: "object", Bases: []*Schema{typed}}, httpencoding.BranchShape{Shape: httpencoding.UnsupportedShape}},
		{"string", plain, httpencoding.BranchShape{Shape: httpencoding.Scalar}},
		{"bytes", &Schema{Type: "string", Format: "byte"}, httpencoding.BranchShape{Shape: httpencoding.Scalar}},
		{"binary", &Schema{Type: "string", Format: "binary"}, httpencoding.BranchShape{Shape: httpencoding.Scalar, Binary: true}},
		{"binary array", &Schema{Type: "array", Items: &Schema{Type: "string", Format: "binary"}}, httpencoding.BranchShape{Shape: httpencoding.Sequence, Items: &httpencoding.BranchShape{Shape: httpencoding.Scalar, Binary: true}}},
		{"integer", integer, httpencoding.BranchShape{Shape: httpencoding.Scalar, Domain: unreserved}},
		{"alias", &Schema{Ref: "#/components/schemas/Alias"}, httpencoding.BranchShape{Shape: httpencoding.Scalar}},
		{"scalar array", &Schema{Type: "array", Items: integer}, httpencoding.BranchShape{Shape: httpencoding.Sequence, Domain: unreserved, Items: &httpencoding.BranchShape{Shape: httpencoding.Scalar, Domain: unreserved}}},
		{"object array", &Schema{Type: "array", Items: &Schema{Type: "object"}}, httpencoding.BranchShape{Shape: httpencoding.UnsupportedShape, Items: &httpencoding.BranchShape{Shape: httpencoding.Nested}}},
		{"fixed object", &Schema{Type: "object", Properties: []NamedProperty{{Name: "wire", Schema: integer}}, Required: []string{"wire"}, AdditionalProperties: &AdditionalProperties{Allowed: &no}}, httpencoding.BranchShape{Shape: httpencoding.Object, Properties: []httpencoding.PropertyFact{{WireName: "wire", Required: true, Kind: httpencoding.Integer, Domain: unreserved}}}},
		{"scalar map", &Schema{Type: "object", AdditionalProperties: &AdditionalProperties{Schema: integer}}, httpencoding.BranchShape{Shape: httpencoding.ScalarMap, Domain: unreserved}},
		{"nested map", &Schema{Type: "object", AdditionalProperties: &AdditionalProperties{Schema: &Schema{Type: "array", Items: plain}}}, httpencoding.BranchShape{Shape: httpencoding.Nested}},
		{"uuid", &Schema{Type: "string", Format: "uuid"}, httpencoding.BranchShape{Shape: httpencoding.Scalar, Domain: unreserved}},
		{"date", &Schema{Type: "string", Format: "date"}, httpencoding.BranchShape{Shape: httpencoding.Scalar, Domain: unreserved}},
		{"plain enum", &Schema{Type: "string", Enum: []any{"a-b", "x_y"}}, httpencoding.BranchShape{Shape: httpencoding.Scalar, Domain: unreserved}},
		{"reserved enum", &Schema{Type: "string", Enum: []any{"a/b"}}, httpencoding.BranchShape{Shape: httpencoding.Scalar}},
		{"untagged union", &Schema{OneOf: []*Schema{plain, integer}}, httpencoding.BranchShape{Shape: httpencoding.Union, Branches: []httpencoding.BranchShape{{Shape: httpencoding.Scalar}, {Shape: httpencoding.Scalar, Domain: unreserved}}}},
		{"nullable", &Schema{Type: "string", Nullable: true}, httpencoding.BranchShape{Shape: httpencoding.Scalar, Nullable: true}},
		{"any", &Schema{Unconstrained: true}, httpencoding.BranchShape{Shape: httpencoding.AnyValue}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, planner.urlShape(tc.schema))
		})
	}
}

func TestParameterShapeAdaptersAgree(t *testing.T) {
	root := codegen.RunDSL(t, testdata.ParameterStylesDSL)
	files, err := openapiv3.Files(root)
	require.NoError(t, err)
	var raw bytes.Buffer
	for _, file := range files {
		if filepath.Ext(file.Path) == ".json" {
			for _, section := range file.AllSections() {
				require.NoError(t, section.Write(&raw))
			}
		}
	}
	// Shape agreement is independent of importer admission of example metadata.
	doc, _, err := Analyze(raw.Bytes())
	require.NoError(t, err)
	require.NotNil(t, doc)
	doc, references := resolveReusableDocument(doc)
	require.Empty(t, references)
	planner := documentPlanner{document: doc, schemas: make(map[string]NamedSchema)}
	planner.index()
	operations := make(map[string]Operation)
	for _, operation := range doc.Operations {
		operations[operation.Path] = operation
	}
	compare := func(name string, attribute *expr.AttributeExpr, schema *Schema) {
		t.Helper()
		expected, actual := expr.HTTPValueShape(attribute), planner.urlShape(schema)
		// Property declaration order is not part of the wire domain.
		slices.SortFunc(expected.Properties, func(a, b httpencoding.PropertyFact) int { return cmp.Compare(a.WireName, b.WireName) })
		slices.SortFunc(actual.Properties, func(a, b httpencoding.PropertyFact) int { return cmp.Compare(a.WireName, b.WireName) })
		require.Equal(t, expected, actual, name)
	}
	for _, endpoint := range root.API.HTTP.Services[0].HTTPEndpoints {
		operation, ok := operations[endpoint.Routes[0].FullPaths()[0]]
		require.True(t, ok, endpoint.Name())
		for _, group := range []struct {
			location   string
			attributes *expr.MappedAttributeExpr
		}{
			{"path", endpoint.PathParams()}, {"query", endpoint.QueryParams()}, {"header", endpoint.Headers}, {"cookie", endpoint.Cookies},
		} {
			if group.attributes == nil || group.attributes.IsEmpty() {
				continue
			}
			for _, field := range *expr.AsObject(group.attributes.Type) {
				wire := group.attributes.ElemName(field.Name)
				found := false
				for _, source := range operation.Parameters {
					parameter, _, err := resolveParameterReference(source, doc.Components)
					require.NoError(t, err)
					if parameter.In == group.location && parameter.Name == wire {
						compare(endpoint.Name()+"/"+wire, field.Attribute, parameter.Schema)
						found = true
					}
				}
				require.True(t, found, endpoint.Name()+"/"+wire)
			}
		}
		if endpoint.FormRequest {
			require.NotNil(t, operation.RequestBody)
			properties := make(map[string]*Schema)
			planner.formProperties(operation.RequestBody.Schema, properties, make(map[*Schema]bool))
			for _, field := range *expr.AsObject(expr.UnwrapInlineHTTPBody(endpoint.Body).Type) {
				name := expr.ElementName(field.Name)
				require.Contains(t, properties, name)
				compare("form/"+name, field.Attribute, properties[name])
			}
		}
		for _, response := range endpoint.Responses {
			if response.Headers == nil || response.Headers.IsEmpty() {
				continue
			}
			for _, field := range *expr.AsObject(response.Headers.Type) {
				name := response.Headers.ElemName(field.Name)
				found := false
				for _, source := range operation.Responses {
					for _, header := range source.Response.Headers {
						if strings.EqualFold(header.Name, name) {
							compare("response/"+name, field.Attribute, header.Header.Schema)
							found = true
						}
					}
				}
				require.True(t, found, name)
			}
		}
	}
}
