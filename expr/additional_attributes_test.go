package expr_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestAdditionalAttributesPolicy checks the typed policy, its placement rules
// and requirements on undeclared members.
func TestAdditionalAttributesPolicy(t *testing.T) {
	for _, tc := range []struct {
		name    string
		design  func() expr.UserType
		failure string
	}{
		{"typed object", func() expr.UserType {
			return usedType(Type("Record", func() {
				Attribute("id", String)
				AdditionalAttributes(Int, func() {
					Minimum(0)
				})
				Required("id")
			}))
		}, ""},
		{"required undeclared member", func() expr.UserType {
			return usedType(Type("Record", func() {
				Attribute("id", String)
				AdditionalAttributes(Int)
				Required("id", "note")
			}))
		}, ""},
		{"required undeclared member without typed policy", func() expr.UserType {
			return usedType(Type("Record", func() {
				Attribute("id", String)
				Required("id", "note")
			}))
		}, `required field "note" does not exist`},
		{"combined with openness metadata", func() expr.UserType {
			return usedType(Type("Record", func() {
				Attribute("id", String)
				AdditionalAttributes(Int)
				Meta("openapi:additionalProperties", "false")
			}))
		}, "AdditionalAttributes cannot be combined with openapi:additionalProperties"},
		{"on a named type occurrence", func() expr.UserType {
			base := Type("Base", func() {
				Attribute("id", String)
			})
			return usedType(Type("Holder", func() {
				Attribute("child", base, func() {
					AdditionalAttributes(Int)
				})
			}))
		}, "AdditionalAttributes must appear in an object declaration"},
		{"occurrence openness over a typed type", func() expr.UserType {
			record := Type("Record", func() {
				Attribute("id", String)
				AdditionalAttributes(Int)
			})
			return usedType(Type("Holder", func() {
				Attribute("record", record, func() {
					Meta("openapi:additionalProperties", "false")
				})
			}))
		}, `cannot override the AdditionalAttributes of "Record"`},
		{"required undeclared member reusing a mapped wire name", func() expr.UserType {
			return usedType(Type("Record", func() {
				Attribute("foo:bar", String)
				AdditionalAttributes(Int)
				Required("bar")
			}))
		}, `required member "bar" is not declared but attribute "foo:bar" already uses that JSON name`},
		{"required undeclared member reusing a JSON tag", func() expr.UserType {
			return usedType(Type("Record", func() {
				Attribute("id", String, func() {
					Meta("struct:tag:json", "count")
				})
				AdditionalAttributes(Int)
				Required("count")
			}))
		}, `already uses that JSON name`},
		{"JSON tag on the additional attributes", func() expr.UserType {
			return usedType(Type("Record", func() {
				Attribute("id", String)
				AdditionalAttributes(Int, func() {
					Meta("struct:tag:json:name", "zzz")
				})
				Required("p", "q")
			}))
		}, "AdditionalAttributes cannot set a JSON tag"},
		{"on a map", func() expr.UserType {
			return usedType(Type("Holder", func() {
				Attribute("labels", MapOf(String, String), func() {
					AdditionalAttributes(Int)
				})
			}))
		}, "AdditionalAttributes must appear in an object declaration"},
		{"declared twice", func() expr.UserType {
			return usedType(Type("Record", func() {
				Attribute("id", String)
				AdditionalAttributes(Int)
				AdditionalAttributes(String)
			}))
		}, "AdditionalAttributes is already declared"},
		{"without declared members", func() expr.UserType {
			return usedType(Type("Record", func() {
				AdditionalAttributes(Int)
				Required("count")
			}))
		}, ""},
		{"invalid additional validation", func() expr.UserType {
			return usedType(Type("Record", func() {
				Attribute("id", String)
				AdditionalAttributes(Int, func() {
					MinLength(1)
				})
			}))
		}, "invalid minimum length validation definition"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var record expr.UserType
			design := func() {
				record = tc.design()
			}
			if tc.failure != "" {
				require.ErrorContains(t, expr.RunInvalidDSL(t, design), tc.failure)
				return
			}
			expr.RunDSL(t, design)
			require.Equal(t, expr.ObjectTyped, record.Attribute().ObjectOpenness())
			require.True(t, record.Attribute().ObjectOpenness().AdmitsUndeclared())
			require.Equal(t, expr.Int, record.Attribute().AdditionalAttributes.Type)
		})
	}
}

