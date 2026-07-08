package expr

import (
	"fmt"
	"testing"

	"github.com/CaliLuke/loom/eval"
)

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

func TestAttributeExprValidateRejectsFractionalIntegerBounds(t *testing.T) {
	cases := map[string]struct {
		kind       DataType
		validation func(*float64, *float64) *ValidationExpr
	}{
		"int minimum maximum": {
			kind: Int,
			validation: func(minimum, maximum *float64) *ValidationExpr {
				return &ValidationExpr{Minimum: minimum, Maximum: maximum}
			},
		},
		"uint exclusive minimum maximum": {
			kind: UInt,
			validation: func(minimum, maximum *float64) *ValidationExpr {
				return &ValidationExpr{ExclusiveMinimum: minimum, ExclusiveMaximum: maximum}
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			minimum := 0.2
			maximum := 0.7
			attribute := AttributeExpr{
				Type:       tc.kind,
				Validation: tc.validation(&minimum, &maximum),
			}

			actual := attribute.Validate("ctx", nil)
			if actual == nil || len(actual.Errors) == 0 {
				t.Fatalf("expected fractional integer bounds to fail validation")
			}
			if got := actual.Errors[0].Error(); got != "ctx - integer bounds must be whole numbers" {
				t.Fatalf("expected integer bounds error, got %q", got)
			}
		})
	}
}

func TestAttributeExprValidateAllowsFractionalFloatBounds(t *testing.T) {
	minimum := 0.2
	maximum := 0.7
	attribute := AttributeExpr{
		Type: Float64,
		Validation: &ValidationExpr{
			Minimum: &minimum,
			Maximum: &maximum,
		},
	}

	actual := attribute.Validate("ctx", nil)
	if actual != nil && len(actual.Errors) > 0 {
		t.Fatalf("expected fractional float bounds to pass validation, got %v", actual.Errors)
	}
}
