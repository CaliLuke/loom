package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestUnionPrimitivePhysicalIdentity(t *testing.T) {
	for _, primitive := range []expr.Primitive{String, Int, Bytes} {
		t.Run(primitive.Name(), func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				Type("Choice", func() {
					OneOf("value", func() {
						for _, name := range []string{"first", "second"} {
							Attribute(name, primitive, func() {
								Meta("struct:field:type", "scalar.Value", "example.com/scalar", "scalar")
								Description("authored branch")
							})
						}
						Attribute("native", primitive)
					})
				})
			})
			union := expr.AsUnion(root.UserType("Choice").Attribute().Find("value").Type)
			require.Len(t, union.Values, 3)
			for _, branch := range union.Values[:2] {
				require.Equal(t, primitive, branch.Attribute.Type)
				require.Equal(t, []string{"scalar.Value", "example.com/scalar", "scalar"}, branch.Attribute.Meta["struct:field:type"])
				require.Equal(t, "authored branch", branch.Attribute.Description)
				require.Equal(t, branch.Name, expr.UnionVariantTag(branch))
			}
			_, promoted := union.Values[2].Attribute.Type.(expr.UserType)
			require.True(t, promoted, "native branch keeps its existing generated identity")
		})
	}
}

func TestUnionCustomPrimitiveKeepsConstraints(t *testing.T) {
	root := expr.RunDSL(t, func() {
		Type("Choice", func() {
			OneOf("value", func() {
				Field(1, "text", String, func() {
					Meta("struct:field:type", "scalar.Text", "example.com/scalar")
					MinLength(2)
					Default("valid")
				})
				Field(2, "number", Int)
			})
		})
	})
	branch := expr.AsUnion(root.UserType("Choice").Attribute().Find("value").Type).Values[0]
	require.Equal(t, String, branch.Attribute.Type)
	require.Equal(t, "valid", branch.Attribute.DefaultValue)
	require.NotNil(t, branch.Attribute.Validation)
	require.Equal(t, 2, *branch.Attribute.Validation.MinLength)
	require.Equal(t, []string{"1"}, branch.Attribute.Meta["rpc:tag"])
}

func TestUntaggedCustomRepresentationStillRejected(t *testing.T) {
	for _, nested := range []bool{false, true} {
		name := map[bool]string{false: "scalar", true: "object-leaf"}[nested]
		t.Run(name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				custom := func() {
					Attribute("text", String, func() {
						Meta("struct:field:type", "scalar.Text", "example.com/scalar")
						MinLength(2)
						Default("valid")
					})
				}
				choice := Type("Choice", func() {
					OneOf("value", func() {
						if nested {
							Attribute("object", custom)
						} else {
							custom()
						}
						Attribute("other", func() {
							Attribute("number", Int)
						})
						Untagged()
					})
				})
				Service("choices", func() {
					Method("echo", func() {
						Result(choice)
					})
				})
			})
			require.Error(t, err)
			if nested {
				require.Contains(t, err.Error(), "cannot use an opaque codec or schema encoding override")
			} else {
				require.Contains(t, err.Error(), `untagged OneOf branch "text" must be a concrete named object or array type`)
			}
		})
	}
}

func TestUnionCustomAnyStillRejected(t *testing.T) {
	for _, nullableMetadata := range []bool{false, true} {
		name := map[bool]string{false: "intrinsic-null", true: "nullable-false"}[nullableMetadata]
		t.Run(name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				choice := Type("Choice", func() {
					OneOf("value", func() {
						Attribute("custom", Any, func() {
							Meta("struct:field:type", "scalar.Text", "example.com/scalar")
							if nullableMetadata {
								Meta("openapi:nullable", "false")
							}
						})
						Attribute("text", String)
					})
				})
				Service("choices", func() {
					Method("echo", func() {
						Result(choice)
					})
				})
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), "null-admitting attributes conflict with custom field type metadata")
		})
	}
}
