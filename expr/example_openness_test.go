package expr_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestExampleAdmissionHonorsNamedOpenness(t *testing.T) {
	for _, policy := range []string{"closed", "open", "unset"} {
		for _, shape := range []string{"root", "field", "alias", "array", "map", "local"} {
			for _, extra := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/extra=%t", policy, shape, extra), func(t *testing.T) {
					design := func() {
						address := Type("Address", func() {
							Attribute("street", String)
							Required("street")
							if policy != "unset" && shape != "local" {
								Meta("openapi:additionalProperties", fmt.Sprint(policy == "open"))
							}
						})
						var typ expr.DataType = address
						if shape == "alias" {
							typ = Type("AliasTwo", Type("AliasOne", address))
						}
						value := map[string]any{}
						// Partial examples may omit required fields.
						if extra {
							value["extra"] = 1
						}
						var example any = value
						if shape != "root" {
							fieldType := typ
							var fieldValue any = value
							switch shape {
							case "array":
								fieldType = ArrayOf(typ)
								fieldValue = []any{value}
							case "map":
								fieldType = MapOf(String, typ)
								fieldValue = map[string]any{"home": value}
							}
							typ = Type("Holder", func() {
								Attribute("addr", fieldType, func() {
									if shape == "local" && policy != "unset" {
										Meta("openapi:additionalProperties", fmt.Sprint(policy == "open"))
									}
								})
							})
							example = map[string]any{"addr": fieldValue}
						}
						Service("example-admission", func() {
							Method("create", func() {
								Payload(typ, func() {
									Example(example)
								})
							})
						})
					}
					if policy == "closed" && extra {
						require.ErrorContains(t, expr.RunInvalidDSL(t, design), "undeclared object member")
					} else {
						expr.RunDSL(t, design)
					}
				})
			}
		}
	}
}

func TestDeclarationExampleRejectsClosedNestedMembers(t *testing.T) {
	for _, used := range []bool{false, true} {
		t.Run(fmt.Sprint(used), func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				address := Type("Address", func() {
					Attribute("street", String)
					Meta("openapi:additionalProperties", "false")
				})
				holder := Type("Holder", func() {
					Attribute("addr", address)
					Example(map[string]any{"addr": map[string]any{"street": "x", "extra": 1}})
				})
				if used {
					Service("records", func() {
						Method("create", func() {
							Payload(holder)
							Result(holder)
						})
					})
				}
			})
			require.ErrorContains(t, err, `type "Holder": example has undeclared object member "extra" at addr`)
			require.Equal(t, 1, strings.Count(err.Error(), "undeclared object member"))
		})
	}
}

func TestExampleOpennessAcrossAdditionalMembersAndUnions(t *testing.T) {
	for _, shape := range []string{"additional", "closed-union", "open-alternative"} {
		t.Run(shape, func(t *testing.T) {
			design := func() {
				closed := Type("Closed", func() {
					Attribute("id", String)
					Meta("openapi:additionalProperties", "false")
				})
				var typ expr.DataType
				var example any = map[string]any{"id": "x", "extra": true}
				if shape == "additional" {
					typ = Type("Dynamic", func() {
						AdditionalAttributes(closed)
					})
					example = map[string]any{"home": example}
				} else {
					other := Type("Other", func() {
						Attribute("id", String)
						if shape != "open-alternative" {
							Meta("openapi:additionalProperties", "false")
						}
					})
					typ = OneOf(closed, other)
				}
				Service("records", func() {
					Method("create", func() {
						Payload(typ, func() {
							Example(example)
						})
					})
				})
			}
			if shape == "open-alternative" {
				expr.RunDSL(t, design)
			} else {
				require.ErrorContains(t, expr.RunInvalidDSL(t, design), "undeclared object member")
			}
		})
	}
}
