package ir

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponseKeysPreserveReferenceSiblingSemantics(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Schema)
	}{
		{"example", func(s *Schema) {
			s.Example = "different"
		}},
		{"default", func(s *Schema) {
			s.DefaultValue = "different"
		}},
		{"not", func(s *Schema) {
			s.Not = &Schema{}
		}},
		{"constraint", func(s *Schema) {
			minimum := 2
			s.MinLength = &minimum
		}},
		{"description", func(s *Schema) {
			s.Description = "different"
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			schemas := map[string]*Schema{"Shared": {Type: "string"}}
			left := responseSchemaValue(&Schema{Ref: "#/components/schemas/Shared"})
			right := responseSchemaValue(&Schema{Ref: "#/components/schemas/Shared"})
			test.change(right.Value.Content["application/json"].Schema)
			a, err := responseKeys(left, "200", schemas)
			require.NoError(t, err)
			b, err := responseKeys(right, "200", schemas)
			require.NoError(t, err)
			require.Equal(t, a.allocation, b.allocation, "historical naming erased these siblings")
			require.NotEqual(t, a.semantic, b.semantic, "naming identity must not permit semantic sharing")
			paths := responsePaths(left, right, false)
			components := componentizeResponses(paths, schemas)
			require.Len(t, components, 2)
			require.NotEqual(t, left.Ref, right.Ref)
		})
	}
}

func TestResponsePreferredBasesUseHistoricalIdentityOnly(t *testing.T) {
	schemas := map[string]*Schema{
		"Leaf":     {Type: "string"},
		"Shared":   {Type: "object", Properties: map[string]*Schema{"value": {Ref: "#/components/schemas/Leaf", Example: "first"}}},
		"Shared_2": {Type: "object", Properties: map[string]*Schema{"value": {Ref: "#/components/schemas/Leaf", Example: "second"}}},
	}
	media := &MediaType{Schema: &Schema{Ref: "#/components/schemas/Shared_2"}}
	require.Equal(t, "SharedStatus200Response", reusableResponseComponentBase(&ResponseRef{Value: &Response{Content: map[string]*MediaType{"application/json": media}}}, "200", schemas))
	require.Equal(t, "SharedRequestBody", reusableRequestBodyComponentBase(&RequestBodyRef{Value: &RequestBody{Content: map[string]*MediaType{"application/json": media}}}, schemas))
	a, err := responseKeys(responseSchemaValue(&Schema{Ref: "#/components/schemas/Shared"}), "200", schemas)
	require.NoError(t, err)
	b, err := responseKeys(responseSchemaValue(&Schema{Ref: "#/components/schemas/Shared_2"}), "200", schemas)
	require.NoError(t, err)
	require.Equal(t, a.allocation, b.allocation)
	require.NotEqual(t, a.semantic, b.semantic)
}

func TestResponseKeysKeepRecursivePurposeCachesIsolated(t *testing.T) {
	input := responseSchemaValue(&Schema{Discriminator: &Discriminator{Mapping: map[string]string{"root": "#/components/schemas/Recursive"}}})
	keys := make([]responseIdentity, 0, 2)
	for _, example := range []string{"first", "second"} {
		schemas := map[string]*Schema{
			"Leaf": {Type: "string"},
			"Recursive": {
				Type:          "object",
				Properties:    map[string]*Schema{"value": {Ref: "#/components/schemas/Leaf", Example: example}},
				Discriminator: &Discriminator{Mapping: map[string]string{"again": "#/components/schemas/Recursive"}},
			},
		}
		first, err := responseKeys(input, "200", schemas)
		require.NoError(t, err)
		repeat, err := responseKeys(input, "200", schemas)
		require.NoError(t, err)
		require.Equal(t, first, repeat)
		require.Equal(t, example, schemas["Recursive"].Properties["value"].Example)
		keys = append(keys, first)
	}
	require.Equal(t, keys[0].allocation, keys[1].allocation)
	require.NotEqual(t, keys[0].semantic, keys[1].semantic)
}

func TestResponseAllocationIsIndependentOfMapInsertion(t *testing.T) {
	build := func(reverse bool) (map[string]*ResponseRef, map[string]*PathItem) {
		a := responseSchemaValue(&Schema{Ref: "#/components/schemas/Shared", Example: "first"})
		b := responseSchemaValue(&Schema{Ref: "#/components/schemas/Shared", Example: "second"})
		paths := responsePaths(a, b, reverse)
		return componentizeResponses(paths, map[string]*Schema{"Shared": {Type: "string"}}), paths
	}
	a, pathsA := build(false)
	b, pathsB := build(true)
	require.Equal(t, a, b)
	require.Equal(t, pathsA, pathsB)
}

func responseSchemaValue(schema *Schema) *ResponseRef {
	return &ResponseRef{Value: &Response{ComponentName: "Named", Description: "OK", Content: map[string]*MediaType{"application/json": {Schema: schema}}}}
}

func responsePaths(left, right *ResponseRef, reverse bool) map[string]*PathItem {
	paths := make(map[string]*PathItem)
	add := func(path string, value *ResponseRef) {
		paths[path] = &PathItem{Operations: map[string]*Operation{"GET": {Responses: map[string]*ResponseRef{"200": value}}}}
	}
	if reverse {
		add("/right", right)
		add("/left", left)
	} else {
		add("/left", left)
		add("/right", right)
	}
	return paths
}
