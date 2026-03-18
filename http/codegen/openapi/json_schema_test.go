package openapi

import (
	"testing"

	"goa.design/goa/v3/expr"
)

func TestAttributeTypeSchemaUsesTaggedUnionExamplesAndEnums(t *testing.T) {
	union := &expr.Union{
		TypeKey:  "kind",
		ValueKey: "data",
		Values: []*expr.NamedAttributeExpr{
			{
				Name: "Single",
				Attribute: &expr.AttributeExpr{
					Type: &expr.Object{
						{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}},
					},
					Meta: expr.MetaExpr{"oneof:type:tag": []string{"single"}},
				},
			},
			{
				Name: "Batch",
				Attribute: &expr.AttributeExpr{
					Type: &expr.Object{
						{Name: "items", Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}}},
					},
					Meta: expr.MetaExpr{"oneof:type:tag": []string{"batch"}},
				},
			},
		},
	}
	attr := &expr.AttributeExpr{
		Type: union,
		UserExamples: []*expr.ExampleExpr{{
			Summary: "default",
			Value:   map[string]any{"name": "alice"},
		}},
	}

	api := &expr.APIExpr{ExampleGenerator: expr.NewRandom("union")}
	schema := buildAttributeSchema(api, NewSchema(), attr)
	typeKey := union.GetTypeKey()
	valueKey := union.GetValueKey()

	if schema.Properties[typeKey].Enum[0] != "single" || schema.Properties[typeKey].Enum[1] != "batch" {
		t.Errorf("got union enum %#v, expected tagged values", schema.Properties[typeKey].Enum)
	}
	example, ok := schema.Example.(map[string]any)
	if !ok {
		t.Fatalf("expected map example, got %T", schema.Example)
	}
	if example[typeKey] != "single" {
		t.Errorf("got type example %#v, expected %q", example[typeKey], "single")
	}
	if _, ok := example[valueKey].(map[string]any); !ok {
		t.Errorf("got value example %#v, expected nested object", example[valueKey])
	}
}

func TestAttributeTypeSchemaLeavesAmbiguousUnionExampleUnchanged(t *testing.T) {
	union := &expr.Union{
		TypeKey:  "kind",
		ValueKey: "data",
		Values: []*expr.NamedAttributeExpr{
			{
				Name:      "Single",
				Attribute: &expr.AttributeExpr{Type: &expr.Object{{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}}}},
			},
			{
				Name:      "Batch",
				Attribute: &expr.AttributeExpr{Type: &expr.Object{{Name: "items", Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}}}}},
			},
		},
	}
	attr := &expr.AttributeExpr{
		Type: union,
		UserExamples: []*expr.ExampleExpr{{
			Summary: "default",
			Value:   map[string]any{},
		}},
	}

	api := &expr.APIExpr{ExampleGenerator: expr.NewRandom("union")}
	schema := buildAttributeSchema(api, NewSchema(), attr)
	example, ok := schema.Example.(map[string]any)
	if !ok {
		t.Fatalf("expected map example, got %T", schema.Example)
	}
	if len(example) != 0 {
		t.Errorf("got canonicalized ambiguous example %#v, expected unchanged empty object", example)
	}
}
