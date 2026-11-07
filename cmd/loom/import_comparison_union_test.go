package main

import (
	"encoding/json/v2"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/openapiimport"
)

func TestComparisonOneOfOrderAndMultiplicity(t *testing.T) {
	text := func() *openapiimport.Schema {
		return &openapiimport.Schema{Type: "string"}
	}
	number := func() *openapiimport.Schema {
		return &openapiimport.Schema{Type: "integer"}
	}
	nested := func(branches ...*openapiimport.Schema) *openapiimport.Schema {
		return &openapiimport.Schema{OneOf: branches}
	}
	document := func(schema *openapiimport.Schema) *openapiimport.Document {
		return &openapiimport.Document{Components: openapiimport.Components{Schemas: []openapiimport.NamedSchema{{Name: "Choice", GoName: "Choice", Schema: schema}}}}
	}
	for _, tc := range []struct {
		name             string
		expected, actual *openapiimport.Schema
		equal            bool
	}{
		{"order", nested(text(), number()), nested(number(), text()), true},
		{"nested order", nested(nested(text(), number()), text()), nested(text(), nested(number(), text())), true},
		{"duplicate branch retained", nested(text(), text(), number()), nested(text(), number()), false},
		{"changed predicate", nested(text(), number()), nested(text(), &openapiimport.Schema{Type: "boolean"}), false},
		{"closure retained", &openapiimport.Schema{OneOf: []*openapiimport.Schema{text(), number()}, UnionClosed: true}, nested(text(), number()), false},
		{"media pairing retained", &openapiimport.Schema{OneOf: []*openapiimport.Schema{text(), number()}, MediaTypes: []string{"text/plain", "application/json"}}, &openapiimport.Schema{OneOf: []*openapiimport.Schema{number(), text()}, MediaTypes: []string{"text/plain", "application/json"}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := compareOpenAPIContracts(document(tc.expected), document(tc.actual))
			if tc.equal {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

// normalizeComparisonUnionOrder compares oneOf as a multiset: order has no
// semantic meaning, while duplicate alternatives still affect exact-one counts.
// Media unions retain their order because MediaTypes indexes the same branches.
func normalizeComparisonUnionOrder(document *openapiimport.Document) error {
	var schemas []*openapiimport.Schema
	walkComparisonSchemas(reflect.ValueOf(document), func(schema *openapiimport.Schema) {
		schemas = append(schemas, schema)
	})
	for index := len(schemas) - 1; index >= 0; index-- {
		schema := schemas[index]
		if len(schema.OneOf) < 2 || len(schema.MediaTypes) != 0 {
			continue
		}
		type keyedBranch struct {
			key    string
			branch *openapiimport.Schema
		}
		branches := make([]keyedBranch, len(schema.OneOf))
		for index, branch := range schema.OneOf {
			key, err := json.Marshal(branch, json.Deterministic(true))
			if err != nil {
				return fmt.Errorf("canonicalize comparison oneOf: %w", err)
			}
			branches[index] = keyedBranch{key: string(key), branch: branch}
		}
		sort.SliceStable(branches, func(i, j int) bool {
			return branches[i].key < branches[j].key
		})
		for index, branch := range branches {
			schema.OneOf[index] = branch.branch
		}
	}
	return nil
}

func TestComparisonOneOfReferences(t *testing.T) {
	for _, changed := range []bool{false, true} {
		expected := &openapiimport.Document{Components: openapiimport.Components{Schemas: []openapiimport.NamedSchema{
			{Name: "Alias", Schema: &openapiimport.Schema{Type: "string", Enum: []any{"text", "other"}}},
			{Name: "Choice", Schema: &openapiimport.Schema{OneOf: []*openapiimport.Schema{{Ref: "#/components/schemas/Alias"}, {Type: "object", Properties: []openapiimport.NamedProperty{{Name: "flag", Schema: &openapiimport.Schema{Type: "boolean"}}}}}}},
		}}}
		actual := cloneOpenAPIContract(t, expected)
		actual.Components.Schemas[1].Schema.OneOf[0].Type = "string"
		actual.Components.Schemas[1].Schema.OneOf[0].Enum = []any{"text", "other"}
		if changed {
			actual.Components.Schemas[1].Schema.OneOf[0].Enum = []any{"text"}
		}
		inline := actual.Components.Schemas[1].Schema.OneOf[1]
		actual.Components.Schemas = append(actual.Components.Schemas, openapiimport.NamedSchema{Name: "GeneratedFlag", Schema: inline})
		actual.Components.Schemas[1].Schema.OneOf[1] = &openapiimport.Schema{Ref: "#/components/schemas/GeneratedFlag"}
		before := cloneOpenAPIContract(t, actual)
		err := compareOpenAPIContracts(expected, actual)
		if changed {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
		require.Equal(t, before, actual)
	}
}

func TestComparisonNullableTypeSpelling(t *testing.T) {
	for _, tc := range []struct {
		name     string
		union    []string
		nullable bool
		equal    bool
	}{
		{"nullable type spelling", []string{"null", "string"}, true, true},
		{"additional type retained", []string{"null", "string", "integer"}, true, false},
		{"missing null retained", []string{"string"}, true, false},
		{"null not admitted", []string{"string", "null"}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected := &openapiimport.Document{Components: openapiimport.Components{Schemas: []openapiimport.NamedSchema{{Name: "Value", Schema: &openapiimport.Schema{Type: "string", Nullable: tc.nullable, TypeUnion: tc.union}}}}}
			actual := cloneOpenAPIContract(t, expected)
			actual.Components.Schemas[0].Schema.TypeUnion = nil
			err := compareOpenAPIContracts(expected, actual)
			if tc.equal {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestComparisonNestedGeneratedReferencesRejectCyclesAndMissingTargets(t *testing.T) {
	for _, missing := range []bool{false, true} {
		ref := "#/components/schemas/Generated"
		if missing {
			ref = "#/components/schemas/Missing"
		}
		actual := &openapiimport.Document{Components: openapiimport.Components{Schemas: []openapiimport.NamedSchema{{
			Name:   "Generated",
			Schema: &openapiimport.Schema{Type: "object", Properties: []openapiimport.NamedProperty{{Name: "nested", Schema: &openapiimport.Schema{Ref: ref}}}},
		}}}}
		_, err := normalizeGeneratedOpenAPIContract(&openapiimport.Document{}, actual)
		require.Error(t, err)
	}
}
