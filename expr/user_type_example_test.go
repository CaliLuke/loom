package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

// TestUserTypeWithUserExample tests that when an attribute uses a UserType
// and has a user-defined example, the user example is used.
func TestUserTypeWithUserExample(t *testing.T) {
	// Create a UserType "URL" with FormatURI validation
	urlType := &expr.UserTypeExpr{
		AttributeExpr: &expr.AttributeExpr{
			Type: expr.String,
			Validation: &expr.ValidationExpr{
				Format: expr.FormatURI,
			},
		},
		TypeName: "URL",
	}

	// Create an attribute that uses the URL type with a custom example
	customURL := "http://www.seller.topi.eu/offer/739b63c2-9e58-4964-b38d-4ebb424de242%2Fv1"
	attr := &expr.AttributeExpr{
		Type: urlType,
		UserExamples: []*expr.ExampleExpr{
			{
				Summary: "default",
				Value:   customURL,
			},
		},
	}

	// Test with both randomizers to ensure user examples always take precedence
	t.Run("with faker randomizer", func(t *testing.T) {
		exampleGen := expr.NewRandom("test")
		example := attr.Example(exampleGen)
		if example != customURL {
			t.Errorf("Attribute with user example should return %q, got %q", customURL, example)
		}
	})

	t.Run("with deterministic randomizer", func(t *testing.T) {
		gen := expr.NewDeterministicRandomizer()
		exampleGen := &expr.ExampleGenerator{
			Randomizer: gen,
		}
		example := attr.Example(exampleGen)
		if example != customURL {
			t.Errorf("Attribute with user example should return %q, got %q", customURL, example)
		}
	})
}

// TestUserTypeFormatWithCustomExample tests the exact scenario from issue #3716
// where a UserType with FormatURI has a custom example that should be preserved
func TestUserTypeFormatWithCustomExample(t *testing.T) {
	// Create a UserType "URL" with FormatURI validation (no example on the type itself)
	urlType := &expr.UserTypeExpr{
		AttributeExpr: &expr.AttributeExpr{
			Type: expr.String,
			Validation: &expr.ValidationExpr{
				Format: expr.FormatURI,
			},
		},
		TypeName: "URL",
	}

	// Create an attribute using the URL type with a custom example
	// This simulates: Attribute("seller_offer_redirect_url", URL, func() { Example("...") })
	customExample := "http://www.seller.topi.eu/offer/739b63c2-9e58-4964-b38d-4ebb424de242%2Fv1"
	attr := &expr.AttributeExpr{
		Type: urlType,
		UserExamples: []*expr.ExampleExpr{
			{
				Summary: "default",
				Value:   customExample,
			},
		},
	}

	// Test with deterministic randomizer (as reported in the issue)
	gen := expr.NewDeterministicRandomizer()
	exampleGen := &expr.ExampleGenerator{
		Randomizer: gen,
	}

	// The bug was that this would return "https://example.com/foo" instead of the custom example
	example := attr.Example(exampleGen)

	if example != customExample {
		t.Errorf("Custom example should be preserved. Expected %q but got %q", customExample, example)
	}
}

// TestIssue3716Regression tests the specific regression where UserTypes with
// format validation would not generate format-appropriate examples
func TestIssue3716Regression(t *testing.T) {
	// This test captures what happens when we need to generate an example
	// for an object that contains a field with a UserType that has format validation

	// Create the URL UserType
	urlType := &expr.UserTypeExpr{
		AttributeExpr: &expr.AttributeExpr{
			Type: expr.String,
			Validation: &expr.ValidationExpr{
				Format: expr.FormatURI,
			},
		},
		TypeName: "URL",
	}

	// Create an object with a field that uses the URL type but no explicit example
	obj := &expr.Object{
		{"redirect_url", &expr.AttributeExpr{Type: urlType}},
	}

	// When generating an example for the object
	gen := expr.NewDeterministicRandomizer()
	exampleGen := &expr.ExampleGenerator{
		Randomizer: gen,
	}

	example := obj.Example(exampleGen)
	objExample, ok := example.(map[string]any)
	if !ok {
		t.Fatalf("Expected map example, got %T", example)
	}

	// The redirect_url field should have a proper URI example
	redirectURL, ok := objExample["redirect_url"].(string)
	if !ok {
		t.Fatalf("Expected string for redirect_url, got %T", objExample["redirect_url"])
	}

	// With the bug, this would be "abc123" (from String type)
	// With the fix, this should be "https://example.com/foo" (from FormatURI)
	if redirectURL != "https://example.com/foo" {
		t.Errorf("Expected URI example for UserType with FormatURI, got %q", redirectURL)
	}
}

