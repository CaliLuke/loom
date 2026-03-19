package openapiv3

import (
	"encoding/json"
	"hash/fnv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"goa.design/goa/v3/codegen"
	dsl "goa.design/goa/v3/dsl"
	"goa.design/goa/v3/expr"
	"goa.design/goa/v3/http/codegen/openapi"
	"goa.design/goa/v3/http/codegen/openapi/v3/testdata/dsls"
	"goa.design/goa/v3/http/codegen/testdata"
)

// describes a type for comparison in tests.
type typ struct {
	Type                 string
	Format               string
	Props                []attr
	SkipProps            bool
	AdditionalProperties *additionalPropsType // nil means no additionalProperties check
}

// additionalPropsType describes additionalProperties for testing
type additionalPropsType struct {
	Type  string               // "string", "array", "object", "" (for reference)
	Items *additionalPropsType // for array items
	Ref   string               // for references like "#/components/schemas/MapData"
}

type attr struct {
	Name string
	Val  typ
}

// types mapped by response code.
type rt map[int]typ

// helpers
var (
	tempty  typ
	tstring = typ{Type: "string"}
	tuuid   = typ{Type: "string", Format: "uuid"}
	tbinary = typ{Type: "string", Format: "binary"}
	tint    = typ{Type: "integer"}
	tarray  = typ{Type: "array"}
)

func tobj(attrs ...any) typ {
	res := typ{Type: "object"}
	if len(attrs) == 0 {
		res.SkipProps = true
	}
	for i := 0; i < len(attrs); i += 2 {
		res.Props = append(res.Props, attr{Name: attrs[i].(string), Val: attrs[i+1].(typ)})
	}
	return res
}

func tmap() typ {
	return typ{Type: "object", Props: []attr{{Name: "map", Val: typ{Type: "object"}}}}
}

func (tt typ) Prop(n string) (typ, bool) {
	for _, att := range tt.Props {
		if att.Name == n {
			return att.Val, true
		}
	}
	return tempty, false
}

func TestBuildBodyTypes(t *testing.T) {
	const svcName = "test service"

	cases := []struct {
		Name string
		DSL  func()

		ExpectedType          typ
		ExpectedFormat        string
		ExpectedResponseTypes rt
		ExpectedExtraTypes    map[string]typ
	}{{
		Name: "string_body",
		DSL:  dsls.StringBodyDSL(svcName, "string_body"),

		ExpectedType:          tstring,
		ExpectedResponseTypes: rt{204: tempty},
	}, {
		Name: "alias_string_body",
		DSL:  dsls.AliasStringBodyDSL(svcName, "alias_string_body"),

		ExpectedType:          tuuid,
		ExpectedResponseTypes: rt{204: tempty},
	}, {
		Name: "object_body",
		DSL:  dsls.ObjectBodyDSL(svcName, "object_body"),

		ExpectedType:          tobj("name", tstring, "age", tint),
		ExpectedResponseTypes: rt{204: tempty},
	}, {
		Name: "map_body",
		DSL:  dsls.MapBodyDSL(svcName, "map_body"),

		ExpectedType:          tmap(),
		ExpectedResponseTypes: rt{204: tempty},
	}, {
		Name: "streaming_string_body",
		DSL:  dsls.RequestStreamingStringBody(svcName, "streaming_string_body"),

		ExpectedType:          tstring,
		ExpectedResponseTypes: rt{204: tempty},
	}, {
		Name: "streaming_object_body",
		DSL:  dsls.RequestStreamingObjectBody(svcName, "streaming_object_body"),

		ExpectedType:          tobj("name", tstring, "age", tint),
		ExpectedResponseTypes: rt{204: tempty},
	}, {
		Name: "string_response_body",
		DSL:  dsls.StringResponseBodyDSL(svcName, "string_response_body"),

		ExpectedType:          tempty,
		ExpectedResponseTypes: rt{200: tstring},
	}, {
		Name: "object_response_body",
		DSL:  dsls.ObjectResponseBodyDSL(svcName, "object_response_body"),

		ExpectedType:          tempty,
		ExpectedResponseTypes: rt{200: tobj("name", tstring, "age", tint, "misc", tempty)},
	}, {
		Name: "multi_cookie_response_body",
		DSL:  dsls.MultiCookieResponseBodyDSL(svcName, "multi_cookie_response_body"),

		ExpectedType:          tempty,
		ExpectedResponseTypes: rt{200: tobj("name", tstring)},
	}, {
		Name: "string_streaming_response_body",
		DSL:  dsls.StringStreamingResponseBodyDSL(svcName, "string_streaming_response_body"),

		ExpectedType:          tempty,
		ExpectedResponseTypes: rt{200: tstring},
	}, {
		Name: "object_streaming_response_body",
		DSL:  dsls.ObjectResponseBodyDSL(svcName, "object_streaming_response_body"),

		ExpectedType:          tempty,
		ExpectedResponseTypes: rt{200: tobj("name", tstring, "age", tint, "misc", tempty)},
	}, {
		Name: "string_error_response",
		DSL:  dsls.StringErrorResponseBodyDSL(svcName, "string_error_response"),

		ExpectedType:          tempty,
		ExpectedResponseTypes: rt{204: tempty, 400: tstring},
	}, {
		Name: "object_error_response",
		DSL:  dsls.ObjectErrorResponseBodyDSL(svcName, "object_error_response"),

		ExpectedType:          tempty,
		ExpectedResponseTypes: rt{204: tempty, 400: tobj("name", tstring, "age", tint)},
	}, {
		Name: "forced_type",
		DSL:  dsls.ForcedTypeDSL(svcName, "forced_type"),

		ExpectedType:          tempty,
		ExpectedResponseTypes: rt{204: tempty},
		ExpectedExtraTypes:    map[string]typ{"Forced": tobj("foo", tstring)},
	}, {
		Name: "forced_result_type",
		DSL:  dsls.ForcedResultTypeDSL(svcName, "forced_result_type"),

		ExpectedType:          tempty,
		ExpectedResponseTypes: rt{204: tempty},
		ExpectedExtraTypes:    map[string]typ{"Forced": tobj("foo", tstring)},
	}}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := codegen.RunDSL(t, c.DSL)

			bodies, types := buildBodyTypes(root.API, root.Types, root.ResultTypes)

			svc, ok := bodies[svcName]
			if !ok {
				t.Errorf("bodies does not contain details for service %q", svcName)
				return
			}
			met, ok := svc[c.Name]
			if !ok {
				t.Errorf("bodies does not contain details for method %q", c.Name)
				return
			}
			requestBody := met.RequestBody
			for s, r := range met.ResponseBodies {
				if len(r) != 1 {
					t.Errorf("got %d response bodies for %d, expected 1", len(r), s)
					return
				}
			}

			matchesSchema(t, "request", requestBody, types, c.ExpectedType)
			if len(c.ExpectedResponseTypes) != len(met.ResponseBodies) {
				t.Errorf("got %d response body(ies), expected %d", len(met.ResponseBodies), len(c.ExpectedResponseTypes))
				return
			}
			for s, r := range c.ExpectedResponseTypes {
				if len(met.ResponseBodies[s]) != 1 {
					t.Errorf("got %d response bodies for code %d, expected 1", len(met.ResponseBodies[s]), s)
					return
				}
				matchesSchema(t, "response", met.ResponseBodies[s][0], types, r)
			}
			for name, forced := range c.ExpectedExtraTypes {
				got, ok := types[name]
				if !ok {
					t.Errorf("missing forced type %q", name)
					continue
				}
				matchesSchema(t, "extra type", got, types, forced)
			}
		})
	}
}