// TestAdditionalAttributesDeclarationShapes checks inline objects that declare
// only additional attributes and policies typed by types declared later.
func TestAdditionalAttributesDeclarationShapes(t *testing.T) {
	t.Run("inline attribute object", func(t *testing.T) {
		var record expr.UserType
		expr.RunDSL(t, func() {
			record = usedType(Type("Record", func() {
				Attribute("labels", func() {
					AdditionalAttributes(Int)
					Required("a")
				})
			}))
		})
		labels := record.Attribute().Find("labels")
		require.NotNil(t, labels)
		require.True(t, expr.IsObject(labels.Type))
		require.Equal(t, expr.ObjectTyped, labels.ObjectOpenness())
		require.True(t, labels.IsRequired("a"))
	})
	t.Run("inline payload object", func(t *testing.T) {
		var method *expr.MethodExpr
		expr.RunDSL(t, func() {
			Service("records", func() {
				Method("create", func() {
					Payload(func() {
						AdditionalAttributes(Int)
					})
					Result(func() {
						AdditionalAttributes(String)
					})
				})
			})
		})
		method = expr.Root.Service("records").Method("create")
		for _, att := range []*expr.AttributeExpr{method.Payload, method.Result} {
			require.NotEqual(t, expr.Empty, att.Type)
			require.True(t, expr.IsObject(att.Type))
			require.Equal(t, expr.ObjectTyped, att.ObjectOpenness())
		}
	})
	t.Run("policy type declared later", func(t *testing.T) {
		var holder expr.UserType
		expr.RunDSL(t, func() {
			record := Type("Record", func() {
				Attribute("id", String)
				AdditionalAttributes("Label")
			})
			holder = usedType(Type("Holder", func() {
				Attribute("record", record, func() {
					Description("a localized copy")
				})
			}))
			Type("Label", func() {
				Attribute("name", String)
				Required("name")
			})
		})
		copied := holder.Attribute().Find("record").TypedObjectDeclaration()
		require.NotNil(t, copied)
		label := copied.AdditionalAttributes
		require.NotNil(t, expr.AsObject(label.Type))
		require.Len(t, *expr.AsObject(label.Type), 1)
		require.True(t, label.IsRequired("name"))
	})
}

// TestAdditionalAttributesInlineJSONSchema checks that inline JSON Schema
// describes typed undeclared members and their requirements.
func TestAdditionalAttributesInlineJSONSchema(t *testing.T) {
	var record expr.UserType
	expr.RunDSL(t, func() {
		record = usedType(Type("Record", func() {
			Attribute("id", String)
			AdditionalAttributes(Int, func() {
				Minimum(0)
			})
			Required("id", "count")
		}))
	})
	encoded, err := expr.InlineJSONSchema(&expr.AttributeExpr{Type: record})
	require.NoError(t, err)
	require.JSONEq(t, `{
		"type": "object",
		"properties": {"id": {"type": "string"}},
		"required": ["id", "count"],
		"additionalProperties": {"type": "integer", "minimum": 0, "maximum": 9223372036854775807}
	}`, string(encoded))
}

