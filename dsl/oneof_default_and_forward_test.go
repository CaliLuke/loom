package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestOneOfTypeConstructorRejectsDefaultInAttribute(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		text := Type("Text", func() {
			Attribute("text", String)
		})
		json := Type("JSON", func() {
			Attribute("message", String)
		})
		Type("Parent", func() {
			Attribute("choice", OneOf(text, json), func() {
				Default(map[string]any{"type": "JSON", "value": map[string]any{"message": "hello"}})
			})
		})
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "default values are not supported for union attributes")
}

func TestOneOfTypeConstructorRejectsDefaultInPayload(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		text := Type("Text", func() {
			Attribute("text", String)
		})
		json := Type("JSON", func() {
			Attribute("message", String)
		})
		Service("Shapes", func() {
			Method("draw", func() {
				Payload(OneOf(text, json), func() {
					Default(map[string]any{"type": "JSON", "value": map[string]any{"message": "hello"}})
				})
			})
		})
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "default values are not supported for union attributes")
}

func TestOneOfDeclarationRejectsDefaultInAttribute(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		text := Type("Text", func() {
			Attribute("text", String)
		})
		json := Type("JSON", func() {
			Attribute("message", String)
		})
		Type("Parent", func() {
			OneOf("choice", func() {
				Attribute("text", text)
				Attribute("json", json)
				Default(map[string]any{"type": "json", "value": map[string]any{"message": "hello"}})
			})
		})
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "default values are not supported for union attributes")
}

func TestOneOfDeclarationRejectsDefaultInPayloadAttribute(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		text := Type("Text", func() {
			Attribute("text", String)
		})
		json := Type("JSON", func() {
			Attribute("message", String)
		})
		Service("Shapes", func() {
			Method("draw", func() {
				Payload(func() {
					OneOf("choice", func() {
						Attribute("text", text)
						Attribute("json", json)
						Default(map[string]any{"type": "json", "value": map[string]any{"message": "hello"}})
					})
				})
			})
		})
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "default values are not supported for union attributes")
}

func TestOneOfTypeConstructorForMethodPayloadAndResult(t *testing.T) {
	root := expr.RunDSL(t, func() {
		circle := Type("Circle", func() {
			Attribute("radius", Int)
			Required("radius")
		})
		square := Type("Square", func() {
			Attribute("side", Int)
			Required("side")
		})

		Service("Shapes", func() {
			Method("draw", func() {
				Payload(OneOf(circle, square))
				Result(OneOf(circle, square))
			})
		})
	})

	method := root.Services[0].Methods[0]
	payload := expr.AsUnion(method.Payload.Type)
	if payload == nil {
		t.Errorf("expected payload union type")
		return
	}
	if payload.TypeName != "CircleOrSquare" {
		t.Errorf("expected payload union type name %q, got %q", "CircleOrSquare", payload.TypeName)
	}

	result := expr.AsUnion(method.Result.Type)
	if result == nil {
		t.Errorf("expected result union type")
		return
	}
	if result.TypeName != "CircleOrSquare" {
		t.Errorf("expected result union type name %q, got %q", "CircleOrSquare", result.TypeName)
	}
}

func TestOneOfTypeConstructorSupportsForwardDeclaredTypeInAttribute(t *testing.T) {
	root := expr.RunDSL(t, func() {
		Type("A", func() {
			Attribute("choice", OneOf("B", Int))
		})
		Type("B", func() {
			Attribute("value", String)
			Required("value")
		})
	})

	atype := root.UserType("A")
	require.NotNil(t, atype)

	choice := atype.Attribute().Find("choice")
	require.NotNil(t, choice)

	union := expr.AsUnion(choice.Type)
	require.NotNil(t, union)
	require.Len(t, union.Values, 2)

	first, ok := union.Values[0].Attribute.Type.(expr.UserType)
	require.True(t, ok, "expected first union branch to resolve to a user type")
	if first.Name() != "B" {
		t.Errorf("expected first union branch type %q, got %q", "B", first.Name())
	}
}