func TestSchemafyUsesTaggedUnionExamplesAndEnums(t *testing.T) {
	union := &expr.Union{
		TypeName: "PayloadResult",
		TypeKey:  "kind",
		ValueKey: "data",
		Values: []*expr.NamedAttributeExpr{
			{
				Name: "Single",
				Attribute: &expr.AttributeExpr{
					Type: &expr.Object{
						{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}},
					},
					Meta: expr.MetaExpr{"oneof:type:tag": []string{"single"}},
				},
			},
			{
				Name: "Batch",
				Attribute: &expr.AttributeExpr{
					Type: &expr.Object{
						{Name: "items", Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}}},
					},
					Meta: expr.MetaExpr{"oneof:type:tag": []string{"batch"}},
				},
			},
		},
	}
	attr := &expr.AttributeExpr{
		Type: union,
		UserExamples: []*expr.ExampleExpr{{
			Summary: "default",
			Value:   map[string]any{"name": "alice"},
		}},
	}

	sf := newSchemafier(expr.NewRandom("union"), false)
	schema := sf.schemafy(attr)

	assertUnionSchema(t, schema, sf.schemas, union.GetTypeKey(), union.GetValueKey(), []string{"batch", "single"})
	example, ok := schema.Example.(map[string]any)
	if !ok {
		t.Fatalf("expected map example, got %T", schema.Example)
	}
	if example[union.GetTypeKey()] != "single" {
		t.Errorf("got type example %#v, expected %q", example[union.GetTypeKey()], "single")
	}
	if _, ok := example[union.GetValueKey()].(map[string]any); !ok {
		t.Errorf("got value example %#v, expected nested object", example[union.GetValueKey()])
	}
}

func TestBuildBodyTypesUnionIncludesDiscriminatorMappingsForRequestAndResponse(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		textResult := dsl.Type("TextResult", func() {
			dsl.Attribute("text", dsl.String)
			dsl.Required("text")
		})
		jsonResult := dsl.Type("JSONResult", func() {
			dsl.Attribute("message", dsl.String)
			dsl.Required("message")
		})
		dsl.Service("union-service", func() {
			dsl.Method("show", func() {
				dsl.Payload(dsl.OneOf(textResult, jsonResult))
				dsl.Result(dsl.OneOf(textResult, jsonResult))
				dsl.HTTP(func() {
					dsl.POST("/")
				})
			})
		})
	})

	bodies, types := buildBodyTypes(root.API, root.Types, root.ResultTypes)
	methodBodies := bodies["union-service"]["show"]
	requestSchema := derefSchema(t, methodBodies.RequestBody, types)
	responseSchema := derefSchema(t, methodBodies.ResponseBodies[200][0], types)

	assertUnionSchema(t, requestSchema, types, "type", "value", []string{"JSONResult", "TextResult"})
	assertUnionSchema(t, responseSchema, types, "type", "value", []string{"JSONResult", "TextResult"})
}