// TestAdditionalAttributesHTTPBodyTypes checks that transport bodies rename
// and relocate the user type of a typed policy like the types of declared
// members.
func TestAdditionalAttributesHTTPBodyTypes(t *testing.T) {
	expr.RunDSL(t, func() {
		labels := Type("Labels", func() {
			Attribute("name", String)
			Meta("struct:pkg:path", "types")
		})
		record := Type("Record", func() {
			Attribute("decl", labels)
			AdditionalAttributes(labels)
		})
		Service("records", func() {
			Method("create", func() {
				Payload(record)
				Result(record)
				HTTP(func() {
					POST("/")
				})
			})
		})
	})
	endpoint := expr.Root.API.HTTP.Service("records").Endpoint("create")
	for name, body := range map[string]*expr.AttributeExpr{
		"request":  endpoint.Body,
		"response": endpoint.Responses[0].Body,
	} {
		t.Run(name, func(t *testing.T) {
			declaration := body.TypedObjectDeclaration()
			require.NotNil(t, declaration)
			declared := declaration.Find("decl").Type.(expr.UserType)
			policy := declaration.AdditionalAttributes.Type.(expr.UserType)
			require.Equal(t, declared.Name(), policy.Name())
			require.NotEqual(t, "Labels", policy.Name())
			_, located := policy.Attribute().Meta["struct:pkg:path"]
			require.False(t, located)
		})
	}
}

