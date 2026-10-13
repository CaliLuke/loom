package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func TestUnionBranchAdmissionRejectsDuplicateIdentities(t *testing.T) {
	for _, tc := range []struct {
		name string
		dsl  func()
	}{
		{"exact", unionAdmissionDesign("same", "same", "", "", false)},
		{"mapped", unionAdmissionDesign("same:left", "same:right", "", "", false)},
		{"name_with_distinct_tags", unionAdmissionDesign("same", "same", "left", "right", false)},
		{"explicit_tags", unionAdmissionDesign("left", "right", "same", "same", false)},
		{"explicit_and_fallback", unionAdmissionDesign("left", "same", "same", "", false)},
		{"untagged_names", unionAdmissionDesign("same", "same", "left", "right", true)},
		{"untagged_tags", unionAdmissionDesign("left", "right", "same", "same", true)},
		{"inherited_tags", unionInheritedTagDesign(false, false)},
		{"forward_inherited_tags", unionInheritedTagDesign(true, false)},
		{"array_nested", unionNestedDuplicateDesign(false)},
		{"map_nested", unionNestedDuplicateDesign(true)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expr.SetupTestDSL(t)
			require.True(t, eval.Execute(tc.dsl, nil), eval.Context.Error())
			err := eval.RunDSL()
			require.Error(t, err, "duplicate branch identity must fail before generation")
			require.Contains(t, err.Error(), "OneOf")
			require.Contains(t, err.Error(), "branch")
			require.Contains(t, err.Error(), "same")
		})
	}
}

func TestUnionBranchAdmissionPreservesValidIdentities(t *testing.T) {
	for _, tc := range []struct {
		name string
		dsl  func()
	}{
		{"mapped_distinct", unionAdmissionDesign("left:a", "right:b", "", "", false)},
		{"ordinary", unionAdmissionDesign("left", "right", "", "", false)},
		{"case_distinct", unionAdmissionDesign("same", "Same", "", "", false)},
		{"distinct_custom_tags", unionAdmissionDesign("left", "right", "custom-left", "custom-right", false)},
		{"untagged", unionAdmissionDesign("left", "right", "", "", true)},
		{"occurrence_override", unionInheritedTagDesign(false, true)},
		{"forward_occurrence_override", unionInheritedTagDesign(true, true)},
		{"constructor_repeated_type", func() {
			named := dsl.Type("Text", dsl.String)
			dsl.Service("svc", func() {
				dsl.Method("call", func() {
					dsl.Payload(dsl.OneOf(named, named))
				})
			})
		}},
		{"separate_union_scopes", func() {
			left := dsl.Type("Left", func() {
				dsl.OneOf("choice", func() {
					dsl.Attribute("same", dsl.String)
					dsl.Attribute("other", dsl.Int)
				})
			})
			right := dsl.Type("Right", func() {
				dsl.OneOf("choice", func() {
					dsl.Attribute("same", dsl.Int)
					dsl.Attribute("other", dsl.Boolean)
				})
			})
			dsl.Service("svc", func() {
				dsl.Method("call", func() {
					dsl.Payload(left)
					dsl.Result(right)
				})
			})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expr.RunDSL(t, tc.dsl)
		})
	}
}

func unionAdmissionDesign(left, right, leftTag, rightTag string, untagged bool) func() {
	return func() {
		leftType := dsl.Type("Left", func() {
			dsl.Attribute("left", dsl.String)
			dsl.Required("left")
		})
		rightType := dsl.Type("Right", func() {
			dsl.Attribute("right", dsl.Int)
			dsl.Required("right")
		})
		dsl.Service("svc", func() {
			dsl.Method("call", func() {
				dsl.Payload(func() {
					dsl.OneOf("choice", func() {
						if untagged {
							dsl.Untagged()
						}
						dsl.Attribute(left, leftType, func() {
							if leftTag != "" {
								dsl.Meta("oneof:type:tag", leftTag)
							}
						})
						dsl.Attribute(right, rightType, func() {
							if rightTag != "" {
								dsl.Meta("oneof:type:tag", rightTag)
							}
						})
					})
				})
			})
		})
	}
}

func unionInheritedTagDesign(forward, override bool) func() {
	return func() {
		declare := func() {
			dsl.Type("Left", dsl.String, func() {
				dsl.Meta("oneof:type:tag", "same")
			})
			dsl.Type("Right", dsl.Int, func() {
				dsl.Meta("oneof:type:tag", "same")
			})
		}
		if !forward {
			declare()
		}
		owner := dsl.Type("Owner", func() {
			dsl.OneOf("choice", func() {
				dsl.Attribute("left", "Left")
				dsl.Attribute("right", "Right", func() {
					if override {
						dsl.Meta("oneof:type:tag", "distinct")
					}
				})
			})
		})
		if forward {
			declare()
		}
		dsl.Service("svc", func() {
			dsl.Method("call", func() {
				dsl.Payload(owner)
			})
		})
	}
}

func unionNestedDuplicateDesign(mapping bool) func() {
	return func() {
		nested := dsl.Type("Nested", func() {
			dsl.OneOf("choice", func() {
				dsl.Attribute("same", dsl.String)
				dsl.Attribute("same", dsl.Int)
			})
		})
		dsl.Service("svc", func() {
			dsl.Method("call", func() {
				if mapping {
					dsl.Payload(dsl.MapOf(dsl.String, nested))
				} else {
					dsl.Payload(dsl.ArrayOf(nested))
				}
			})
		})
	}
}
