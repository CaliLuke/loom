package codegen

import (
	"encoding/json/v2"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestHTTPValueCarriersUseServiceOwnership(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.Service("values", func() {
			dsl.Method("show", func() {
				dsl.Payload(func() {
					dsl.Attribute("body", dsl.Float32)
					dsl.Attribute("trace", dsl.String)
					dsl.Example(map[string]any{"body": float32(0.1), "trace": "source"})
				})
				dsl.Result(dsl.String)
				dsl.HTTP(func() {
					dsl.POST("/")
					dsl.Body("body")
					dsl.Header("trace")
					dsl.Response(dsl.StatusOK, func() { dsl.ContentType("application/octet-stream") })
				})
			})
		})
	})
	services := service.NewServicesData(root)
	method := services.Get("values").Method("show")
	prepared, err := representation.PrepareService(root.API.HTTP.Services[0], services.Get("values"))
	require.NoError(t, err)
	ir := prepared.Endpoints[0]
	require.Same(t, method.PayloadValue, ir.Request.BodyValue.Source)
	require.Same(t, method.PayloadValue, ir.Request.Headers[0].Value.Source)
	require.Equal(t, []string{"body"}, ir.Request.BodyValue.Selection)
	require.Equal(t, expr.ValueCodecJSON, ir.Response.Responses[0].BodyValue.Codec, "media type does not select the codec")
	plan := representation.BuildValuePlan(ir.Request.BodyValue, ir.Request.Body, httpContext(codegen.NewNameScope(), true, false), expr.ValuePlanRuntime)
	require.NoError(t, plan.Error)
	projected := method.PayloadValue.Context.ProjectJSON(method.PayloadValue.Example, plan.Plan)
	require.Equal(t, expr.ProjectionEmitted, projected.Outcome())
}

func TestHTTPValueRetainedPlans(t *testing.T) {
	for _, test := range []struct {
		name   string
		design func()
	}{
		{"type identities", testdata.TypeIdentityDSL},
		{"selected views", testdata.ExplicitBodyUserResultMultipleViewsDSL},
		{"streaming payload", testdata.StreamingPayloadDSL},
		{"mixed results", testdata.MixedResultsDSL},
		{"error headers", sharedErrorHeaderDSL},
		{"inherited API error", testdata.APIErrorResponseDSL},
		{"problem error", testdata.DefaultErrorResponseDSL},
		{"form", testdata.PayloadFormBodyObjectDSL},
		{"multipart", testdata.PayloadMultipartUserTypeDSL},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := RunHTTPDSL(t, test.design)
			services := CreateHTTPServices(root)
			checked := 0
			check := func(body *TypeData) {
				if body == nil {
					return
				}
				checked++
				require.NotNil(t, body.Value, body.Name)
				require.NoError(t, body.Value.Error, body.Name)
				require.NotNil(t, body.Value.Source, body.Name)
			}
			for _, svc := range root.API.HTTP.Services {
				for _, endpoint := range services.Get(svc.Name()).Endpoints {
					require.NotNil(t, endpoint.valueTransport)
					if endpoint.Payload != nil && endpoint.Payload.Request != nil {
						check(endpoint.Payload.Request.ServerBody)
						check(endpoint.Payload.Request.ClientBody)
					}
					if endpoint.Result != nil {
						for _, response := range endpoint.Result.Responses {
							for _, body := range response.ServerBody {
								check(body)
							}
							check(response.ClientBody)
						}
					}
					for _, group := range endpoint.Errors {
						for _, failure := range group.Errors {
							for _, body := range failure.Response.ServerBody {
								check(body)
							}
							check(failure.Response.ClientBody)
						}
					}
					if endpoint.ServerWebSocket != nil {
						check(endpoint.ServerWebSocket.Payload)
					}
				}
			}
			require.Positive(t, checked)
		})
	}
}