// TestAdditionalAttributesComposition checks per-source composition of typed
// policies through Extend.
func TestAdditionalAttributesComposition(t *testing.T) {
	typedBase := func(fn ...func()) expr.UserType {
		return Type("Base", func() {
			Attribute("id", String)
			AdditionalAttributes(Int, func() {
				Minimum(0)
			})
			Required("id", "count")
			for _, f := range fn {
				f()
			}
		})
	}
	for _, tc := range []struct {
		name    string
		design  func() expr.UserType
		failure string
		check   func(t *testing.T, derived expr.UserType)
	}{
		{"inherits the typed policy and undeclared requirement", func() expr.UserType {
			base := typedBase()
			return usedType(Type("Derived", func() {
				Extend(base)
			}))
		}, "", func(t *testing.T, derived expr.UserType) {
			att := derived.Attribute()
			require.Equal(t, expr.ObjectTyped, att.ObjectOpenness())
			require.Equal(t, expr.Int, att.AdditionalAttributes.Type)
			require.True(t, att.IsRequired("count"))
			require.Nil(t, att.Find("count"), "an inherited required extra is not promoted to a declaration")
		}},
		{"derived member satisfies the base policy", func() expr.UserType {
			base := typedBase()
			return usedType(Type("Derived", func() {
				Extend(base)
				Attribute("total", Int)
			}))
		}, "", func(t *testing.T, derived expr.UserType) {
			total := derived.Attribute().Find("total")
			require.NotNil(t, total.Validation)
			require.NotNil(t, total.Validation.Minimum, "the base policy constrains a member it does not declare")
			require.Equal(t, "0", fmt.Sprint(*total.Validation.Minimum))
		}},
		{"derived member violates the base policy", func() expr.UserType {
			base := typedBase()
			return usedType(Type("Derived", func() {
				Extend(base)
				Attribute("note", String)
			}))
		}, `incompatible Extend types "string" and "int"`, nil},
		{"derived policy refines the base policy", func() expr.UserType {
			base := typedBase()
			return usedType(Type("Derived", func() {
				Extend(base)
				AdditionalAttributes(Int, func() {
					Maximum(10)
				})
			}))
		}, "", func(t *testing.T, derived expr.UserType) {
			validation := derived.Attribute().AdditionalAttributes.Validation
			require.NotNil(t, validation)
			require.Equal(t, "0", fmt.Sprint(*validation.Minimum))
			require.Equal(t, "10", fmt.Sprint(*validation.Maximum))
		}},
		{"typed derived over a closed base", func() expr.UserType {
			base := Type("Base", func() {
				Attribute("id", String)
				Meta("openapi:additionalProperties", "false")
			})
			return usedType(Type("Derived", func() {
				Extend(base)
				AdditionalAttributes(Int)
			}))
		}, "cannot reopen inherited closed object", nil},
		{"disjoint peer policies close the object", func() expr.UserType {
			base := Type("Base", func() {
				Attribute("id", String)
				AdditionalAttributes(Int)
			})
			peer := Type("Peer", func() {
				Attribute("id", String)
				AdditionalAttributes(String)
			})
			return usedType(Type("Derived", func() {
				Extend(base)
				Extend(peer)
			}))
		}, "", func(t *testing.T, derived expr.UserType) {
			require.Equal(t, expr.ObjectClosed, derived.Attribute().ObjectOpenness())
		}},
		{"disjoint peer policies with an undeclared requirement", func() expr.UserType {
			base := typedBase()
			peer := Type("Peer", func() {
				Attribute("id", String)
				AdditionalAttributes(String)
			})
			return usedType(Type("Derived", func() {
				Extend(base)
				Extend(peer)
			}))
		}, `required member "count" is unsatisfiable`, nil},
		{"contradictory peer constraints close the object", func() expr.UserType {
			base := Type("Base", func() {
				Attribute("id", String)
				AdditionalAttributes(Int, func() {
					Minimum(5)
				})
			})
			peer := Type("Peer", func() {
				Attribute("id", String)
				AdditionalAttributes(Int, func() {
					Maximum(3)
				})
			})
			return usedType(Type("Derived", func() {
				Extend(base)
				Extend(peer)
			}))
		}, "", func(t *testing.T, derived expr.UserType) {
			require.Equal(t, expr.ObjectClosed, derived.Attribute().ObjectOpenness())
		}},
		{"an unvalidated Any policy does not constrain the conjunction", func() expr.UserType {
			base := typedBase()
			return usedType(Type("Derived", func() {
				Extend(base)
				AdditionalAttributes(Any)
			}))
		}, "", func(t *testing.T, derived expr.UserType) {
			att := derived.Attribute()
			require.Equal(t, expr.ObjectTyped, att.ObjectOpenness())
			require.Equal(t, expr.Int, att.AdditionalAttributes.Type)
			require.NotNil(t, att.AdditionalAttributes.Validation.Minimum)
		}},
		{"an inherited unvalidated Any policy keeps the derived policy", func() expr.UserType {
			base := Type("Base", func() {
				Attribute("id", String)
				AdditionalAttributes(Any)
			})
			return usedType(Type("Derived", func() {
				Extend(base)
				AdditionalAttributes(Int)
			}))
		}, "", func(t *testing.T, derived expr.UserType) {
			require.Equal(t, expr.Int, derived.Attribute().AdditionalAttributes.Type)
		}},
		{"an inherited unvalidated Any policy admits derived members", func() expr.UserType {
			base := Type("Base", func() {
				Attribute("id", String)
				AdditionalAttributes(Any)
			})
			return usedType(Type("Derived", func() {
				Extend(base)
				Attribute("name", String)
			}))
		}, "", func(t *testing.T, derived expr.UserType) {
			require.Equal(t, expr.String, derived.Attribute().Find("name").Type)
			require.Equal(t, expr.ObjectTyped, derived.Attribute().ObjectOpenness())
		}},
		{"explicit open derived over a typed base", func() expr.UserType {
			base := typedBase()
			return usedType(Type("Derived", func() {
				Extend(base)
				Meta("openapi:additionalProperties", "true")
			}))
		}, "cannot reopen inherited typed object", nil},
		{"explicit open derived over an Any base", func() expr.UserType {
			base := Type("Base", func() {
				Attribute("id", String)
				AdditionalAttributes(Any)
			})
			return usedType(Type("Derived", func() {
				Extend(base)
				Meta("openapi:additionalProperties", "true")
			}))
		}, "cannot reopen inherited typed object", nil},
		{"overlapping integer and number policies are rejected, not closed", func() expr.UserType {
			base := typedBase()
			return usedType(Type("Derived", func() {
				Extend(base)
				AdditionalAttributes(Float64)
			}))
		}, "overlap but neither refines the other", nil},
		{"overlapping integer carriers are rejected, not closed", func() expr.UserType {
			base := typedBase()
			return usedType(Type("Derived", func() {
				Extend(base)
				AdditionalAttributes(Int32)
			}))
		}, "overlap but neither refines the other", nil},
		{"types extending a contradictory composition also close", func() expr.UserType {
			base := Type("Base", func() {
				Attribute("id", String)
				AdditionalAttributes(Int, func() {
					Minimum(5)
				})
			})
			peer := Type("Peer", func() {
				Attribute("id", String)
				AdditionalAttributes(Int, func() {
					Maximum(3)
				})
			})
			middle := Type("Middle", func() {
				Extend(base)
				Extend(peer)
			})
			return usedType(Type("Derived", func() {
				Extend(middle)
			}))
		}, "", func(t *testing.T, derived expr.UserType) {
			require.Equal(t, expr.ObjectClosed, derived.Attribute().ObjectOpenness())
		}},
		{"peer enum and bound policies compose in either order", func() expr.UserType {
			enum := Type("Enumerated", func() {
				Attribute("id", String)
				AdditionalAttributes(Int, func() {
					Enum(-1, 5)
				})
			})
			bound := Type("Bounded", func() {
				Attribute("id", String)
				AdditionalAttributes(Int, func() {
					Minimum(0)
				})
			})
			reversed := Type("Reversed", func() {
				Extend(bound)
				Extend(enum)
			})
			derived := Type("Derived", func() {
				Extend(enum)
				Extend(bound)
			})
			return usedType(Type("Holder", func() {
				Attribute("derived", derived)
				Attribute("reversed", reversed)
			}))
		}, "", func(t *testing.T, derived expr.UserType) {
			for _, name := range []string{"Derived", "Reversed"} {
				att := expr.Root.UserType(name).Attribute()
				require.Equal(t, expr.ObjectTyped, att.ObjectOpenness(), name)
				constraints, err := expr.EffectiveConstraintsFor(att.AdditionalAttributes)
				require.NoError(t, err, name)
				values, present := constraints.EnumCandidates()
				require.True(t, present, name)
				require.Len(t, values, 1, name)
			}
		}},
		{"authored enum member violating an inherited bound", func() expr.UserType {
			bound := Type("Bounded", func() {
				Attribute("id", String)
				AdditionalAttributes(Int, func() {
					Minimum(0)
				})
			})
			return usedType(Type("Derived", func() {
				Extend(bound)
				AdditionalAttributes(Int, func() {
					Enum(-1, 5)
				})
			}))
		}, "violates the effective contract", nil},
		{"disjoint nullable peer policies", func() expr.UserType {
			base := Type("Base", func() {
				Attribute("id", String)
				AdditionalAttributes(Int, func() {
					Nullable()
				})
			})
			peer := Type("Peer", func() {
				Attribute("id", String)
				AdditionalAttributes(String, func() {
					Nullable()
				})
			})
			return usedType(Type("Derived", func() {
				Extend(base)
				Extend(peer)
			}))
		}, "admit only null", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var derived expr.UserType
			design := func() {
				derived = tc.design()
			}
			if tc.failure != "" {
				require.ErrorContains(t, expr.RunInvalidDSL(t, design), tc.failure)
				return
			}
			expr.RunDSL(t, design)
			tc.check(t, derived)
		})
	}
}

