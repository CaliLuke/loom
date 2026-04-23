package expr

import (
	"testing"
)

func TestAttributeExprAllRequired(t *testing.T) {
	cases := map[string]struct {
		typ      DataType
		expected []string
	}{
		"some required": {
			typ: &UserTypeExpr{
				AttributeExpr: &AttributeExpr{
					Validation: &ValidationExpr{
						Required: []string{"foo", "bar"},
					},
				},
			},
			expected: []string{"foo", "bar"},
		},
		"no required": {
			typ:      Boolean,
			expected: nil,
		},
	}

	for k, tc := range cases {
		attribute := AttributeExpr{
			Type: tc.typ,
		}
		if actual := attribute.AllRequired(); len(tc.expected) != len(actual) {
			t.Errorf("%s: expected the number of all required values to match %d got %d ", k, len(tc.expected), len(actual))
		} else {
			for i, v := range actual {
				if v != tc.expected[i] {
					t.Errorf("%s: got %#v, expected %#v at index %d", k, v, tc.expected[i], i)
				}
			}
		}
	}
}

func TestAttributeExprIsRequired(t *testing.T) {
	cases := map[string]struct {
		attName  string
		expected bool
	}{
		"required": {
			attName:  "foo",
			expected: true,
		},
		"not required": {
			attName:  "bar",
			expected: false,
		},
	}

	for k, tc := range cases {
		attribute := AttributeExpr{
			Type: &UserTypeExpr{
				AttributeExpr: &AttributeExpr{
					Validation: &ValidationExpr{
						Required: []string{"foo"},
					},
				},
			},
		}
		if actual := attribute.IsRequired(tc.attName); tc.expected != actual {
			t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
		}
	}
}

func TestAttributeExprIsRequiredNoDefault(t *testing.T) {
	cases := map[string]struct {
		attName  string
		expected bool
	}{
		"required and no default value": {
			attName:  "foo",
			expected: true,
		},
		"required and default value": {
			attName:  "bar",
			expected: false,
		},
		"not required": {
			attName:  "baz",
			expected: false,
		},
	}

	attribute := AttributeExpr{
		Type: &UserTypeExpr{
			AttributeExpr: &AttributeExpr{
				Type: &Object{
					&NamedAttributeExpr{
						Name:      "foo",
						Attribute: &AttributeExpr{},
					},
					&NamedAttributeExpr{
						Name: "bar",
						Attribute: &AttributeExpr{
							DefaultValue: 1,
						},
					},
					&NamedAttributeExpr{
						Name:      "baz",
						Attribute: &AttributeExpr{},
					},
				},
				Validation: &ValidationExpr{
					Required: []string{"foo", "bar"},
				},
			},
		},
	}
	for k, tc := range cases {
		if actual := attribute.IsRequiredNoDefault(tc.attName); tc.expected != actual {
			t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
		}
	}
}

func TestAttributeExprIsPrimitivePointer(t *testing.T) {
	var (
		attributeBoolean = AttributeExpr{
			Type: Boolean,
		}
		attributeObject = AttributeExpr{
			Type: &Object{
				&NamedAttributeExpr{
					Name: "foo",
					Attribute: &AttributeExpr{
						Type: String,
					},
				},
				&NamedAttributeExpr{
					Name: "bar",
					Attribute: &AttributeExpr{
						Type: &Array{
							ElemType: &AttributeExpr{
								Type: String,
							},
						},
					},
				},
				&NamedAttributeExpr{
					Name: "baz",
					Attribute: &AttributeExpr{
						Type: Bytes,
					},
				},
				&NamedAttributeExpr{
					Name: "qux",
					Attribute: &AttributeExpr{
						Type: Any,
					},
				},
				&NamedAttributeExpr{
					Name: "quux",
					Attribute: &AttributeExpr{
						Type: String,
					},
				},
				&NamedAttributeExpr{
					Name: "corge",
					Attribute: &AttributeExpr{
						Type:         String,
						DefaultValue: "default",
					},
				},
			},
			Validation: &ValidationExpr{
				Required: []string{"quux"},
			},
		}
	)
	cases := map[string]struct {
		attribute  AttributeExpr
		attName    string
		useDefault bool
		expected   bool
	}{
		"primitive pointer": {
			attribute:  attributeObject,
			attName:    "foo",
			useDefault: false,
			expected:   true,
		},
		"no attribute": {
			attribute:  attributeObject,
			attName:    "zoo",
			useDefault: false,
			expected:   false,
		},
		"not primitive": {
			attribute:  attributeObject,
			attName:    "bar",
			useDefault: false,
			expected:   false,
		},
		"primitive but bytes": {
			attribute:  attributeObject,
			attName:    "baz",
			useDefault: false,
			expected:   false,
		},
		"primitive but any": {
			attribute:  attributeObject,
			attName:    "qux",
			useDefault: false,
			expected:   false,
		},
		"primitive but required": {
			attribute:  attributeObject,
			attName:    "quux",
			useDefault: false,
			expected:   false,
		},
		"primitive but default value": {
			attribute:  attributeObject,
			attName:    "corge",
			useDefault: true,
			expected:   false,
		},
		"non object": {
			attribute:  attributeBoolean,
			attName:    "",    // should have panicked!
			useDefault: false, // should have panicked!
			expected:   false, // should have panicked!
		},
		// Test that non-primitive types (like maps, user types) return false early
		// This verifies the fix that adds an early return check for !IsPrimitive(att.Type)
		"non-primitive-early-return": {
			attribute: AttributeExpr{
				Type: &Object{
					&NamedAttributeExpr{
						Name: "userType",
						Attribute: &AttributeExpr{
							Type: &UserTypeExpr{
								AttributeExpr: &AttributeExpr{
									Type: &Object{
										&NamedAttributeExpr{
											Name:      "name",
											Attribute: &AttributeExpr{Type: String},
										},
									},
								},
							},
						},
					},
					&NamedAttributeExpr{
						Name: "mapType",
						Attribute: &AttributeExpr{
							Type: &Map{
								KeyType:  &AttributeExpr{Type: String},
								ElemType: &AttributeExpr{Type: String},
							},
						},
					},
				},
			},
			attName:    "userType",
			useDefault: false,
			expected:   false, // Non-primitive types should return false
		},
	}

	for k, tc := range cases {
		func() {
			// panic recover
			defer func() {
				if k != "non object" {
					return
				}

				if recover() == nil {
					t.Errorf("should have panicked!")
				}
			}()

			if actual := tc.attribute.IsPrimitivePointer(tc.attName, tc.useDefault); tc.expected != actual {
				t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
			}
		}()
	}
}