func TestHTTPValuePlansKeepBranchDecoderOwnership(t *testing.T) {
	endpoint := firstEndpointData(t, func() {
		closed := dsl.Type("Closed", func() {
			dsl.Attribute("a", dsl.String)
			dsl.Meta("openapi:additionalProperties", "false")
		})
		other := dsl.Type("Other", func() {
			dsl.Attribute("b", dsl.String)
			dsl.Required("b")
		})
		dsl.Service("owner", func() {
			dsl.Method("show", func() {
				dsl.Result(func() {
					dsl.Attribute("ordinary", closed)
					dsl.Attribute("choice", dsl.OneOf(closed, other), func() { dsl.Untagged() })
					dsl.Required("ordinary", "choice")
					dsl.Example(map[string]any{"ordinary": map[string]any{"a": "kept"}, "choice": map[string]any{"b": "chosen"}})
				})
				dsl.HTTP(func() { dsl.GET("/") })
			})
		})
	})
	body := endpoint.Result.Responses[0].ServerBody[0]
	require.NoError(t, body.Value.Error)
	projected := body.Value.Source.Context.ProjectJSON(body.Value.Source.Example, body.Value.Plan)
	require.Equal(t, expr.ProjectionEmitted, projected.Outcome(), projected.Diagnostics())
}

func TestHTTPValueDocumentationAndCodecOwners(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		dsl.Service("contracts", func() {
			dsl.Method("raw", func() {
				dsl.Result(dsl.String, func() { dsl.Example("service") })
				dsl.HTTP(func() {
					dsl.POST("/raw")
					dsl.SkipRequestBodyEncodeDecode()
					dsl.OpenAPIRequestBody(dsl.Bytes, "application/octet-stream", true, func() { dsl.Example([]byte("request")) })
					dsl.Response(dsl.StatusOK, func() {
						dsl.OpenAPIBody(dsl.String, func() { dsl.Example("documentation") })
					})
				})
			})
		})
	})
	endpoint := CreateHTTPServices(root).Get("contracts").Endpoints[0]
	ir := endpoint.valueTransport
	require.True(t, ir.Request.DocumentValue.Documentary)
	require.NoError(t, ir.Request.DocumentValue.Error)
	document := ir.Response.Responses[0].DocumentValue
	require.True(t, document.Documentary)
	require.NoError(t, document.Error)
	require.Same(t, endpoint.Method.ResultValue.Context, document.Source.Context)
	require.NotEqual(t, endpoint.Method.ResultValue.Occurrence.ID(), document.Source.Occurrence.ID())
	require.NotEqual(t, endpoint.Method.ResultValue.Example.SourceID(), document.Source.Example.SourceID())
	raw, present := endpoint.Method.ResultValue.Example.LegacyValue()
	require.True(t, present)
	require.Equal(t, "service", raw)
	for _, codec := range []expr.ValueCodec{expr.ValueCodecRaw, expr.ValueCodecText, expr.ValueCodecForm, expr.ValueCodecMultipart, expr.ValueCodecCustom} {
		target := representation.ValueTarget(endpoint.Method.ResultValue, ir.Response.Responses[0].Body, "", codec, false)
		require.Equal(t, codec, target.Codec)
		require.NotEmpty(t, target.Boundary)
	}
}

func TestHTTPValueNestedAuthoredSourceRemainsInherited(t *testing.T) {
	endpoint := firstEndpointData(t, func() {
		dsl.Service("nested", func() {
			dsl.Method("show", func() {
				dsl.Payload(func() {
					dsl.Attribute("body:wire", dsl.String, func() { dsl.Example("nested source") })
					dsl.Attribute("trace", dsl.String)
				})
				dsl.HTTP(func() {
					dsl.POST("/")
					dsl.Body("body")
					dsl.Header("trace")
				})
			})
		})
	})
	target := endpoint.Payload.Request.ClientBody.Value
	require.NoError(t, target.Error)
	require.Same(t, endpoint.Method.PayloadValue, target.Source)
	require.Equal(t, []string{"body:wire"}, target.Selection)
	require.True(t, target.Source.Example.Synthesized())
	result := target.Source.Context.ProjectJSON(target.Source.Example, target.Plan)
	require.Equal(t, expr.ProjectionEmitted, result.Outcome(), result.Diagnostics())
}

