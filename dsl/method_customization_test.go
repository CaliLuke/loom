package dsl_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

type methodTypeSlot struct {
	name      string
	define    func(any, ...any)
	attribute func(*expr.MethodExpr) *expr.AttributeExpr
}

var methodTypeSlots = []methodTypeSlot{
	{"Payload", Payload, func(m *expr.MethodExpr) *expr.AttributeExpr {
		return m.Payload
	}},
	{"Result", Result, func(m *expr.MethodExpr) *expr.AttributeExpr {
		return m.Result
	}},
	{"StreamingPayload", StreamingPayload, func(m *expr.MethodExpr) *expr.AttributeExpr {
		return m.StreamingPayload
	}},
	{"StreamingResult", StreamingResult, func(m *expr.MethodExpr) *expr.AttributeExpr {
		return m.StreamingResult
	}},
}

// TestNamedMethodScalarCustomizationRunsOnce checks that a named scalar
// occurrence owns exactly the authored local sample and bound, while its
// declaration keeps the contributor's distinct sample and bound.
func TestNamedMethodScalarCustomizationRunsOnce(t *testing.T) {
	for _, slot := range methodTypeSlots {
		t.Run(slot.name, func(t *testing.T) {
			calls := 0
			root := expr.RunDSL(t, func() {
				count := Type("Count", Int, func() {
					Minimum(0)
					Example(1)
				})
				Service("svc", func() {
					Method("plain", func() {
						Payload(count)
					})
					Method("custom", func() {
						slot.define(count, "local count", func() {
							calls++
							Minimum(5)
							Example(8)
						})
					})
				})
			})
			assertCalledOnce(t, calls)
			custom := slot.attribute(root.Service("svc").Method("custom"))
			require.Equal(t, "local count", custom.Description)
			if assertSingleExample(t, custom, 8) {
				require.NotNil(t, custom.Validation)
				require.Equal(t, "5", custom.Validation.Minimum.String())
			}
			declaration := custom.Type.(expr.UserType)
			require.Equal(t, "Count", declaration.Name(), "local bounds and samples keep the declaration identity")
			require.NotSame(t, root.UserType("Count"), declaration)
			for _, contributor := range []*expr.AttributeExpr{declaration.Attribute(), root.UserType("Count").Attribute()} {
				assertSingleExample(t, contributor, 1)
				require.Equal(t, "0", contributor.Validation.Minimum.String())
			}
			plain := root.Service("svc").Method("plain").Payload
			require.Same(t, root.UserType("Count"), plain.Type)
			require.Empty(t, plain.UserExamples)
		})
	}
}