// TestUserTypeWithOwnExample tests that a UserType can have its own example
func TestUserTypeWithOwnExample(t *testing.T) {
	// Create a UserType "URL" with FormatURI validation AND its own example
	customExample := "https://api.example.com/v1/resources"
	urlType := &expr.UserTypeExpr{
		AttributeExpr: &expr.AttributeExpr{
			Type: expr.String,
			Validation: &expr.ValidationExpr{
				Format: expr.FormatURI,
			},
			UserExamples: []*expr.ExampleExpr{
				{
					Summary: "default",
					Value:   customExample,
				},
			},
		},
		TypeName: "URL",
	}

	// Use deterministic randomizer
	gen := expr.NewDeterministicRandomizer()
	exampleGen := &expr.ExampleGenerator{
		Randomizer: gen,
	}

	// The UserType itself should return its custom example
	example := urlType.Example(exampleGen)

	if example != customExample {
		t.Errorf("UserType with custom example should return %q, got %q", customExample, example)
	}
}

// TestRecursiveNamedArrayExample checks that the example of a named array of
// objects that reach the named array through array elements and map values
// has the Go types of those collections at every depth, and that the example
// of an element whose type is still being generated is an empty array rather
// than a nil slice, which would render as null.
func TestRecursiveNamedArrayExample(t *testing.T) {
	node := &expr.UserTypeExpr{TypeName: "Node", AttributeExpr: &expr.AttributeExpr{}}
	nodes := &expr.UserTypeExpr{TypeName: "Nodes", AttributeExpr: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: node}}}}
	node.AttributeExpr.Type = &expr.Object{
		{Name: "grid", Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: nodes}}}},
		{Name: "index", Attribute: &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: nodes}}}},
	}
	placeholders := 0
	for _, seed := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		t.Run(seed, func(t *testing.T) {
			var example any
			require.NotPanics(t, func() {
				example = (&expr.AttributeExpr{Type: nodes}).Example(expr.NewRandom(seed))
			})
			placeholders += requireNodesExample(t, example)
		})
	}
	require.Positive(t, placeholders, "no seed generated the example of an element whose type is still being generated")
}

// requireNodesExample checks that example is a non-nil list of node objects
// whose "grid" and "index" members hold non-nil lists of node objects, and
// returns the number of empty lists it holds.
func requireNodesExample(t *testing.T, example any) int {
	t.Helper()
	list, ok := example.([]map[string]any)
	require.True(t, ok, "nodes example has type %T", example)
	require.NotNil(t, list)
	empty := 0
	if len(list) == 0 {
		empty++
	}
	for _, item := range list {
		if grid, ok := item["grid"]; ok {
			lists, ok := grid.([][]map[string]any)
			require.True(t, ok, "grid example has type %T", grid)
			for _, nested := range lists {
				empty += requireNodesExample(t, nested)
			}
		}
		if index, ok := item["index"]; ok {
			lists, ok := index.(map[string][]map[string]any)
			require.True(t, ok, "index example has type %T", index)
			for _, nested := range lists {
				empty += requireNodesExample(t, nested)
			}
		}
	}
	return empty
}