func TestHTTPValueCustomTargetBoundaryFollowsEmission(t *testing.T) {
	for _, hidden := range []bool{false, true} {
		t.Run(fmt.Sprintf("hidden=%t", hidden), func(t *testing.T) {
			field := &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:field:type": {"time.Time", "time"}}}
			if hidden {
				field.Meta["struct:tag:json"] = []string{"-"}
			}
			attribute := &expr.AttributeExpr{Type: &expr.Object{{Name: "when", Attribute: field}}}
			context := expr.NewValueContext()
			occurrence, err := context.NewOccurrence(attribute)
			require.NoError(t, err)
			source := context.SupplyValue(expr.ValueInput{Raw: map[string]any{"when": "not a timestamp"}})
			result := context.Resolve(occurrence, source, expr.ValueRoleExample)
			value := &transportir.ValueTarget{Source: &service.ValueData{Context: context, Occurrence: occurrence, Example: result}, Codec: expr.ValueCodecJSON}
			planned := representation.BuildValuePlan(value, attribute, httpContext(codegen.NewNameScope(), false, true), expr.ValuePlanRuntime)
			require.NoError(t, planned.Error)
			if hidden {
				require.Empty(t, planned.Boundary, "excluded custom fields create no wire obligation")
				require.Equal(t, expr.ProjectionEmitted, context.ProjectJSON(result, planned.Plan).Outcome())
			} else {
				require.NotEmpty(t, planned.Boundary, "String does not authorize time.Time decoding")
				var scalar string
				require.NoError(t, json.Unmarshal([]byte(`"not a timestamp"`), &scalar))
				var actual time.Time
				require.Error(t, json.Unmarshal([]byte(`"not a timestamp"`), &actual))
			}
		})
	}
}

func TestHTTPValueTargetGraphRecursiveDecoderContexts(t *testing.T) {
	named := &expr.UserTypeExpr{TypeName: "Recursive", AttributeExpr: &expr.AttributeExpr{}}
	named.Type = &expr.Object{{Name: "next", Attribute: &expr.AttributeExpr{Type: named, Nullable: true}}}
	choice := &expr.Union{Untagged: true, Values: []*expr.NamedAttributeExpr{{Name: "recursive", Attribute: &expr.AttributeExpr{Type: named}}}}
	root := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "ordinary", Attribute: &expr.AttributeExpr{Type: named}},
		{Name: "choice", Attribute: &expr.AttributeExpr{Type: choice}},
	}}
	copied := representation.TargetGraph(root)
	ordinary := expr.AsObject(copied.Type).Attribute("ordinary").Type.(expr.UserType)
	branch := expr.AsUnion(expr.AsObject(copied.Type).Attribute("choice").Type).Values[0].Attribute.Type.(expr.UserType)
	require.NotSame(t, ordinary, branch, "branch decoder context is distinct")
	require.Same(t, ordinary, expr.AsObject(branch.Attribute().Type).Attribute("next").Type, "consuming a member returns to ordinary decoding")
	require.Same(t, ordinary, expr.AsObject(ordinary.Attribute().Type).Attribute("next").Type, "finite graph reservation supports recursive declarations")
	require.Same(t, named, expr.AsObject(root.Type).Attribute("ordinary").Type, "source declaration is untouched")
}

// TestHTTPInheritedErrorPreparedSchemas exercises the public generation entry
// with inherited mappings, including same-name default error declarations.
func TestHTTPInheritedErrorPreparedSchemas(t *testing.T) {
	for _, test := range []struct {
		name   string
		design func()
	}{
		{"body", testdata.APIErrorResponseDSL},
		{"body content type", testdata.APIErrorResponseWithContentTypeDSL},
		{"header", testdata.APINoBodyErrorResponseDSL},
		{"header content type", testdata.APINoBodyErrorResponseWithContentTypeDSL},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := RunHTTPDSL(t, test.design)
			files, err := OpenAPIFiles(root)
			require.NoError(t, err)
			require.NotEmpty(t, files)
		})
	}
}
