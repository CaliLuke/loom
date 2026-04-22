package openapiv3

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi/v3/testdata/dsls"
	"github.com/CaliLuke/loom/http/codegen/testdata"
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

	sf := newSchemafier(expr.NewRandom("union"))
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

func TestSchemafyUserTypeRegistersComponentReferenceByDefault(t *testing.T) {
	sf := newSchemafier(expr.NewRandom("schemafy-ref"))
	attr := &expr.AttributeExpr{
		Type: &expr.UserTypeExpr{
			AttributeExpr: &expr.AttributeExpr{
				Type: &expr.Object{
					{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}},
				},
			},
			TypeName: "Payload",
		},
	}

	schema := sf.schemafy(attr)

	require.Equal(t, "#/components/schemas/Payload", schema.Ref)
	require.Contains(t, sf.schemas, "Payload")
}

func TestSchemafyUserTypeNoRefSkipsReferenceReuse(t *testing.T) {
	sf := newSchemafier(expr.NewRandom("schemafy-inline"))
	attr := &expr.AttributeExpr{
		Type: &expr.UserTypeExpr{
			AttributeExpr: &expr.AttributeExpr{
				Type: &expr.Object{
					{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}},
				},
			},
			TypeName: "Payload",
		},
	}

	first := sf.schemafy(attr)
	second := sf.schemafy(attr, true)

	require.Equal(t, "#/components/schemas/Payload", first.Ref)
	require.NotEqual(t, first.Ref, second.Ref)
	require.Len(t, sf.schemas, 2)
	require.Contains(t, sf.schemas, "Payload")
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
