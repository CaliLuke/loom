package openapi

import (
	"testing"

	"github.com/CaliLuke/loom/expr"
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

	if schema.Discriminator == nil || schema.Discriminator.PropertyName != typeKey {
		t.Fatalf("got discriminator %#v, expected property %q", schema.Discriminator, typeKey)
	}
	if len(schema.OneOf) != 2 {
		t.Fatalf("got %d union branches, expected 2", len(schema.OneOf))
	}
	firstType := schema.OneOf[0].Properties[typeKey]
	if firstType.Enum[0] != "single" {
		t.Errorf("got first union enum %#v, expected %q", firstType.Enum, "single")
	}
	secondType := schema.OneOf[1].Properties[typeKey]
	if secondType.Enum[0] != "batch" {
		t.Errorf("got second union enum %#v, expected %q", secondType.Enum, "batch")
	}
	if schema.OneOf[0].Properties[valueKey] == nil || schema.OneOf[1].Properties[valueKey] == nil {
		t.Fatalf("expected value schemas on all union branches")
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

func TestAttributeSchemaUsesSemanticNullability(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: expr.String, Nullable: true}
	schema := buildAttributeSchema(&expr.APIExpr{ExampleGenerator: expr.NewRandom("nullable")}, NewSchema(), attribute)
	if len(schema.AnyOf) != 2 || schema.AnyOf[0].Type != String || schema.AnyOf[1].Type != Null {
		t.Fatalf("got nullable schema %#v, expected string-or-null anyOf", schema.AnyOf)
	}
}
