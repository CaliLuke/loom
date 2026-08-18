package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func TestOneOfCustomKeys(t *testing.T) {
	cases := []struct {
		name           string
		typeKey        string
		valueKey       string
		expectedType   string
		expectedValue  string
		expectError    bool
		errorSubstring string
	}{
		{
			name:          "custom type and value keys",
			typeKey:       "kind",
			valueKey:      "data",
			expectedType:  "kind",
			expectedValue: "data",
		},
		{
			name:          "default keys when not specified",
			typeKey:       "",
			valueKey:      "",
			expectedType:  "type",
			expectedValue: "value",
		},
		{
			name:          "custom type key only",
			typeKey:       "discriminator",
			valueKey:      "",
			expectedType:  "discriminator",
			expectedValue: "value",
		},
		{
			name:          "custom value key only",
			typeKey:       "",
			valueKey:      "payload",
			expectedType:  "type",
			expectedValue: "payload",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eval.SetupTestContext(t)

			ut := &expr.UserTypeExpr{
				AttributeExpr: &expr.AttributeExpr{
					Type: &expr.Object{},
				},
				TypeName: "TestType",
			}

			eval.Execute(func() {
				OneOf("Shape", func() {
					if tc.typeKey != "" {
						Meta("oneof:type:field", tc.typeKey)
					}
					if tc.valueKey != "" {
						Meta("oneof:value:field", tc.valueKey)
					}
					Attribute("Circle", Int)
					Attribute("Square", String)
				})
			}, ut)

			if tc.expectError {
				if eval.Context.Errors == nil {
					t.Fatal("expected DSL error, got none")
				}
				return
			}

			if eval.Context.Errors != nil {
				t.Fatalf("unexpected DSL errors: %v", eval.Context.Errors)
			}

			obj := ut.Attribute().Type.(*expr.Object)
			shapeAttr := obj.Attribute("Shape")
			if shapeAttr == nil {
				t.Fatal("Shape attribute not found")
			}

			union, ok := shapeAttr.Type.(*expr.Union)
			if !ok {
				t.Fatalf("expected Union type, got %T", shapeAttr.Type)
			}

			if union.GetTypeKey() != tc.expectedType {
				t.Errorf("expected GetTypeKey() %q, got %q", tc.expectedType, union.GetTypeKey())
			}
			if union.GetValueKey() != tc.expectedValue {
				t.Errorf("expected GetValueKey() %q, got %q", tc.expectedValue, union.GetValueKey())
			}
		})
	}
}

func TestOneOfCustomKeysSameKeyError(t *testing.T) {
	eval.SetupTestContext(t)

	ut := &expr.UserTypeExpr{
		AttributeExpr: &expr.AttributeExpr{
			Type: &expr.Object{},
		},
		TypeName: "TestType",
	}

	eval.Execute(func() {
		OneOf("Shape", func() {
			Meta("oneof:type:field", "same")
			Meta("oneof:value:field", "same")
			Attribute("Circle", Int)
		})
	}, ut)

	if eval.Context.Errors == nil {
		t.Fatal("expected DSL error for same type and value keys, got none")
	}
}

func TestOneOfTypeConstructor(t *testing.T) {
	expr.SetupTestDSL(t)

	circle := Type("Circle", func() {
		Attribute("radius", Int)
	})
	square := Type("Square", func() {
		Attribute("side", Int)
	})

	dt := OneOf(circle, square)
	if eval.Context.Errors != nil {
		t.Errorf("unexpected DSL errors: %v", eval.Context.Errors)
	}

	union, ok := dt.(*expr.Union)
	if !ok {
		t.Errorf("expected *expr.Union, got %T", dt)
		return
	}
	if union.TypeName != "CircleOrSquare" {
		t.Errorf("expected union type name %q, got %q", "CircleOrSquare", union.TypeName)
	}
	if len(union.Values) != 2 {
		t.Errorf("expected 2 union branches, got %d", len(union.Values))
		return
	}
	if union.Values[0].Name != "Circle" {
		t.Errorf("expected first branch name %q, got %q", "Circle", union.Values[0].Name)
	}
	if union.Values[1].Name != "Square" {
		t.Errorf("expected second branch name %q, got %q", "Square", union.Values[1].Name)
	}
}

