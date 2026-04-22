package expr

import (
	"testing"
)

func TestTaggedAttribute(t *testing.T) {
	tagged := &AttributeExpr{
		Type: &Object{
			&NamedAttributeExpr{
				Name: "Foo",
				Attribute: &AttributeExpr{
					Meta: MetaExpr{"foo": []string{"foo"}},
				},
			},
		},
	}
	cases := map[string]struct {
		a        *AttributeExpr
		expected string
	}{
		"tagged attribute": {
			a:        tagged,
			expected: "Foo",
		},
		"not object": {
			a: &AttributeExpr{
				Type: Boolean,
			},
			expected: "",
		},
		"no meta": {
			a: &AttributeExpr{
				Type: &Object{
					&NamedAttributeExpr{
						Name:      "Foo",
						Attribute: &AttributeExpr{},
					},
				},
			},
			expected: "",
		},
		"extended": {
			a: &AttributeExpr{
				Type: &Object{
					&NamedAttributeExpr{
						Name:      "bar",
						Attribute: &AttributeExpr{Type: String},
					},
				},
				Bases: []DataType{
					&UserTypeExpr{
						TypeName:      "Extended",
						AttributeExpr: tagged,
					},
				},
			},
			expected: "Foo",
		},
		"recursively extended": {
			a: &AttributeExpr{
				Type: &Object{
					&NamedAttributeExpr{
						Name:      "bar",
						Attribute: &AttributeExpr{Type: String},
					},
				},
				Bases: []DataType{
					&UserTypeExpr{
						TypeName: "Extended",
						AttributeExpr: &AttributeExpr{
							Type: &Object{
								&NamedAttributeExpr{
									Name: "foobar",
									Attribute: &AttributeExpr{
										Meta: MetaExpr{"foobar": []string{"foobar"}},
									},
								},
							},
							Bases: []DataType{
								&UserTypeExpr{
									TypeName:      "AnotherExtended",
									AttributeExpr: tagged,
								},
							},
						},
					},
				},
			},
			expected: "Foo",
		},
	}

	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			if actual := TaggedAttribute(tc.a, "foo"); tc.expected != actual {
				t.Errorf("got %#v, expected %#v", actual, tc.expected)
			}
		})
	}
}

func TestExtractUserExamplesFromExtendedType(t *testing.T) {
	base := &UserTypeExpr{
		TypeName: "BaseRequest",
		AttributeExpr: &AttributeExpr{
			Type: &Object{
				&NamedAttributeExpr{
					Name:      "query",
					Attribute: &AttributeExpr{Type: String},
				},
			},
			UserExamples: []*ExampleExpr{
				{Summary: "simple", Value: map[string]any{"query": "soup"}},
				{Summary: "advanced", Value: map[string]any{"query": "stew"}},
			},
		},
	}

	wrapper := &AttributeExpr{
		Type: &UserTypeExpr{
			TypeName: "BaseRequestBody",
			AttributeExpr: &AttributeExpr{
				Type: &Object{},
				Bases: []DataType{
					base,
				},
			},
		},
	}

	examples := wrapper.ExtractUserExamples()
	if len(examples) != 2 {
		t.Fatalf("got %d examples, expected 2", len(examples))
	}
	if examples[0].Summary != "simple" {
		t.Fatalf("got first example %q, expected simple", examples[0].Summary)
	}
	if examples[1].Summary != "advanced" {
		t.Fatalf("got second example %q, expected advanced", examples[1].Summary)
	}
}

func TestExampleExprMeta(t *testing.T) {
	example := &ExampleExpr{}
	example.AddMeta("openapi:component:example", "ArtifactThreadExample")
	if got := example.Meta["openapi:component:example"]; len(got) != 1 || got[0] != "ArtifactThreadExample" {
		t.Fatalf("got %#v", got)
	}
	example.DeleteMeta("openapi:component:example")
	if _, ok := example.Meta["openapi:component:example"]; ok {
		t.Fatal("expected metadata to be removed")
	}
}