func TestBuildBodyTypesUnionSupportsCustomKeysAndStableEnvelopeRefs(t *testing.T) {
	root := codegen.RunDSL(t, testdata.PayloadBodyUnionCustomKeysDSL)

	bodies, types := buildBodyTypes(root.API, root.Types, root.ResultTypes)
	requestSchema := derefSchema(t, bodies["ServiceBodyUnionCustomKeys"]["MethodBodyUnionCustomKeys"].RequestBody, types)
	unionSchema := requestSchema.Properties["Values"]

	assertUnionSchema(t, unionSchema, types, "kind", "data", []string{"Int", "String"})
}

func TestBuildBodyTypesUnionRenamedTypesKeepDeclaredDiscriminators(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		alpha := dsl.Type("AlphaPayload", func() {
			dsl.Meta("openapi:typename", "RenamedAlphaPayload")
			dsl.Meta("oneof:type:tag", "AlphaPayload")
			dsl.Attribute("alpha", dsl.String)
			dsl.Required("alpha")
		})
		beta := dsl.Type("BetaPayload", func() {
			dsl.Meta("openapi:typename", "RenamedBetaPayload")
			dsl.Meta("oneof:type:tag", "BetaPayload")
			dsl.Attribute("beta", dsl.String)
			dsl.Required("beta")
		})
		dsl.Service("renamed-union-service", func() {
			dsl.Method("show", func() {
				dsl.Payload(dsl.OneOf(alpha, beta))
				dsl.HTTP(func() {
					dsl.POST("/")
				})
			})
		})
	})

	bodies, types := buildBodyTypes(root.API, root.Types, root.ResultTypes)
	requestSchema := derefSchema(t, bodies["renamed-union-service"]["show"].RequestBody, types)

	require.NotNil(t, requestSchema.Discriminator)
	require.Contains(t, requestSchema.Discriminator.Mapping, "AlphaPayload")
	require.Contains(t, requestSchema.Discriminator.Mapping, "BetaPayload")
	require.NotContains(t, requestSchema.Discriminator.Mapping, "RenamedAlphaPayload")
	require.NotContains(t, requestSchema.Discriminator.Mapping, "RenamedBetaPayload")
}

func TestBuildBodyTypesDeduplicatesGeneratedRequestBodySchemasByStructure(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.Service("dedup-service", func() {
			dsl.Method("first", func() {
				dsl.Payload(func() {
					dsl.Attribute("name", dsl.String)
					dsl.Required("name")
				})
				dsl.HTTP(func() {
					dsl.POST("/first")
				})
			})
			dsl.Method("second", func() {
				dsl.Payload(func() {
					dsl.Attribute("name", dsl.String)
					dsl.Required("name")
				})
				dsl.HTTP(func() {
					dsl.POST("/second")
				})
			})
		})
	})

	bodies, types := buildBodyTypes(root.API, root.Types, root.ResultTypes)
	firstRef := bodies["dedup-service"]["first"].RequestBody.Ref
	secondRef := bodies["dedup-service"]["second"].RequestBody.Ref

	require.Equal(t, firstRef, secondRef)
	require.Len(t, types, 1)
	matchesSchema(t, "shared request body", types[nameFromRef(firstRef)], types, tobj("name", tstring))
}

func TestBuildBodyTypesDeduplicatesRepeatedUnionEnvelopeSchemas(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		alpha := dsl.Type("Alpha", func() {
			dsl.Attribute("alpha", dsl.String)
			dsl.Required("alpha")
		})
		beta := dsl.Type("Beta", func() {
			dsl.Attribute("beta", dsl.String)
			dsl.Required("beta")
		})
		dsl.Service("union-dedup-service", func() {
			for _, name := range []string{"first", "second"} {
				methodName := name
				dsl.Method(methodName, func() {
					dsl.Payload(dsl.OneOf(alpha, beta))
					dsl.HTTP(func() {
						dsl.POST("/" + methodName)
					})
				})
			}
		})
	})

	bodies, types := buildBodyTypes(root.API, root.Types, root.ResultTypes)
	firstSchema := derefSchema(t, bodies["union-dedup-service"]["first"].RequestBody, types)
	secondSchema := derefSchema(t, bodies["union-dedup-service"]["second"].RequestBody, types)

	require.Equal(t, firstSchema.Discriminator.Mapping, secondSchema.Discriminator.Mapping)
	require.Len(t, firstSchema.Discriminator.Mapping, 2)
	require.Len(t, countEnvelopeSchemas(types), 2)
}

func TestBuildBodyTypesKeepsExplicitOpenAPITypenamesDistinct(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.Service("named-service", func() {
			dsl.Method("foo", func() {
				dsl.Payload(func() {
					dsl.Meta("openapi:typename", "FooPayload")
					dsl.Attribute("value", dsl.String)
					dsl.Required("value")
				})
				dsl.HTTP(func() {
					dsl.POST("/foo")
				})
			})
			dsl.Method("bar", func() {
				dsl.Payload(func() {
					dsl.Meta("openapi:typename", "BarPayload")
					dsl.Attribute("value", dsl.String)
					dsl.Required("value")
				})
				dsl.HTTP(func() {
					dsl.POST("/bar")
				})
			})
		})
	})

	bodies, _ := buildBodyTypes(root.API, root.Types, root.ResultTypes)
	fooRef := bodies["named-service"]["foo"].RequestBody.Ref
	barRef := bodies["named-service"]["bar"].RequestBody.Ref

	require.NotEqual(t, fooRef, barRef)
	require.Equal(t, "#/components/schemas/FooPayload", fooRef)
	require.Equal(t, "#/components/schemas/BarPayload", barRef)
}

