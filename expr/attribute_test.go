package expr

import (
	"fmt"
	"testing"

	"github.com/CaliLuke/loom/eval"
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

func TestAttributeExprValidate(t *testing.T) {
	var (
		ctx           = "ctx"
		normalizedCtx = ctx + " - "

		validation = &ValidationExpr{
			Required: []string{"foo"},
		}

		metadata = MetaExpr{
			"view": []string{"foo"},
		}

		fieldNotExistType      = &Object{}
		notAResultType         = Boolean
		viewNotDefinedTypeName = "ViewNotDefinedType"

		errAttributeTypeNil      = fmt.Errorf("attribute type is nil")
		errRequiredFieldNotExist = fmt.Errorf(`%srequired field %q does not exist in type %s`, normalizedCtx, "foo", fieldNotExistType.Name())
		errViewButNotAResultType = fmt.Errorf("%s uses view %q but %q is not a result type", normalizedCtx, metadata["view"][0], notAResultType.Name())
		errTypeNotDefineView     = fmt.Errorf("%s: type %q does not define view %q", normalizedCtx, viewNotDefinedTypeName, "foo")
		errConflictingTypes      = fmt.Errorf("type \"%s\" has conflicting packages %s and %s", "SecondType", "types2", "types")
	)
	cases := map[string]struct {
		typ        DataType
		validation *ValidationExpr
		metadata   MetaExpr
		expected   *eval.ValidationErrors
	}{
		"no error": {
			typ:      Boolean,
			expected: &eval.ValidationErrors{},
		},
		"attribute type is nil": {
			typ:      nil,
			expected: &eval.ValidationErrors{Errors: []error{errAttributeTypeNil}},
		},
		"attribute type is nil in the object": {
			typ: &Object{
				&NamedAttributeExpr{
					Name: "foo",
					Attribute: &AttributeExpr{
						Type: nil,
					},
				},
			},
			expected: &eval.ValidationErrors{Errors: []error{errAttributeTypeNil}},
		},
		"attribute type is nil in the array": {
			typ: &Array{
				ElemType: &AttributeExpr{
					Type: nil,
				},
			},
			expected: &eval.ValidationErrors{Errors: []error{errAttributeTypeNil}},
		},
		"required field does not exist": {
			typ: &Object{
				&NamedAttributeExpr{
					Name: "bar",
					Attribute: &AttributeExpr{
						Type: Boolean,
					},
				},
			},
			validation: validation,
			expected:   &eval.ValidationErrors{Errors: []error{errRequiredFieldNotExist}},
		},
		"required field does not exist in the object": {
			typ: &Object{
				&NamedAttributeExpr{
					Name: "bar",
					Attribute: &AttributeExpr{
						Type: &Object{
							&NamedAttributeExpr{
								Name: "baz",
								Attribute: &AttributeExpr{
									Type: Boolean,
								},
							},
						},
					},
				},
			},
			validation: validation,
			expected:   &eval.ValidationErrors{Errors: []error{errRequiredFieldNotExist}},
		},
		"required field does not exist in the array": {
			typ: &Object{
				&NamedAttributeExpr{
					Name: "bar",
					Attribute: &AttributeExpr{
						Type: &Array{
							ElemType: &AttributeExpr{
								Type: Boolean,
							},
						},
					},
				},
			},
			validation: validation,
			expected:   &eval.ValidationErrors{Errors: []error{errRequiredFieldNotExist}},
		},
		"required field exists in extended attribute": {
			typ: &UserTypeExpr{
				TypeName: "Extended2Attr",
				AttributeExpr: &AttributeExpr{
					Type: &Object{
						&NamedAttributeExpr{
							Name: "bar",
							Attribute: &AttributeExpr{
								Type: &Array{
									ElemType: &AttributeExpr{
										Type: Boolean,
									},
								},
							},
						},
					},
					Bases: []DataType{
						&UserTypeExpr{
							TypeName: "Extended1Attr",
							AttributeExpr: &AttributeExpr{
								Type: &Object{
									&NamedAttributeExpr{
										Name: "foobar",
										Attribute: &AttributeExpr{
											Type: &Array{
												ElemType: &AttributeExpr{
													Type: Boolean,
												},
											},
										},
									},
								},
								Bases: []DataType{
									&UserTypeExpr{
										TypeName: "Attr",
										AttributeExpr: &AttributeExpr{
											Type: &Object{
												&NamedAttributeExpr{
													Name: "foo",
													Attribute: &AttributeExpr{
														Type: &Array{
															ElemType: &AttributeExpr{
																Type: Boolean,
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			validation: validation,
			expected:   &eval.ValidationErrors{Errors: []error{}},
		},
		"defines a view but is not a result type": {
			typ:      Boolean,
			metadata: metadata,
			expected: &eval.ValidationErrors{Errors: []error{errViewButNotAResultType}},
		},
		"type does not define view": {
			typ: &ResultTypeExpr{
				UserTypeExpr: &UserTypeExpr{
					TypeName: viewNotDefinedTypeName,
					AttributeExpr: &AttributeExpr{
						Type: Boolean,
					},
				},
				Views: []*ViewExpr{
					{Name: "bar"},
				},
			},
			metadata: metadata,
			expected: &eval.ValidationErrors{Errors: []error{errTypeNotDefineView}},
		},
		"custom package in parent type": {
			typ: &UserTypeExpr{
				TypeName: "FirstType",
				AttributeExpr: &AttributeExpr{
					Meta: MetaExpr{"struct:pkg:path": []string{"types"}},
					Type: &Object{
						&NamedAttributeExpr{
							Name: "thing",
							Attribute: &AttributeExpr{
								Type: &UserTypeExpr{
									AttributeExpr: &AttributeExpr{
										Type: &Object{
											&NamedAttributeExpr{
												Name: "Description",
												Attribute: &AttributeExpr{
													Type: String,
												},
											},
										},
									},
									TypeName: "SecondType",
								},
							},
						},
					},
				},
			},
			expected: &eval.ValidationErrors{Errors: []error{}},
		},
		"custom packages in child type": {
			typ: &UserTypeExpr{
				TypeName: "FirstType",
				AttributeExpr: &AttributeExpr{
					Type: &Object{
						&NamedAttributeExpr{
							Name: "thing",
							Attribute: &AttributeExpr{
								Type: &UserTypeExpr{
									AttributeExpr: &AttributeExpr{
										Meta: MetaExpr{"struct:pkg:path": []string{"types"}},
										Type: &Object{
											&NamedAttributeExpr{
												Name: "Description",
												Attribute: &AttributeExpr{
													Type: String,
												},
											},
										},
									},
									TypeName: "SecondType",
								},
							},
						},
					},
				},
			},
			expected: &eval.ValidationErrors{Errors: []error{}},
		},
		"matching custom packages between sub-type and parent": {
			typ: &UserTypeExpr{
				TypeName: "FirstType",
				AttributeExpr: &AttributeExpr{
					Meta: MetaExpr{"struct:pkg:path": []string{"types"}},
					Type: &Object{
						&NamedAttributeExpr{
							Name: "thing",
							Attribute: &AttributeExpr{
								Type: &UserTypeExpr{
									AttributeExpr: &AttributeExpr{
										Meta: MetaExpr{"struct:pkg:path": []string{"types"}},
										Type: &Object{
											&NamedAttributeExpr{
												Name: "Description",
												Attribute: &AttributeExpr{
													Type: String,
												},
											},
										},
									},
									TypeName: "SecondType",
								},
							},
						},
					},
				},
			},
			expected: &eval.ValidationErrors{Errors: []error{}},
		},
		"conflicting custom packages between sub-type and parent": {
			typ: &UserTypeExpr{
				TypeName: "FirstType",
				AttributeExpr: &AttributeExpr{
					Meta: MetaExpr{"struct:pkg:path": []string{"types"}},
					Type: &Object{
						&NamedAttributeExpr{
							Name: "thing",
							Attribute: &AttributeExpr{
								Type: &UserTypeExpr{
									AttributeExpr: &AttributeExpr{
										Meta: MetaExpr{"struct:pkg:path": []string{"types2"}},
										Type: &Object{
											&NamedAttributeExpr{
												Name: "Description",
												Attribute: &AttributeExpr{
													Type: String,
												},
											},
										},
									},
									TypeName: "SecondType",
								},
							},
						},
					},
				},
			},
			expected: &eval.ValidationErrors{Errors: []error{errConflictingTypes}},
		},
		"conflicting custom packages between sub-type in array and parent": {
			typ: &UserTypeExpr{
				TypeName: "FirstType",
				AttributeExpr: &AttributeExpr{
					Meta: MetaExpr{"struct:pkg:path": []string{"types"}},
					Type: &Object{
						&NamedAttributeExpr{
							Name: "thing",
							Attribute: &AttributeExpr{
								Type: &Array{
									ElemType: &AttributeExpr{
										Type: &UserTypeExpr{
											AttributeExpr: &AttributeExpr{
												Meta: MetaExpr{"struct:pkg:path": []string{"types2"}},
												Type: &Object{
													&NamedAttributeExpr{
														Name: "Description",
														Attribute: &AttributeExpr{
															Type: String,
														},
													},
												},
											},
											TypeName: "SecondType",
										},
									},
								},
							},
						},
					},
				},
			},
			expected: &eval.ValidationErrors{Errors: []error{errConflictingTypes}},
		},
		"conflicting custom packages between sub-type in map key and parent": {
			typ: &UserTypeExpr{
				TypeName: "FirstType",
				AttributeExpr: &AttributeExpr{
					Meta: MetaExpr{"struct:pkg:path": []string{"types"}},
					Type: &Object{
						&NamedAttributeExpr{
							Name: "thing",
							Attribute: &AttributeExpr{
								Type: &Map{
									KeyType: &AttributeExpr{
										Type: &UserTypeExpr{
											AttributeExpr: &AttributeExpr{
												Meta: MetaExpr{"struct:pkg:path": []string{"types2"}},
												Type: &Object{
													&NamedAttributeExpr{
														Name: "Description",
														Attribute: &AttributeExpr{
															Type: String,
														},
													},
												},
											},
											TypeName: "SecondType",
										},
									},
									ElemType: &AttributeExpr{
										Type: String,
									},
								},
							},
						},
					},
				},
			},
			expected: &eval.ValidationErrors{Errors: []error{errConflictingTypes}},
		},
		"conflicting custom packages between sub-type in map element and parent": {
			typ: &UserTypeExpr{
				TypeName: "FirstType",
				AttributeExpr: &AttributeExpr{
					Meta: MetaExpr{"struct:pkg:path": []string{"types"}},
					Type: &Object{
						&NamedAttributeExpr{
							Name: "thing",
							Attribute: &AttributeExpr{
								Type: &Map{
									KeyType: &AttributeExpr{
										Type: String,
									},
									ElemType: &AttributeExpr{
										Type: &UserTypeExpr{
											AttributeExpr: &AttributeExpr{
												Meta: MetaExpr{"struct:pkg:path": []string{"types2"}},
												Type: &Object{
													&NamedAttributeExpr{
														Name: "Description",
														Attribute: &AttributeExpr{
															Type: String,
														},
													},
												},
											},
											TypeName: "SecondType",
										},
									},
								},
							},
						},
					},
				},
			},
			expected: &eval.ValidationErrors{Errors: []error{errConflictingTypes}},
		},
		"array default matches enum value": {
			typ: &Array{
				ElemType: &AttributeExpr{Type: String},
			},
			validation: &ValidationExpr{
				Values: []any{
					[]string{"a", "b"},
				},
			},
			expected: &eval.ValidationErrors{Errors: []error{}},
		},
		"array default does not match enum value": {
			typ: &Array{
				ElemType: &AttributeExpr{Type: String},
			},
			validation: &ValidationExpr{
				Values: []any{
					[]string{"a", "b"},
				},
			},
			expected: &eval.ValidationErrors{
				Errors: []error{
					fmt.Errorf(`%sdefault value %#v is not one of the accepted values: %#v`, normalizedCtx, []string{"b", "c"}, []any{[]string{"a", "b"}}),
				},
			},
		},
	}

	for k, tc := range cases {
		attribute := AttributeExpr{
			Type:       tc.typ,
			Validation: tc.validation,
			Meta:       tc.metadata,
		}
		switch k {
		case "array default matches enum value":
			attribute.DefaultValue = []string{"a", "b"}
		case "array default does not match enum value":
			attribute.DefaultValue = []string{"b", "c"}
		}
		if actual := attribute.Validate(ctx, nil); tc.expected != actual {
			if len(tc.expected.Errors) != len(actual.Errors) {
				t.Errorf("%s: expected the number of error values to match %d got %d ", k, len(tc.expected.Errors), len(actual.Errors))
				if len(actual.Errors) > 0 {
					t.Errorf("%#v", actual.Errors[0])
				}
			} else {
				for i, err := range actual.Errors {
					if err.Error() != tc.expected.Errors[i].Error() {
						t.Errorf("%s: got %#v, expected %#v at index %d", k, err, tc.expected.Errors[i], i)
					}
				}
			}
		}
	}
}
