package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestUnionDefaultAtAttributeAndPayload(t *testing.T) {
	for _, name := range []string{"attribute", "payload", "alias"} {
		t.Run(name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				choice := Type("Choice", OneOf(func() {
					Untagged()
					Attribute("text", String)
					Attribute("number", Int)
				}))
				if name == "alias" {
					choice = Type("Alias", choice)
				}
				Service("defaults", func() {
					Method("read", func() {
						if name == "attribute" {
							Payload(func() {
								Attribute("choice", choice, func() {
									Default("hello")
								})
							})
						} else {
							Payload(choice, func() {
								Default("hello")
							})
						}
					})
				})
			})
			require.ErrorContains(t, err, "default values are not supported for union attributes")
		})
	}
}

func TestUnionDefaultAfterTransportBinding(t *testing.T) {
	for _, name := range []string{"param", "header", "aliased param"} {
		t.Run(name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				choice := Type("Choice", OneOf(func() {
					Untagged()
					Attribute("text", String)
					Attribute("number", Int)
				}))
				if name == "aliased param" {
					choice = Type("Alias", choice)
				}
				Service("default_contract", func() {
					Method("read", func() {
						Payload(func() {
							Attribute("choice", choice)
						})
						HTTP(func() {
							GET("/")
							if name == "header" {
								Header("choice:X-Choice", func() {
									Default("hello")
								})
							} else {
								Param("choice", func() {
									Default("hello")
								})
							}
						})
					})
				})
			})
			require.ErrorContains(t, err, "default values are not supported for union attributes")
		})
	}
}

func TestUnionContainerDefaultAdmission(t *testing.T) {
	for _, name := range []string{"map", "array", "object", "empty map", "empty array", "omitted union"} {
		t.Run(name, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				choice := Type("Choice", OneOf(func() {
					Untagged()
					Attribute("text", String, func() {
						Default("branch")
					})
					Attribute("number", Int)
				}))
				Type("Container", func() {
					switch name {
					case "map":
						Attribute("values", MapOf(String, choice), func() {
							Default(map[string]any{"a": "hello"})
						})
					case "array":
						Attribute("values", ArrayOf(choice), func() {
							Default([]any{3})
						})
					case "empty map":
						Attribute("values", MapOf(String, choice), func() {
							Default(map[string]any{})
						})
					case "empty array":
						Attribute("values", ArrayOf(choice), func() {
							Default([]any{})
						})
					default:
						Attribute("values", func() {
							Attribute("choice", choice)
							Attribute("label", String)
							if name == "omitted union" {
								Default(map[string]any{"label": "kept"})
							} else {
								Default(map[string]any{"choice": "hello"})
							}
						})
					}
				})
			})
			require.NotNil(t, root.UserType("Container"))
		})
	}
}