// TestAdditionalAttributesValueAdmission checks that values satisfy the typed
// policy and its undeclared requirements, at any depth of a named type.
func TestAdditionalAttributesValueAdmission(t *testing.T) {
	var record expr.UserType
	expr.RunDSL(t, func() {
		record = usedType(Type("Record", func() {
			Attribute("id", String)
			AdditionalAttributes(Int, func() {
				Minimum(0)
			})
			Required("id", "count")
		}))
	})
	for _, tc := range []struct {
		name    string
		raw     map[string]any
		role    expr.ValueRole
		outcome expr.ValueOutcome
	}{
		{"declared and extra members", map[string]any{"id": "a", "count": 1, "other": 2}, expr.ValueRoleDefault, expr.ValueResolved},
		{"missing undeclared requirement", map[string]any{"id": "a"}, expr.ValueRoleDefault, expr.ValueInvalid},
		{"incomplete example without undeclared requirement", map[string]any{"id": "a"}, expr.ValueRoleExample, expr.ValueIncomplete},
		{"extra member of the wrong type", map[string]any{"id": "a", "count": "x"}, expr.ValueRoleDefault, expr.ValueInvalid},
		{"extra member violates validation", map[string]any{"id": "a", "count": 1, "other": -1}, expr.ValueRoleDefault, expr.ValueInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			context := expr.NewValueContext()
			occurrence, err := context.NewOccurrence(&expr.AttributeExpr{Type: record})
			require.NoError(t, err)
			result := context.Resolve(occurrence, context.SupplyValue(expr.ValueInput{Raw: tc.raw}), tc.role)
			require.Equal(t, tc.outcome, result.Outcome(), "%v", result.Diagnostics())
		})
	}
}