func TestTypeNamesWrappedOneOfWithoutMutatingSource(t *testing.T) {
	var base expr.DataType
	var baseName string
	root := expr.RunDSL(t, func() {
		start := Type("WrappedStart", func() {
			Attribute("start", String)
		})
		stop := Type("WrappedStop", func() {
			Attribute("stop", String)
		})
		base = OneOf(start, stop)
		baseName = expr.AsUnion(base).TypeName
		Type("Command", base)
		Type("Instruction", base)
	})

	command := root.UserType("Command")
	instruction := root.UserType("Instruction")
	require.NotNil(t, command)
	require.NotNil(t, instruction)
	require.Equal(t, "Command", expr.AsUnion(command.Attribute().Type).TypeName)
	require.Equal(t, "Instruction", expr.AsUnion(instruction.Attribute().Type).TypeName)
	require.Equal(t, baseName, expr.AsUnion(base).TypeName)
	require.NotSame(t, expr.AsUnion(command.Attribute().Type), expr.AsUnion(instruction.Attribute().Type))
}

func TestUntaggedMarksOneOfTypeConstructor(t *testing.T) {
	root := expr.RunDSL(t, func() {
		ok := Type("UntaggedOK", func() {
			Attribute("data", String)
			Required("data")
		})
		failure := Type("UntaggedFailure", func() {
			Attribute("error", String)
			Required("error")
		})
		Service("untagged", func() {
			Method("show", func() {
				Result(OneOf(ok, failure), func() {
					Untagged()
				})
			})
		})
	})

	union := expr.AsUnion(root.Services[0].Methods[0].Result.Type)
	require.NotNil(t, union)
	require.True(t, union.Untagged)
}

func TestUntaggedRejectsNonObjectBranches(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Service("untagged", func() {
			Method("show", func() {
				Result(OneOf(String, Int), func() {
					Untagged()
				})
			})
		})
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), `untagged OneOf branch "String" must be a concrete named object type`)
}

func TestUntaggedRejectsNestedBranchFields(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		nested := Type("NestedUntagged", func() {
			Attribute("child", func() {
				Attribute("value", String)
			})
		})
		flat := Type("FlatUntagged", func() {
			Attribute("value", String)
		})
		Service("untagged", func() {
			Method("show", func() {
				Result(OneOf(nested, flat), func() {
					Untagged()
				})
			})
		})
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), `untagged OneOf branch "NestedUntagged" field "child" must be primitive`)
}

func TestUntaggedRejectsAmbiguousJSONFieldMetadata(t *testing.T) {
	tests := map[string]struct {
		branch func() expr.UserType
		want   string
	}{
		"ignored field": {
			branch: func() expr.UserType {
				return Type("IgnoredJSONField", func() {
					Attribute("value", String, func() {
						Meta("struct:tag:json", "-")
					})
				})
			},
			want: `field "value" cannot use json tag "-"`,
		},
		"duplicate name": {
			branch: func() expr.UserType {
				return Type("DuplicateJSONField", func() {
					Attribute("first", String, func() {
						Meta("struct:tag:json:name", "same")
					})
					Attribute("second", String, func() {
						Meta("struct:tag:json", "same,omitempty")
					})
				})
			},
			want: `duplicate JSON field name "same"`,
		},
		"explicit open object": {
			branch: func() expr.UserType {
				return Type("ExplicitOpenJSONField", func() {
					Meta("openapi:additionalProperties", "true")
					Attribute("value", String)
				})
			},
			want: "must use the default open object or openapi:additionalProperties false",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				branch := test.branch()
				other := Type("OtherJSONField", func() {
					Attribute("other", String)
				})
				Service("untagged", func() {
					Method("show", func() {
						Result(OneOf(branch, other), func() {
							Untagged()
						})
					})
				})
			})

			require.Error(t, err)
			require.Contains(t, err.Error(), test.want)
		})
	}
}

func TestOneOfTypeConstructorDuplicateNames(t *testing.T) {
	expr.SetupTestDSL(t)

	dt := OneOf(String, String)
	if eval.Context.Errors != nil {
		t.Errorf("unexpected DSL errors: %v", eval.Context.Errors)
	}

	union, ok := dt.(*expr.Union)
	if !ok {
		t.Errorf("expected *expr.Union, got %T", dt)
		return
	}
	if len(union.Values) != 2 {
		t.Errorf("expected 2 union branches, got %d", len(union.Values))
		return
	}
	if union.Values[0].Name != "String" {
		t.Errorf("expected first branch name %q, got %q", "String", union.Values[0].Name)
	}
	if union.Values[1].Name != "String2" {
		t.Errorf("expected second branch name %q, got %q", "String2", union.Values[1].Name)
	}
}