func TestBuildBodyTypesUsesExplicitOpenAPITypenameAsCanonicalBodyComponentName(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		body := dsl.Type("AuthSessionResponseBodyType", func() {
			dsl.Meta("openapi:typename", "AuthSessionResponseBody")
			dsl.Attribute("authenticated", dsl.Boolean)
			dsl.Required("authenticated")
		})
		dsl.Service("auth-service", func() {
			dsl.Method("session", func() {
				dsl.Result(body)
				dsl.HTTP(func() {
					dsl.GET("/session")
					dsl.Response(200, func() {
						dsl.Body(body)
					})
				})
			})
		})
	})

	bodies, types := buildBodyTypes(root.API, root.Types, root.ResultTypes)
	responseRef := bodies["auth-service"]["session"].ResponseBodies[200][0].Ref

	require.Equal(t, "#/components/schemas/AuthSessionResponseBody", responseRef)
	require.Contains(t, types, "AuthSessionResponseBody")
	for name := range types {
		require.NotContains(t, name, "AuthSessionResponseBody_")
	}
}

func TestBuildBodyTypesPanicsOnConflictingExplicitOpenAPITypename(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		first := dsl.Type("AuthSessionResponseBodyA", func() {
			dsl.Meta("openapi:typename", "AuthSessionResponseBody")
			dsl.Attribute("authenticated", dsl.Boolean)
			dsl.Required("authenticated")
		})
		second := dsl.Type("AuthSessionResponseBodyB", func() {
			dsl.Meta("openapi:typename", "AuthSessionResponseBody")
			dsl.Attribute("authenticated", dsl.Boolean)
			dsl.Attribute("user_id", dsl.Int64)
			dsl.Required("authenticated", "user_id")
		})
		dsl.Service("auth-conflict-service", func() {
			dsl.Method("session", func() {
				dsl.Result(first)
				dsl.HTTP(func() {
					dsl.GET("/session")
					dsl.Response(200, func() {
						dsl.Body(first)
					})
				})
			})
			dsl.Method("profile", func() {
				dsl.Result(second)
				dsl.HTTP(func() {
					dsl.GET("/profile")
					dsl.Response(200, func() {
						dsl.Body(second)
					})
				})
			})
		})
	})

	require.PanicsWithValue(t,
		"openapi: explicit component name \"AuthSessionResponseBody\" is claimed by multiple different schemas; use distinct Meta(\"openapi:typename\", ...) values",
		func() {
			buildBodyTypes(root.API, root.Types, root.ResultTypes)
		},
	)
}

func TestBuildBodyTypesClosedObjectModeClosesObjectsAndLeavesMapsOpen(t *testing.T) {
	root := codegen.RunDSL(t, testdata.OpenAPIClosedObjectsDSL)

	bodies, types := buildBodyTypes(root.API, root.Types, root.ResultTypes)

	objectSchema := derefSchema(t, bodies["closedObjectsService"]["object"].RequestBody, types)
	require.Equal(t, false, objectSchema.AdditionalProperties)

	addressSchema, ok := objectSchema.Properties["address"]
	require.True(t, ok)
	addressSchema = derefSchema(t, addressSchema, types)
	require.Equal(t, false, addressSchema.AdditionalProperties)

	mapSchema := derefSchema(t, bodies["closedObjectsService"]["map_object"].RequestBody, types)
	labelsSchema, ok := mapSchema.Properties["labels"]
	require.True(t, ok)
	labelsSchema = derefSchema(t, labelsSchema, types)
	require.IsType(t, &openapi.Schema{}, labelsSchema.AdditionalProperties)

	unionSchema := derefSchema(t, bodies["closedObjectsService"]["union_object"].RequestBody, types)
	require.Equal(t, false, unionSchema.UnevaluatedProperties)
}

func TestInitExamplesCanonicalizesMultipleUnionExamples(t *testing.T) {
	union := &expr.Union{
		TypeKey:  "kind",
		ValueKey: "data",
		Values: []*expr.NamedAttributeExpr{
			{
				Name: "Single",
				Attribute: &expr.AttributeExpr{
					Type: &expr.Object{
						{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}},
					},
					Meta: expr.MetaExpr{"oneof:type:tag": []string{"single"}},
				},
			},
			{
				Name: "Batch",
				Attribute: &expr.AttributeExpr{
					Type: &expr.Object{
						{Name: "items", Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}}},
					},
					Meta: expr.MetaExpr{"oneof:type:tag": []string{"batch"}},
				},
			},
		},
	}
	attr := &expr.AttributeExpr{
		Type: union,
		UserExamples: []*expr.ExampleExpr{
			{Summary: "single", Value: map[string]any{"name": "alice"}},
			{Summary: "batch", Value: map[string]any{"items": []any{"a", "b"}}},
		},
	}

	mt := &MediaType{}
	initExamples(mt, attr, expr.NewRandom("union"), false)

	if len(mt.Examples) != 2 {
		t.Fatalf("got %d examples, expected 2", len(mt.Examples))
	}
	single := mt.Examples["single"].Value.Value.(map[string]any)
	if single["kind"] != "single" {
		t.Errorf("got single example kind %#v, expected %q", single["kind"], "single")
	}
	batch := mt.Examples["batch"].Value.Value.(map[string]any)
	if batch["kind"] != "batch" {
		t.Errorf("got batch example kind %#v, expected %q", batch["kind"], "batch")
	}
}