// TestAdditionalAttributesGeneratedExample checks that generated examples
// include required members the object does not declare.
func TestAdditionalAttributesGeneratedExample(t *testing.T) {
	var record expr.UserType
	expr.RunDSL(t, func() {
		record = usedType(Type("Record", func() {
			Attribute("id", String)
			AdditionalAttributes(Int, func() {
				Minimum(0)
			})
			Required("id", "count")
		}))
	})
	example, ok := record.Attribute().Example(expr.NewRandom("additional")).(map[string]any)
	require.True(t, ok)
	require.Contains(t, example, "id")
	require.Contains(t, example, "count")
	context := expr.NewValueContext()
	occurrence, err := context.NewOccurrence(&expr.AttributeExpr{Type: record})
	require.NoError(t, err)
	result := context.Resolve(occurrence, context.SupplyValue(expr.ValueInput{Raw: example}), expr.ValueRoleDefault)
	require.Equal(t, expr.ValueResolved, result.Outcome(), "%v", result.Diagnostics())
}

// TestAdditionalAttributesRejectedPlacements checks the placements that typed
// objects do not support.
func TestAdditionalAttributesRejectedPlacements(t *testing.T) {
	typed := func() expr.UserType {
		return Type("Record", func() {
			Attribute("id", String)
			AdditionalAttributes(Int)
		})
	}
	for _, tc := range []struct {
		name    string
		design  func()
		failure string
	}{
		{"gRPC message", func() {
			record := typed()
			Service("records", func() {
				Method("create", func() {
					Payload(record)
					GRPC(func() {})
				})
			})
		}, "objects with AdditionalAttributes are not supported by gRPC"},
		{"form body", func() {
			record := typed()
			Service("records", func() {
				Method("create", func() {
					Payload(record)
					HTTP(func() {
						POST("/")
						FormRequest()
					})
				})
			})
		}, "HTTP form and multipart bodies do not support objects with AdditionalAttributes"},
		{"untagged branch", func() {
			record := typed()
			other := Type("Other", func() {
				Attribute("name", String)
			})
			Service("records", func() {
				Method("show", func() {
					Result(OneOf(record, other), func() {
						Untagged()
					})
				})
			})
		}, "cannot use AdditionalAttributes"},
		{"case-insensitive member", func() {
			usedType(Type("Record", func() {
				Attribute("id", String, func() {
					Meta("struct:tag:json", "id,case:ignore")
				})
				AdditionalAttributes(Int)
			}))
		}, "cannot use JSON option case:ignore"},
		{"result type view", func() {
			rt := ResultType("application/vnd.record", func() {
				TypeName("Record")
				Attributes(func() {
					Attribute("id", String)
					AdditionalAttributes(Int)
				})
				View("default", func() {
					Attribute("id")
				})
			})
			Service("records", func() {
				Method("show", func() {
					Result(rt)
				})
			})
		}, "with AdditionalAttributes cannot declare views"},
		{"Go field collision", func() {
			usedType(Type("Record", func() {
				Attribute("additional_attributes", String)
				AdditionalAttributes(Int)
			}))
		}, "which carries additional attributes"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.ErrorContains(t, expr.RunInvalidDSL(t, tc.design), tc.failure)
		})
	}
}