// TestNamedMethodObjectCustomizationRunsOnce checks Required and Extend
// materialization after the single callback, for plain types and result types.
func TestNamedMethodObjectCustomizationRunsOnce(t *testing.T) {
	for _, slot := range methodTypeSlots {
		for _, resultType := range []bool{false, true} {
			name := slot.name + "/type"
			if resultType {
				name = slot.name + "/result-type"
			}
			t.Run(name, func(t *testing.T) {
				calls := 0
				root := expr.RunDSL(t, func() {
					audit := Type("Audit", func() {
						Attribute("actor", String)
						Required("actor")
					})
					var value expr.UserType
					fields := func() {
						Attribute("id", String)
						Attribute("note", String)
					}
					if resultType {
						value = ResultType("application/vnd.value", "Value", func() {
							Attributes(fields)
						})
					} else {
						value = Type("Value", fields)
					}
					Service("svc", func() {
						Method("plain", func() {
							Payload(value)
							Result(value)
						})
						Method("custom", func() {
							slot.define(value, func() {
								calls++
								Extend(audit)
								Required("id")
								Example(map[string]any{"id": "a", "actor": "b"})
							})
						})
						Method("sibling", func() {
							slot.define(value, func() {
								Required("note")
							})
						})
					})
				})
				assertCalledOnce(t, calls)
				custom := slot.attribute(root.Service("svc").Method("custom"))
				assertSingleExample(t, custom, map[string]any{"id": "a", "actor": "b"})
				declaration := custom.Type.(expr.UserType)
				require.Equal(t, "Value_custom_"+slot.name, declaration.Name())
				owner := declaration.Attribute()
				require.NotNil(t, custom.Find("actor"))
				require.True(t, owner.IsRequired("id"), "the method declaration owns local requirements")
				require.True(t, owner.IsRequired("actor"), "the method declaration owns inherited requirements")
				require.False(t, owner.IsRequired("note"))

				sibling := slot.attribute(root.Service("svc").Method("sibling"))
				require.Equal(t, "Value_sibling_"+slot.name, sibling.Type.Name())
				require.Nil(t, sibling.Find("actor"))
				require.True(t, sibling.IsRequired("note"))
				require.False(t, sibling.IsRequired("id"))

				for _, plain := range []*expr.AttributeExpr{
					root.Service("svc").Method("plain").Payload,
					root.Service("svc").Method("plain").Result,
				} {
					require.Equal(t, "Value", plain.Type.Name())
					require.Nil(t, plain.Find("actor"))
					require.False(t, plain.IsRequired("id"))
					require.False(t, plain.IsRequired("note"))
				}
				if result, ok := declaration.(*expr.ResultTypeExpr); ok {
					projected, err := expr.Project(result, expr.DefaultView)
					require.NoError(t, err)
					require.NotNil(t, expr.AsObject(projected.Type).Attribute("actor"))
					require.True(t, projected.IsRequired("id"))
					require.True(t, projected.IsRequired("actor"))
				}
			})
		}
	}
}

// TestNamedMethodCustomizationRequiresDeclaredRequirement keeps the existing
// contract: an authored Required always gives the method its own declaration,
// even when the named type already requires the field.
func TestNamedMethodCustomizationRequiresDeclaredRequirement(t *testing.T) {
	root := expr.RunDSL(t, func() {
		value := Type("Value", func() {
			Attribute("id", String)
			Required("id")
		})
		Service("svc", func() {
			Method("custom", func() {
				Payload(value, func() {
					Required("id")
				})
			})
		})
	})
	payload := root.Service("svc").Method("custom").Payload
	require.Equal(t, "Value_custom_Payload", payload.Type.Name())
	require.Equal(t, []string{"id"}, payload.Type.(expr.UserType).Attribute().Validation.Required)
	require.Equal(t, "Value", root.UserType("Value").Name())
}

// TestNamedMethodCustomizationPreservesRecursiveIdentity checks that the
// method declaration closes its own cycle and leaves the source cycle intact.
func TestNamedMethodCustomizationPreservesRecursiveIdentity(t *testing.T) {
	root := expr.RunDSL(t, func() {
		node := Type("Node", func() {
			Attribute("name", String)
			Attribute("child", "Node")
		})
		Service("svc", func() {
			Method("custom", func() {
				Payload(node, func() {
					Required("name")
				})
			})
		})
	})
	payload := root.Service("svc").Method("custom").Payload
	require.Equal(t, "Node_custom_Payload", payload.Type.Name())
	require.Same(t, payload.Type, payload.Find("child").Type)
	source := root.UserType("Node")
	require.Equal(t, "Node", source.Name())
	require.Same(t, source, source.Attribute().Find("child").Type)
	require.False(t, source.Attribute().IsRequired("name"))
}

// TestNamedMethodCustomizationKeepsReplacedTypeShared checks that only the
// method's own copy is materialized: a callback that installs another named
// type directly must not rename or extend that shared declaration.
func TestNamedMethodCustomizationKeepsReplacedTypeShared(t *testing.T) {
	root := expr.RunDSL(t, func() {
		audit := Type("Audit", func() {
			Attribute("actor", String)
		})
		value := Type("Value", func() {
			Attribute("id", String)
		})
		other := Type("Other", func() {
			Attribute("name", String)
		})
		Service("svc", func() {
			Method("custom", func() {
				Payload(value, func() {
					eval.Current().(*expr.AttributeExpr).Type = other
					Extend(audit)
				})
			})
			Method("plain", func() {
				Payload(other)
			})
		})
	})
	other := root.UserType("Other")
	require.Equal(t, "Other", other.Name())
	require.Nil(t, other.Attribute().Find("actor"), "the replaced shared type must not inherit method bases")
	payload := root.Service("svc").Method("custom").Payload
	require.NotSame(t, other, payload.Type, "occurrence composition owns its inherited fields")
	require.NotNil(t, payload.Find("actor"))
	require.NotNil(t, payload.Find("name"))
	require.Nil(t, root.UserType("Value").Attribute().Find("actor"))
}