func TestOneOfTypeConstructorReservesExplicitNames(t *testing.T) {
	expr.SetupTestDSL(t)

	typeA := Type("TypeA", func() {
		Attribute("value", String)
	})
	typeA2 := Type("TypeA2", func() {
		Attribute("value", Int)
	})

	dt := OneOf(typeA, typeA, typeA2)
	if eval.Context.Errors != nil {
		t.Errorf("unexpected DSL errors: %v", eval.Context.Errors)
	}

	union, ok := dt.(*expr.Union)
	if !ok {
		t.Errorf("expected *expr.Union, got %T", dt)
		return
	}
	if len(union.Values) != 3 {
		t.Errorf("expected 3 union branches, got %d", len(union.Values))
		return
	}
	if union.Values[0].Name != "TypeA" {
		t.Errorf("expected first branch name %q, got %q", "TypeA", union.Values[0].Name)
	}
	if union.Values[1].Name != "TypeA3" {
		t.Errorf("expected second branch name %q, got %q", "TypeA3", union.Values[1].Name)
	}
	if union.Values[2].Name != "TypeA2" {
		t.Errorf("expected third branch name %q, got %q", "TypeA2", union.Values[2].Name)
	}
}

func TestOneOfTypeConstructorNormalizesDuplicateNamesDeterministicallyAcrossOrder(t *testing.T) {
	buildNames := func(firstName, secondName string) map[string]string {
		expr.SetupTestDSL(t)

		first := Type(firstName, func() {
			Attribute("text", String)
		})
		second := Type(secondName, func() {
			Attribute("count", Int)
		})

		union, ok := OneOf(first, second).(*expr.Union)
		require.True(t, ok)
		require.Nil(t, eval.Context.Errors)

		namesByHash := make(map[string]string, len(union.Values))
		for _, branch := range union.Values {
			namesByHash[branch.Attribute.Type.Hash()] = branch.Name
		}
		return namesByHash
	}

	forward := buildNames("foo", "Foo")
	reverse := buildNames("Foo", "foo")
	require.Equal(t, forward, reverse)
}

func TestOneOfTypeConstructorTypeNameStableAcrossOrder(t *testing.T) {
	buildTypeName := func(firstName, secondName string) string {
		expr.SetupTestDSL(t)

		first := Type(firstName, func() {
			Attribute("text", String)
		})
		second := Type(secondName, func() {
			Attribute("count", Int)
		})

		union, ok := OneOf(first, second).(*expr.Union)
		require.True(t, ok)
		require.Nil(t, eval.Context.Errors)
		return union.TypeName
	}

	require.Equal(t, buildTypeName("TextPayload", "JSONPayload"), buildTypeName("JSONPayload", "TextPayload"))
}

func TestOneOfTypeConstructorUsesDeclaredNamesForDiscriminators(t *testing.T) {
	expr.SetupTestDSL(t)

	alpha := Type("AlphaPayload", String, func() {
		TypeName("RenamedAlphaPayload")
	})
	beta := Type("BetaPayload", String, func() {
		TypeName("RenamedBetaPayload")
	})

	union, ok := OneOf(alpha, beta).(*expr.Union)
	require.True(t, ok)
	require.Nil(t, eval.Context.Errors)

	require.Equal(t, "AlphaPayloadOrBetaPayload", union.TypeName)
	require.Len(t, union.Values, 2)
	require.Equal(t, "AlphaPayload", union.Values[0].Name)
	require.Equal(t, "BetaPayload", union.Values[1].Name)
}

func TestOneOfTypeConstructorAllowsNamedComplexAliases(t *testing.T) {
	expr.SetupTestDSL(t)

	ids := Type("IDs", ArrayOf(String))
	counts := Type("Counts", MapOf(String, Int))

	union, ok := OneOf(ids, counts).(*expr.Union)
	require.True(t, ok)
	require.Nil(t, eval.Context.Errors)

	require.Equal(t, "CountsOrIDs", union.TypeName)
	require.Len(t, union.Values, 2)
	require.Equal(t, "IDs", union.Values[0].Name)
	require.Equal(t, "Counts", union.Values[1].Name)
}