func assertUnionSchema(t *testing.T, schema *openapi.Schema, types map[string]*openapi.Schema, typeKey, valueKey string, wantTags []string) {
	t.Helper()

	require.NotNil(t, schema)
	require.NotNil(t, schema.Discriminator)
	require.Equal(t, typeKey, schema.Discriminator.PropertyName)
	require.Len(t, schema.OneOf, len(wantTags))
	require.Len(t, schema.Discriminator.Mapping, len(wantTags))

	expectedTags := make(map[string]struct{}, len(wantTags))
	for _, tag := range wantTags {
		expectedTags[tag] = struct{}{}
	}

	oneOfRefs := make(map[string]struct{}, len(schema.OneOf))
	for _, branch := range schema.OneOf {
		require.NotNil(t, branch)
		require.NotEmpty(t, branch.Ref)
		oneOfRefs[branch.Ref] = struct{}{}
	}

	for tag := range expectedTags {
		ref, ok := schema.Discriminator.Mapping[tag]
		require.Truef(t, ok, "missing discriminator mapping for %q", tag)
		if _, ok := oneOfRefs[ref]; !ok {
			t.Fatalf("mapping ref %q for tag %q is not present in oneOf", ref, tag)
		}
		branchSchema, ok := types[nameFromRef(ref)]
		require.Truef(t, ok, "missing branch schema for %q", ref)
		require.Equal(t, string(openapi.Object), string(branchSchema.Type))
		require.ElementsMatch(t, []string{typeKey, valueKey}, branchSchema.Required)

		typeSchema, ok := branchSchema.Properties[typeKey]
		require.Truef(t, ok, "missing discriminator property %q", typeKey)
		require.Equal(t, string(openapi.String), string(typeSchema.Type))
		require.Equal(t, []any{tag}, typeSchema.Enum)

		valueSchema, ok := branchSchema.Properties[valueKey]
		require.Truef(t, ok, "missing value property %q", valueKey)
		require.NotNil(t, valueSchema)
	}
}

func derefSchema(t *testing.T, schema *openapi.Schema, types map[string]*openapi.Schema) *openapi.Schema {
	t.Helper()

	require.NotNil(t, schema)
	if schema.Ref == "" {
		return schema
	}
	resolved, ok := types[nameFromRef(schema.Ref)]
	require.Truef(t, ok, "missing schema for ref %q", schema.Ref)
	return resolved
}

func matchesSchema(t *testing.T, ctx string, s *openapi.Schema, types map[string]*openapi.Schema, tt typ) {
	matchesSchemaWithPrefix(t, ctx, s, types, tt, "")
}
func matchesSchemaWithPrefix(t *testing.T, ctx string, s *openapi.Schema, types map[string]*openapi.Schema, tt typ, prefix string) {
	if s == nil {
		if tt.Type != "" {
			t.Errorf("%s: %sgot type Empty, expected %q", ctx, prefix, tt.Type)
		}
		return
	}
	if s.Ref != "" {
		var ok bool
		s, ok = types[nameFromRef(s.Ref)]
		if !ok {
			t.Errorf("could not find type for ref %q", s.Ref)
			return
		}
	}
	if tt.Type != string(s.Type) {
		t.Errorf("%s: %sgot type %q, expected %q", ctx, prefix, s.Type, tt.Type)
	}
	if tt.Format != "" {
		if s.Format != tt.Format {
			t.Errorf("%s: %sgot format %q, expected %q", ctx, prefix, s.Format, tt.Format)
		}
	}
	if tt.Type == "object" {
		if tt.SkipProps {
			return
		}
		for n, v := range s.Properties {
			p, ok := tt.Prop(n)
			if !ok {
				t.Errorf("%s: %sgot unexpected field %q", ctx, prefix, n)
				continue
			}
			matchesSchemaWithPrefix(t, ctx, v, types, p, n+": ")
		}

		// Check additionalProperties
		if tt.AdditionalProperties != nil {
			validateAdditionalProperties(t, ctx, s.AdditionalProperties, types, tt.AdditionalProperties, prefix)
		}
	}
}