// TestNamedMethodUnionCustomizationRunsOnce checks that union metadata
// applies to the method copy once and leaves the named union unchanged.
func TestNamedMethodUnionCustomizationRunsOnce(t *testing.T) {
	calls := 0
	root := expr.RunDSL(t, func() {
		text := Type("Text", func() {
			Attribute("text", String)
		})
		count := Type("Total", func() {
			Attribute("count", Int)
		})
		choice := Type("Choice", OneOf(text, count))
		Service("svc", func() {
			Method("custom", func() {
				Payload(choice, func() {
					calls++
					Meta("oneof:type:field", "kind")
				})
			})
		})
	})
	assertCalledOnce(t, calls)
	payload := root.Service("svc").Method("custom").Payload
	require.Equal(t, []string{"kind"}, payload.Meta["oneof:type:field"])
	require.Equal(t, "kind", expr.AsUnion(payload.Type).GetTypeKey())
	require.Equal(t, "Choice", payload.Type.Name())
	require.NotSame(t, root.UserType("Choice"), payload.Type)
	require.Equal(t, "type", expr.AsUnion(root.UserType("Choice")).GetTypeKey())
}

// TestNamedMethodCustomizationReportsCallbackErrorOnce checks that invalid
// customization DSL is diagnosed once per authored call.
func TestNamedMethodCustomizationReportsCallbackErrorOnce(t *testing.T) {
	for _, slot := range methodTypeSlots {
		t.Run(slot.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				count := Type("Count", Int)
				Service("svc", func() {
					Method("custom", func() {
						slot.define(count, func() {
							Attribute("extra", String)
						})
					})
				})
			})
			require.Equal(t, 1, strings.Count(err.Error(), `can't define child attribute "extra"`), err.Error())
		})
	}
}

// TestMethodCustomizationControlsRunOnce covers direct types and function
// forms, which never shared the named-type copy path.
func TestMethodCustomizationControlsRunOnce(t *testing.T) {
	for _, slot := range methodTypeSlots {
		for _, form := range []string{"primitive", "function"} {
			t.Run(slot.name+"/"+form, func(t *testing.T) {
				calls := 0
				root := expr.RunDSL(t, func() {
					Service("svc", func() {
						Method("custom", func() {
							if form == "primitive" {
								slot.define(Int, func() {
									calls++
									Minimum(5)
									Example(8)
								})
								return
							}
							slot.define(func() {
								calls++
								Attribute("id", String)
								Example(map[string]any{"id": "a"})
							})
						})
					})
				})
				assertCalledOnce(t, calls)
				custom := slot.attribute(root.Service("svc").Method("custom"))
				if form == "primitive" {
					assertSingleExample(t, custom, 8)
					require.Equal(t, expr.Int, custom.Type)
					return
				}
				assertSingleExample(t, custom, map[string]any{"id": "a"})
			})
		}
	}
}

func assertCalledOnce(t *testing.T, calls int) {
	t.Helper()
	if calls != 1 {
		t.Errorf("customization callback ran %d times, want exactly once", calls)
	}
}

func assertSingleExample(t *testing.T, att *expr.AttributeExpr, want any) bool {
	t.Helper()
	if len(att.UserExamples) != 1 {
		values := make([]any, len(att.UserExamples))
		for i, example := range att.UserExamples {
			values[i] = example.Value
		}
		t.Errorf("examples = %#v, want exactly one %#v", values, want)
		return false
	}
	require.EqualValues(t, want, att.UserExamples[0].Value)
	return true
}