func TestOneOfTypeConstructorSupportsTypeNameOverride(t *testing.T) {
	root := expr.RunDSL(t, func() {
		Type("Envelope", func() {
			Attribute("event", OneOf("NodeCreated", "NodeDeleted"), func() {
				Meta("oneof:typename", "RealtimeEvent")
				Meta("oneof:type:field", "type")
				Meta("oneof:value:field", "payload")
			})
		})
		Type("NodeCreated", func() {
			Attribute("id", String)
			Required("id")
		})
		Type("NodeDeleted", func() {
			Attribute("id", String)
			Required("id")
		})
	})

	envelope := root.UserType("Envelope")
	require.NotNil(t, envelope)

	event := envelope.Attribute().Find("event")
	require.NotNil(t, event)

	union := expr.AsUnion(event.Type)
	require.NotNil(t, union)
	require.True(t, union.ExplicitTypeName)
	require.Equal(t, "RealtimeEvent", union.TypeName)
	require.Equal(t, "type", union.GetTypeKey())
	require.Equal(t, "payload", union.GetValueKey())
}

func TestOneOfTypeConstructorSupportsForwardDeclaredTypeInPayloadAndResult(t *testing.T) {
	root := expr.RunDSL(t, func() {
		Service("calc", func() {
			Method("show", func() {
				Payload(OneOf("Later", Int))
				Result(OneOf("Later", Int))
			})
		})
		Type("Later", func() {
			Attribute("message", String)
			Required("message")
		})
	})

	method := root.Services[0].Methods[0]
	payload := expr.AsUnion(method.Payload.Type)
	require.NotNil(t, payload)
	require.Len(t, payload.Values, 2)

	payloadType, ok := payload.Values[0].Attribute.Type.(expr.UserType)
	require.True(t, ok, "expected payload branch to resolve to a user type")
	if payloadType.Name() != "Later" {
		t.Errorf("expected payload branch type %q, got %q", "Later", payloadType.Name())
	}

	result := expr.AsUnion(method.Result.Type)
	require.NotNil(t, result)
	require.Len(t, result.Values, 2)

	resultType, ok := result.Values[0].Attribute.Type.(expr.UserType)
	require.True(t, ok, "expected result branch to resolve to a user type")
	if resultType.Name() != "Later" {
		t.Errorf("expected result branch type %q, got %q", "Later", resultType.Name())
	}
}

func TestOneOfTypeConstructorSupportsRecursiveNamedVariants(t *testing.T) {
	root := expr.RunDSL(t, func() {
		Type("Node", func() {
			Attribute("next", OneOf("Node", "Leaf"))
		})
		Type("Leaf", func() {
			Attribute("value", String)
		})
	})

	node := root.UserType("Node")
	require.NotNil(t, node)

	next := node.Attribute().Find("next")
	require.NotNil(t, next)

	union := expr.AsUnion(next.Type)
	require.NotNil(t, union)
	require.Len(t, union.Values, 2)

	first, ok := union.Values[0].Attribute.Type.(expr.UserType)
	require.True(t, ok, "expected recursive branch to resolve to a user type")
	if first.Name() != "Node" {
		t.Errorf("expected recursive branch type %q, got %q", "Node", first.Name())
	}
}

func TestOneOfTypeConstructorReportsUnresolvedForwardType(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Type("A", func() {
			Attribute("choice", OneOf("Missing", Int))
		})
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), `unknown type reference "Missing"`)
}

func TestOneOfTypeConstructorWithAttributeDSLReportsUnresolvedForwardType(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Type("Envelope", func() {
			Attribute("choice", OneOf("Missing", Int), func() {
				Description("choice")
			})
		})
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), `unknown type reference "Missing"`)
}

func TestOneOfDeclarationFormUsedAsPayloadTypeFails(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Service("Shapes", func() {
			Method("draw", func() {
				Payload(OneOf("Inner", func() {
					Attribute("value", String)
				}))
			})
		})
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid use of OneOf")
}

func TestOneOfDeclarationFormUsedAsAttributeTypeFails(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Type("Parent", func() {
			Attribute("choice", OneOf("Inner", func() {
				Attribute("value", String)
			}))
		})
	})

	require.Error(t, err)
}