func TestOneOfTypeConstructorRejectsInvalidVariant(t *testing.T) {
	expr.SetupTestDSL(t)

	dt := OneOf(String, 42)
	union, ok := dt.(*expr.Union)
	if !ok {
		t.Errorf("expected invalid union placeholder, got %T", dt)
	}
	if eval.Context.Errors == nil {
		t.Errorf("expected DSL error for invalid OneOf variant")
	}
	if union != nil && union.TypeName != "InvalidOneOf" {
		t.Errorf("expected invalid union type name %q, got %q", "InvalidOneOf", union.TypeName)
	}
}

func TestOneOfTypeConstructorResolvesNamedUserTypes(t *testing.T) {
	expr.SetupTestDSL(t)

	Type("CustomType", func() {
		Attribute("value", String)
	})

	dt := OneOf("CustomType", Int)
	if eval.Context.Errors != nil {
		t.Errorf("unexpected DSL errors: %v", eval.Context.Errors)
	}

	union, ok := dt.(*expr.Union)
	if !ok {
		t.Errorf("expected *expr.Union, got %T", dt)
		return
	}
	if len(union.Values) != 2 {
		t.Errorf("expected 2 union branches, got %d", len(union.Values))
		return
	}
	if union.Values[0].Name != "CustomType" {
		t.Errorf("expected first branch name %q, got %q", "CustomType", union.Values[0].Name)
	}
	if union.Values[1].Name != "Int" {
		t.Errorf("expected second branch name %q, got %q", "Int", union.Values[1].Name)
	}
}

func TestOneOfTypeConstructorInsideAttributeWithNamedUserType(t *testing.T) {
	root := expr.RunDSL(t, func() {
		Type("CustomType", func() {
			Attribute("value", String)
		})
		Type("Parent", func() {
			Attribute("choice", OneOf("CustomType", Int))
		})
	})
	parent := root.UserType("Parent")
	if parent == nil {
		t.Errorf("expected Parent type")
		return
	}

	choice := parent.Attribute().Find("choice")
	if choice == nil {
		t.Errorf("expected choice attribute")
		return
	}
	union := expr.AsUnion(choice.Type)
	if union == nil {
		t.Errorf("expected union attribute type")
		return
	}
	if len(union.Values) != 2 {
		t.Errorf("expected 2 union branches, got %d", len(union.Values))
	}
}

func TestOneOfTypeConstructorRejectsUnnamedComplexVariants(t *testing.T) {
	expr.SetupTestDSL(t)

	dt := OneOf(ArrayOf(String), ArrayOf(Int))
	union, ok := dt.(*expr.Union)
	if !ok {
		t.Errorf("expected invalid union placeholder, got %T", dt)
		return
	}
	if union.TypeName != "InvalidOneOf" {
		t.Errorf("expected invalid union type name %q, got %q", "InvalidOneOf", union.TypeName)
	}
	if eval.Context.Errors == nil {
		t.Errorf("expected DSL error for unnamed complex OneOf variants")
	}
}

func TestOneOfTypeConstructorMissingTypeDoesNotFallbackToString(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Type("Parent", func() {
			Attribute("choice", OneOf("MissingType", Int))
		})
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), `unknown type reference "MissingType"`)
}

func TestOneOfTypeConstructorPayloadMetaOverridesKeys(t *testing.T) {
	root := expr.RunDSL(t, func() {
		circle := Type("Circle", func() {
			Attribute("radius", Int)
		})
		square := Type("Square", func() {
			Attribute("side", Int)
		})

		Service("Shapes", func() {
			Method("draw", func() {
				Payload(OneOf(circle, square), func() {
					Meta("oneof:type:field", "kind")
					Meta("oneof:value:field", "data")
				})
			})
		})
	})

	payload := expr.AsUnion(root.Services[0].Methods[0].Payload.Type)
	if payload == nil {
		t.Errorf("expected payload union type")
		return
	}
	if payload.GetTypeKey() != "kind" {
		t.Errorf("expected payload type key %q, got %q", "kind", payload.GetTypeKey())
	}
	if payload.GetValueKey() != "data" {
		t.Errorf("expected payload value key %q, got %q", "data", payload.GetValueKey())
	}
}