// TestAdditionalAttributesTypeIdentity checks that hashing and structural
// equality distinguish inline objects that differ only in their typed policy.
func TestAdditionalAttributesTypeIdentity(t *testing.T) {
	arrayOf := func(additional *expr.AttributeExpr, required ...string) *expr.Array {
		element := &expr.AttributeExpr{
			Type:                 &expr.Object{{Name: "id", Attribute: &expr.AttributeExpr{Type: expr.String}}},
			AdditionalAttributes: additional,
		}
		if len(required) > 0 {
			element.Validation = &expr.ValidationExpr{Required: required}
		}
		return &expr.Array{ElemType: element}
	}
	typed := arrayOf(&expr.AttributeExpr{Type: expr.Int})
	for name, other := range map[string]*expr.Array{
		"without a policy":        arrayOf(nil),
		"with another policy":     arrayOf(&expr.AttributeExpr{Type: expr.String}),
		"requiring an undeclared": arrayOf(&expr.AttributeExpr{Type: expr.Int}, "count"),
	} {
		t.Run(name, func(t *testing.T) {
			require.NotEqual(t, typed.Hash(), other.Hash())
			require.False(t, expr.Equal(typed, other))
		})
	}
	same := arrayOf(&expr.AttributeExpr{Type: expr.Int})
	require.Equal(t, typed.Hash(), same.Hash())
	require.True(t, expr.Equal(typed, same))
}

// TestAdditionalAttributesTransportBoundMembers checks that a member bound to
// a path parameter leaves a typed body without becoming a synthesized member,
// and that undeclared body members still cannot reuse its name.
func TestAdditionalAttributesTransportBoundMembers(t *testing.T) {
	expr.RunDSL(t, func() {
		labels := Type("Labels", func() {
			Attribute("id", String)
			AdditionalAttributes(Int)
			Required("id", "count")
		})
		Service("records", func() {
			Method("create", func() {
				Payload(labels)
				HTTP(func() {
					POST("/p/{id}")
				})
			})
		})
	})
	body := expr.Root.API.HTTP.Service("records").Endpoint("create").Body
	declaration := body.TypedObjectDeclaration()
	require.NotNil(t, declaration)
	require.Nil(t, declaration.Find("id"))
	require.Equal(t, []string{"count"}, declaration.AllRequired())
	require.Equal(t, []string{"count"}, body.RequiredAdditionalMembers())
	require.Equal(t, []string{"id"}, body.PhysicalWireNames())
}

// TestAdditionalAttributesExtrasOnlyHTTPBodies checks that HTTP request and
// response bodies keep an object that declares only additional attributes.
func TestAdditionalAttributesExtrasOnlyHTTPBodies(t *testing.T) {
	expr.RunDSL(t, func() {
		Service("records", func() {
			Method("create", func() {
				Payload(func() {
					AdditionalAttributes(Int)
				})
				Result(func() {
					AdditionalAttributes(String)
				})
				HTTP(func() {
					POST("/")
				})
			})
		})
	})
	endpoint := expr.Root.API.HTTP.Service("records").Endpoint("create")
	for name, body := range map[string]*expr.AttributeExpr{
		"request":  endpoint.Body,
		"response": endpoint.Responses[0].Body,
	} {
		t.Run(name, func(t *testing.T) {
			require.NotEqual(t, expr.Empty, body.Type)
			require.NotNil(t, body.TypedObjectDeclaration())
		})
	}
}