func TestMapTypes(t *testing.T) {
	svcName := "test-service"

	testCases := []struct {
		Name     string
		DSL      func()
		Expected typ
	}{
		{
			Name: "map_int_array_string",
			DSL:  dsls.MapIntKeyBodyDSL(svcName, "map_int_array_string"),
			Expected: typ{
				Type: "object",
				Props: []attr{{Name: "intmap", Val: typ{
					Type: "object",
					AdditionalProperties: &additionalPropsType{
						Type:  "array",
						Items: &additionalPropsType{Type: "string"},
					},
				}}},
			},
		},
		{
			Name: "map_int_array_object",
			DSL:  dsls.MapIntKeyObjectBodyDSL(svcName, "map_int_array_object"),
			Expected: typ{
				Type: "object",
				Props: []attr{{Name: "intmap", Val: typ{
					Type: "object",
					AdditionalProperties: &additionalPropsType{
						Type:  "array",
						Items: &additionalPropsType{Ref: "#/components/schemas/MapData"},
					},
				}}},
			},
		},
		{
			Name: "map_int_string",
			DSL:  dsls.MapIntKeyStringBodyDSL(svcName, "map_int_string"),
			Expected: typ{
				Type: "object",
				Props: []attr{{Name: "intmap", Val: typ{
					Type:                 "object",
					AdditionalProperties: &additionalPropsType{Type: "string"},
				}}},
			},
		},
		{
			Name: "map_int_object_direct",
			DSL:  dsls.MapIntKeyObjectDirectBodyDSL(svcName, "map_int_object_direct"),
			Expected: typ{
				Type: "object",
				Props: []attr{{Name: "intmap", Val: typ{
					Type:                 "object",
					AdditionalProperties: &additionalPropsType{Ref: "#/components/schemas/MapData"},
				}}},
			},
		},
		{
			Name: "map_string_int",
			DSL:  dsls.MapStringKeyIntBodyDSL(svcName, "map_string_int"),
			Expected: typ{
				Type: "object",
				Props: []attr{{Name: "stringmap", Val: typ{
					Type:                 "object",
					AdditionalProperties: &additionalPropsType{Type: "integer"},
				}}},
			},
		},
		{
			Name: "map_string_object_direct",
			DSL:  dsls.MapStringKeyObjectDirectBodyDSL(svcName, "map_string_object_direct"),
			Expected: typ{
				Type: "object",
				Props: []attr{{Name: "stringmap", Val: typ{
					Type:                 "object",
					AdditionalProperties: &additionalPropsType{Ref: "#/components/schemas/MapData"},
				}}},
			},
		},
		{
			Name: "map_string_array_object",
			DSL:  dsls.MapStringKeyArrayObjectBodyDSL(svcName, "map_string_array_object"),
			Expected: typ{
				Type: "object",
				Props: []attr{{Name: "stringmap", Val: typ{
					Type: "object",
					AdditionalProperties: &additionalPropsType{
						Type:  "array",
						Items: &additionalPropsType{Ref: "#/components/schemas/MapData"},
					},
				}}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// Build the OpenAPI spec
			root := codegen.RunDSL(t, tc.DSL)
			bodies, types := buildBodyTypes(root.API, root.Types, root.ResultTypes)

			// Find the service and method
			svcBodies, ok := bodies[svcName]
			if !ok {
				t.Fatalf("Could not find service %s in bodies", svcName)
			}

			methodBody, ok := svcBodies[tc.Name]
			if !ok {
				t.Fatalf("Could not find method %s in service bodies", tc.Name)
			}

			// Get the request body schema
			requestBodyRef := methodBody.RequestBody.Ref
			if requestBodyRef == "" {
				t.Fatal("Expected request body reference")
			}

			requestBodyTypeName := nameFromRef(requestBodyRef)
			requestBodySchema, ok := types[requestBodyTypeName]
			if !ok {
				t.Fatalf("Could not find request body type %s", requestBodyTypeName)
			}

			// Validate the schema
			matchesSchema(t, tc.Name, requestBodySchema, types, tc.Expected)
		})
	}
}

func validateAdditionalProperties(t *testing.T, ctx string, addProps any, types map[string]*openapi.Schema, expected *additionalPropsType, prefix string) {
	if addProps == nil {
		t.Errorf("%s: %sexpected additionalProperties to be set", ctx, prefix)
		return
	}

	// Check if additionalProperties is a schema
	schema, ok := addProps.(*openapi.Schema)
	if !ok {
		t.Errorf("%s: %sexpected additionalProperties to be schema, got %T", ctx, prefix, addProps)
		return
	}

	validateAdditionalPropsSchema(t, ctx, schema, types, expected, prefix+"additionalProperties: ")
}

func validateAdditionalPropsSchema(t *testing.T, ctx string, schema *openapi.Schema, types map[string]*openapi.Schema, expected *additionalPropsType, prefix string) {
	// Handle reference case
	if expected.Ref != "" {
		if schema.Ref == "" {
			t.Errorf("%s: %sexpected reference to %s, but got inline schema", ctx, prefix, expected.Ref)
			return
		}
		if schema.Ref != expected.Ref {
			t.Errorf("%s: %sexpected reference %s, got %s", ctx, prefix, expected.Ref, schema.Ref)
		}
		return
	}

	// Resolve reference if present
	if schema.Ref != "" {
		typeName := nameFromRef(schema.Ref)
		resolvedSchema, ok := types[typeName]
		if !ok {
			t.Errorf("%s: %scould not resolve reference %s", ctx, prefix, schema.Ref)
			return
		}
		schema = resolvedSchema
	}

	// Check type
	if string(schema.Type) != expected.Type {
		t.Errorf("%s: %sexpected type %s, got %s", ctx, prefix, expected.Type, schema.Type)
	}

	// Check array items if expected
	if expected.Items != nil {
		if schema.Items == nil {
			t.Errorf("%s: %sexpected array items to be set", ctx, prefix)
		} else {
			validateAdditionalPropsSchema(t, ctx, schema.Items, types, expected.Items, prefix+"items: ")
		}
	}
}

func TestTypesOnlyDifferByEnum(t *testing.T) {
	root := codegen.RunDSL(t, dsls.StringEnumBodyDSL())

	bodies, types := buildBodyTypes(root.API, root.Types, root.ResultTypes)

	svc1, ok := bodies["svc_enum_1"]
	if !ok {
		t.Errorf("bodies does not contain details for service %q", "svc_enum_1")
		return
	}
	svc2, ok := bodies["svc_enum_2"]
	if !ok {
		t.Errorf("bodies does not contain details for service %q", "svc_enum_2")
		return
	}

	svc1MethodRB := svc1["method_enum"].RequestBody.Ref
	svc2MethodRB := svc2["method_enum"].RequestBody.Ref

	if svc1MethodRB == svc2MethodRB {
		t.Errorf("expected different refs, got %q", svc1MethodRB)

		name := nameFromRef(svc1MethodRB)
		derefed := types[name]
		jsoned, _ := json.Marshal(derefed)
		t.Errorf("shared referenced type (%s) was: %v", name, string(jsoned))
		return
	}
}

func TestSchemafierUniquifyUsesStableHashSuffix(t *testing.T) {
	sf := newSchemafier(expr.NewRandom("test"), false)
	sf.schemas["CreateThreadRequest"] = openapi.NewSchema()
	sf.schemas["CreateThreadRequest2"] = openapi.NewSchema()

	name := sf.uniquify("CreateThreadRequest", 0x1234abcd)
	if name != "CreateThreadRequest_000000001234abcd" {
		t.Fatalf("got %q, expected deterministic hash suffix", name)
	}

	name = sf.uniquify("CreateThreadRequest2", 0x99)
	if name != "CreateThreadRequest2_0000000000000099" {
		t.Fatalf("got %q, expected original trailing digits to be preserved", name)
	}

	name = sf.uniquify("FreshName", 0xbeef)
	if name != "FreshName" {
		t.Fatalf("got %q, expected unsuffixed fresh name", name)
	}
}

func TestClaimExplicitNamePanicsOnConflictingSchema(t *testing.T) {
	sf := newSchemafier(expr.NewRandom("test"), false)
	sf.schemaHashes["AuthSessionResponseBody"] = 0x1

	require.PanicsWithValue(t,
		"openapi: explicit component name \"AuthSessionResponseBody\" is claimed by multiple different schemas; use distinct Meta(\"openapi:typename\", ...) values",
		func() {
			sf.claimExplicitName("AuthSessionResponseBody", 0x2)
		},
	)
}

func TestHashAttribute(t *testing.T) {
	type (
		testAttr struct {
			name string
			att  *expr.AttributeExpr
		}

		hashBehavior int

		testGroup struct {
			name     string
			attrs    []testAttr
			behavior hashBehavior
		}
	)

	const (
		uniqueHashes hashBehavior = iota
		identicalHashes
	)

	var (
		metaNotGenerate = expr.MetaExpr{"openapi:generate": []string{"false"}}
		metaEmpty       = expr.MetaExpr{}
	)

	cases := []testGroup{
		{
			name:     "Primitive types",
			behavior: uniqueHashes,
			attrs: []testAttr{
				{name: "bool", att: &expr.AttributeExpr{Type: expr.Boolean}},
				{name: "int", att: &expr.AttributeExpr{Type: expr.Int}},
				{name: "int32", att: &expr.AttributeExpr{Type: expr.Int32}},
				{name: "int64", att: &expr.AttributeExpr{Type: expr.Int64}},
				{name: "uint", att: &expr.AttributeExpr{Type: expr.UInt}},
				{name: "uint32", att: &expr.AttributeExpr{Type: expr.UInt32}},
				{name: "uint64", att: &expr.AttributeExpr{Type: expr.UInt64}},
				{name: "float32", att: &expr.AttributeExpr{Type: expr.Float32}},
				{name: "float64", att: &expr.AttributeExpr{Type: expr.Float64}},
				{name: "string", att: &expr.AttributeExpr{Type: expr.String}},
				{name: "bytes", att: &expr.AttributeExpr{Type: expr.Bytes}},
				{name: "any", att: &expr.AttributeExpr{Type: expr.Any}},
			},
		}, {
			name:     "Collection types",
			behavior: uniqueHashes,
			attrs: []testAttr{
				{name: "array-bool", att: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Boolean}}}},
				{name: "array-int", att: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Int}}}},
				{name: "map-str-int", att: &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.Int}}}},
				{name: "map-str-str", att: &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}}},
			},
		}, {
			name:     "Objects with validation rules",
			behavior: uniqueHashes,
			attrs: []testAttr{
				{name: "no-validation", att: newObj("foo", false)},
				{name: "required-validation", att: newObj("foo", true)},
				{name: "pattern-validation", att: &expr.AttributeExpr{
					Type: expr.String,
					Validation: &expr.ValidationExpr{
						Pattern: "^[a-z]+$",
					},
				}},
				{name: "enum-validation", att: &expr.AttributeExpr{
					Type: expr.String,
					Validation: &expr.ValidationExpr{
						Values: []any{"foo", "bar"},
					},
				}},
			},
		}, {
			name:     "Result types with different views",
			behavior: uniqueHashes,
			attrs: []testAttr{
				{name: "no-view", att: newRT("id", newObj("foo", true))},
				{name: "default-view", att: newRTWithView("id", newObj("foo", true), "default")},
				{name: "tiny-view", att: newRTWithView("id", newObj("foo", true), "tiny")},
			},
		}, {
			name:     "Objects with openapi:generate:false metadata",
			behavior: identicalHashes,
			attrs: []testAttr{
				{name: "obj-with-skipped-field", att: newObj2Meta("foo", "bar", expr.String, expr.String, metaEmpty, metaNotGenerate)},
				{name: "obj-without-skipped-field", att: newObj("foo", false)},
			},
		}, {
			name:     "Complex map types",
			behavior: uniqueHashes,
			attrs: []testAttr{
				{name: "map-int-array", att: &expr.AttributeExpr{Type: &expr.Map{
					KeyType:  &expr.AttributeExpr{Type: expr.Int},
					ElemType: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}},
				}}},
				{name: "map-array-int", att: &expr.AttributeExpr{Type: &expr.Map{
					KeyType:  &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}},
					ElemType: &expr.AttributeExpr{Type: expr.Int},
				}}},
			},
		}, {
			name:     "Nested user types",
			behavior: uniqueHashes,
			attrs: []testAttr{
				{name: "single-nest", att: newUserType("foo", newObj("bar", false))},
				{name: "double-nest", att: newUserType("foo", newUserType("bar", newObj("baz", false)))},
			},
		}, {
			name:     "Recursive types",
			behavior: identicalHashes,
			attrs: []testAttr{
				{name: "recursive-1", att: newRecursiveType("foo")},
				{name: "recursive-2", att: newRecursiveType("foo")},
			},
		},
	}

	h := fnv.New64()
	sf := newSchemafier(expr.NewRandom("test"), false)

	for _, group := range cases {
		t.Run(group.name, func(t *testing.T) {
			seen := make(map[uint64][]string)

			// Collect all hashes in this group
			for _, attr := range group.attrs {
				hash := sf.hashAttribute(attr.att, h)
				seen[hash] = append(seen[hash], attr.name)
			}

			switch group.behavior {
			case uniqueHashes:
				// Verify all hashes are different
				for hash, names := range seen {
					if len(names) > 1 {
						t.Errorf("expected unique hashes but got collision between %v (hash: %d)",
							names, hash)
					}
				}
			case identicalHashes:
				// Verify all hashes are the same
				if len(seen) > 1 {
					t.Errorf("expected identical hashes but got different ones: %v", seen)
				}
			}
		})
	}
}

func newObj(n string, req bool) *expr.AttributeExpr {
	attr := &expr.AttributeExpr{
		Type:       &expr.Object{{Name: n, Attribute: &expr.AttributeExpr{Type: expr.String}}},
		Validation: &expr.ValidationExpr{},
	}
	if req {
		attr.Validation.Required = []string{n}
	}
	return attr
}

func newObj2Meta(n, o string, t, u expr.DataType, l, m expr.MetaExpr, reqs ...string) *expr.AttributeExpr {
	attr := &expr.AttributeExpr{
		Type: &expr.Object{
			{Name: n, Attribute: &expr.AttributeExpr{Type: t, Meta: l}},
			{Name: o, Attribute: &expr.AttributeExpr{Type: u, Meta: m}},
		},
		Validation: &expr.ValidationExpr{},
	}
	attr.Validation.Required = append(attr.Validation.Required, reqs...)
	return attr
}

func newRT(id string, att *expr.AttributeExpr) *expr.AttributeExpr {
	return &expr.AttributeExpr{
		Type: &expr.ResultTypeExpr{
			Identifier: id,
			UserTypeExpr: &expr.UserTypeExpr{
				AttributeExpr: att,
			},
		},
	}
}

// Helper function for result types with views
func newRTWithView(id string, att *expr.AttributeExpr, view string) *expr.AttributeExpr {
	rt := newRT(id, att)
	rt.Type.(*expr.ResultTypeExpr).Meta = expr.MetaExpr{
		expr.ViewMetaKey: []string{view},
	}
	return rt
}

// Helper function for user types
func newUserType(name string, att *expr.AttributeExpr) *expr.AttributeExpr {
	return &expr.AttributeExpr{
		Type: &expr.UserTypeExpr{
			AttributeExpr: att,
			TypeName:      name,
		},
	}
}

// Helper function for recursive types
func newRecursiveType(name string) *expr.AttributeExpr {
	// Create a user type that references itself
	ut := &expr.UserTypeExpr{
		TypeName: name,
	}
	att := &expr.AttributeExpr{
		Type: &expr.Object{
			&expr.NamedAttributeExpr{
				Name: "self",
				Attribute: &expr.AttributeExpr{
					Type: ut,
				},
			},
		},
	}
	ut.AttributeExpr = att
	return &expr.AttributeExpr{Type: ut}
}

// nameFromRef does the reverse of toRef: it returns the type name from its
// JSON Schema reference.
func nameFromRef(ref string) string {
	elems := strings.Split(ref, "/")
	return elems[len(elems)-1]
}

func countEnvelopeSchemas(types map[string]*openapi.Schema) []string {
	var names []string
	for name := range types {
		if strings.Contains(name, "Envelope") {
			names = append(names, name)
		}
	}
	return names
}
