package expr_test

import (
	"errors"
	"strings"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/expr/testdata"
)

func TestHTTPRouteValidation(t *testing.T) {
	cases := []struct {
		Name  string
		DSL   func()
		Error string
	}{
		{"valid", testdata.ValidRouteDSL, ""},
		{"duplicate-wc-route", testdata.DuplicateWCRouteDSL, `route POST "/{id}" of service "DuplicateWCRoute" HTTP endpoint "Method": Wildcard "id" appears multiple times in full path "/{id}/{id}"`},
		{"disallow-response-body", testdata.DisallowResponseBodyHeadDSL, `route HEAD "/" of service "DisallowResponseBody" HTTP endpoint "Method": HTTP status 200: Response body defined for HEAD method which does not allow response body.
route HEAD "/" of service "DisallowResponseBody" HTTP endpoint "Method": HTTP status 404: Response body defined for HEAD method which does not allow response body.`,
		},
		{"invalid", testdata.InvalidRouteDSL, "routes at the service level are only allowed for JSON-RPC services. Use method-level routes instead. in service \"InvalidRoute\""},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			if c.Error == "" {
				expr.RunDSL(t, c.DSL)
			} else {
				err := expr.RunInvalidDSL(t, c.DSL)
				got := stripValidationLocations(err.Error())
				if !strings.HasSuffix(got, c.Error) {
					t.Errorf("got error %q\nexpected %q", got, c.Error)
				}
			}
		})
	}
}

func TestHTTPRouteParamConsistency(t *testing.T) {
	err := expr.RunInvalidDSL(t, inconsistentRouteParamsDSL)
	got := stripValidationLocations(err.Error())
	if !strings.Contains(got, `service "RouteMismatch" HTTP endpoint "Show": Param "id" does not appear in all routes`) {
		t.Fatalf("missing id route mismatch error: %q", got)
	}
	if !strings.Contains(got, `service "RouteMismatch" HTTP endpoint "Show": Param "slug" does not appear in all routes`) {
		t.Fatalf("missing slug route mismatch error: %q", got)
	}
}

func TestHTTPEndpointPrepare(t *testing.T) {
	cases := map[string]struct {
		DSL     func()
		Headers []string
		Params  []string
		Cookies []string
		Error   string
	}{
		"valid": {
			DSL:    testdata.ValidRouteDSL,
			Params: []string{"base_id", "id"},
		},
		"with parent": {
			DSL:     testdata.EndpointWithParentDSL,
			Headers: []string{"pheader", "header"},
			Params:  []string{"pparam", "param"},
			Cookies: []string{"pcookie", "cookie"},
		},
		"with parent revert": {
			DSL:     testdata.EndpointWithParentRevertDSL,
			Headers: []string{"pheader", "header"},
			Params:  []string{"pparam", "param"},
			Cookies: []string{"pcookie", "cookie"},
		},
		"error": {
			DSL:   testdata.EndpointRecursiveParentDSL,
			Error: "service \"Parent\": Parent service Child is also child\nservice \"Child\": Parent service Parent is also child",
		},
	}
	for n, c := range cases {
		t.Run(n, func(t *testing.T) {
			if c.Error == "" {
				root := expr.RunDSL(t, c.DSL)
				e := root.API.HTTP.Services[len(root.API.HTTP.Services)-1].HTTPEndpoints[0]

				ht := expr.AsObject(e.Headers.Type)
				if len(*ht) != len(c.Headers) {
					t.Errorf("got %d headers, expected %d", len(*ht), len(c.Headers))
				} else {
					for _, n := range c.Headers {
						if ht.Attribute(n) == nil {
							t.Errorf("header %q is missing", n)
						}
					}
				}

				ct := expr.AsObject(e.Cookies.Type)
				if len(*ct) != len(c.Cookies) {
					t.Errorf("got %d cookies, expected %d", len(*ct), len(c.Cookies))
				} else {
					for _, n := range c.Cookies {
						if ct.Attribute(n) == nil {
							t.Errorf("cookie %q is missing", n)
						}
					}
				}

				pt := expr.AsObject(e.Params.Type)
				if len(*pt) != len(c.Params) {
					t.Errorf("got %d params, expected %d", len(*pt), len(c.Params))
				} else {
					for _, n := range c.Params {
						if pt.Attribute(n) == nil {
							t.Errorf("param %q is missing", n)
						}
					}
				}
			} else {
				err := expr.RunInvalidDSL(t, c.DSL)
				got := stripValidationLocations(err.Error())
				if got != c.Error {
					t.Errorf("got error %q, expected %q", got, c.Error)
				}
			}
		})
	}
}

func TestHTTPEndpointValidation(t *testing.T) {
	cases := map[string]struct {
		DSL   func()
		Error string
	}{
		"endpoint-body-as-payload-prop": {
			DSL: testdata.EndpointBodyAsPayloadProp,
		},
		"endpoint-body-as-missed-payload-prop": {
			DSL:   testdata.EndpointBodyAsMissedPayloadProp,
			Error: `Request type does not have an attribute named "name" in service "Service" HTTP endpoint "Method"`,
		},
		"endpoint-body-extend-payload": {
			DSL: testdata.EndpointBodyExtendPayload,
		},
		"endpoint-body-as-user-type": {
			DSL: testdata.EndpointBodyAsUserType,
		},
		"endpoint-optional-request-body": {
			DSL: testdata.EndpointOptionalRequestBody,
		},
		"endpoint-optional-request-body-missing-body": {
			DSL:   testdata.EndpointOptionalRequestBodyMissingBody,
			Error: `service "Service" HTTP endpoint "Method": HTTP endpoint uses OptionalRequestBody but does not define a request body.`,
		},
		"endpoint-optional-request-body-with-form": {
			DSL: testdata.EndpointOptionalRequestBodyWithForm,
			Error: `service "Service" HTTP endpoint "Method": HTTP endpoint cannot use OptionalRequestBody with FormRequest.
service "Service" HTTP endpoint "Method": HTTP endpoint defines FormRequest and body. At most one of these must be defined.`,
		},
		"endpoint-optional-request-body-with-multipart": {
			DSL: testdata.EndpointOptionalRequestBodyWithMultipart,
			Error: `service "Service" HTTP endpoint "Method": HTTP endpoint cannot use OptionalRequestBody with MultipartRequest.
service "Service" HTTP endpoint "Method": HTTP endpoint defines MultipartRequest and body. At most one of these must be defined.`,
		},
		"endpoint-optional-request-body-required-attribute": {
			DSL:   testdata.EndpointOptionalRequestBodyRequiredAttribute,
			Error: `service "Service" HTTP endpoint "Method": OptionalRequestBody requires the request body to have no required attributes.`,
		},
		"endpoint-optional-request-body-required-origin-attribute": {
			DSL:   testdata.EndpointOptionalRequestBodyRequiredOriginAttribute,
			Error: `service "Service" HTTP endpoint "Method": OptionalRequestBody requires the payload attribute mapped to the request body to be optional.`,
		},
		"endpoint-missing-token": {
			DSL:   testdata.EndpointMissingToken,
			Error: `service "Service" method "Method": payload of method "Method" of service "Service" does not define a JWT attribute, use Token to define one`,
		},
		"endpoint-missing-token-payload": {
			DSL:   testdata.EndpointMissingTokenPayload,
			Error: `service "Service" method "Method": payload of method "Method" of service "Service" does not define a JWT attribute, use Token to define one`,
		},
		"endpoint-missing-token-extend": {
			DSL: testdata.EndpointExtendToken,
		},
		"endpoint-has-parent": {
			DSL: testdata.EndpointHasParent,
		},
		"endpoint-has-parent-and-other": {
			DSL: testdata.EndpointHasParentAndOther,
		},
		"endpoint-has-skip-request-encode-and-payload-streaming": {
			DSL:   testdata.EndpointHasSkipRequestEncodeAndPayloadStreaming,
			Error: `service "Service" HTTP endpoint "Method": Endpoint cannot use SkipRequestBodyEncodeDecode when method defines a StreamingPayload.`,
		},
		"endpoint-skip-request-encode-with-form-body": {
			DSL: testdata.EndpointHasSkipRequestEncodeAndFormWithBody,
			Error: `service "Service" HTTP endpoint "Method": HTTP endpoint cannot use FormRequest with SkipRequestBodyEncodeDecode.
service "Service" HTTP endpoint "Method": HTTP endpoint request body must be empty when using SkipRequestBodyEncodeDecode but not all method payload attributes are mapped to headers and params. Make sure to define Headers and Params as needed.`,
		},
		"endpoint-skip-request-encode-with-multipart-body": {
			DSL: testdata.EndpointHasSkipRequestEncodeAndMultipartWithBody,
			Error: `service "Service" HTTP endpoint "Method": Cannot define a request body when using SkipRequestBodyEncodeDecode.
service "Service" HTTP endpoint "Method": HTTP endpoint defines MultipartRequest and body. At most one of these must be defined.
service "Service" HTTP endpoint "Method": HTTP endpoint request body must be empty when using SkipRequestBodyEncodeDecode but not all method payload attributes are mapped to headers and params. Make sure to define Headers and Params as needed.`,
		},
		"endpoint-has-skip-request-encode-and-result-streaming": {
			DSL:   testdata.EndpointHasSkipRequestEncodeAndResultStreaming,
			Error: `service "Service" HTTP endpoint "Method": Endpoint cannot use SkipRequestBodyEncodeDecode when method defines a StreamingResult. Use SkipResponseBodyEncodeDecode instead.`,
		},
		"endpoint-has-skip-response-encode-and-payload-streaming": {
			DSL:   testdata.EndpointHasSkipResponseEncodeAndPayloadStreaming,
			Error: `service "Service" HTTP endpoint "Method": Endpoint cannot use SkipResponseBodyEncodeDecode when method defines a StreamingPayload. Use SkipRequestBodyEncodeDecode instead.`,
		},
		"endpoint-has-skip-response-encode-and-result-streaming": {
			DSL: testdata.EndpointHasSkipResponseEncodeAndResultStreaming,
			Error: `service "Service" HTTP endpoint "Method": Endpoint cannot use SkipResponseBodyEncodeDecode when method defines a StreamingResult.
service "Service" HTTP endpoint "Method": HTTP endpoint response body must be empty when using SkipResponseBodyEncodeDecode. Make sure to define headers and cookies as needed.`,
		},
		"endpoint-has-skip-encode-and-grpc": {
			DSL:   testdata.EndpointHasSkipEncodeAndGRPC,
			Error: `service "Service" HTTP endpoint "Method": Endpoint cannot use SkipRequestBodyEncodeDecode and define a gRPC transport.`,
		},
		"endpoint-payload-missing-required": {
			DSL:   testdata.EndpointPayloadMissingRequired,
			Error: `service "Service" HTTP endpoint "Method": The following HTTP request body attribute is required but the corresponding method payload attribute is not: nonreq. Use 'Required' to make the attribute required in the method payload as well.`,
		},
		"streaming-endpoint-has-request-body": {
			DSL: testdata.StreamingEndpointRequestBody,
			Error: `service "Service" HTTP endpoint "MethodA": HTTP endpoint request body must be empty when the endpoint uses streaming. Payload attributes must be mapped to headers and/or params.
service "Service" HTTP endpoint "MethodB": HTTP endpoint request body must be empty when the endpoint uses streaming. Payload attributes must be mapped to headers and/or params.
service "Service" HTTP endpoint "MethodC": HTTP endpoint request body must be empty when the endpoint uses streaming. Payload attributes must be mapped to headers and/or params.`,
		},
		"endpoint-union-query-param": {
			DSL:   testdata.EndpointUnionQueryParam,
			Error: `service "Service" HTTP endpoint "Method": path parameter filter cannot be an object, path parameter types must be primitive, array or map (query string only)`,
		},
		"endpoint-union-header": {
			DSL:   testdata.EndpointUnionHeader,
			Error: `service "Service" HTTP endpoint "Method": header "filter" must be primitive or array`,
		},
		"endpoint-union-cookie": {
			DSL:   testdata.EndpointUnionCookie,
			Error: `service "Service" HTTP endpoint "Method": cookie "filter" must be primitive`,
		},
		"endpoint-multipart-constructor-union": {
			DSL:   testdata.EndpointMultipartConstructorUnion,
			Error: `service "Service" HTTP endpoint "Method": MultipartRequest requires an object payload, constructor unions are not supported`,
		},
		"endpoint-array-payload-body-conflict": {
			DSL:   testdata.EndpointArrayPayloadBodyConflict,
			Error: `service "Service" HTTP endpoint "Method": Payload type is array but HTTP endpoint body is not.`,
		},
		"endpoint-map-payload-body-conflict": {
			DSL:   testdata.EndpointMapPayloadBodyConflict,
			Error: `service "Service" HTTP endpoint "Method": Payload type is map but HTTP endpoint body is not.`,
		},
		"endpoint-object-payload-body-conflict": {
			DSL:   testdata.EndpointObjectPayloadBodyConflict,
			Error: `service "Service" HTTP endpoint "Method": Body "value" is not found in Payload.`,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if c.Error == "" {
				expr.RunDSL(t, c.DSL)
			} else {
				var errs []error
				err := expr.RunInvalidDSL(t, c.DSL)
				if err != nil {
					var merr eval.MultiError
					if errors.As(err, &merr) {
						for _, e := range merr {
							errs = append(errs, e.GoError)
						}
					} else {
						errs = append(errs, err)
					}
				}
				if len(errs) > 1 || len(errs) == 0 {
					t.Errorf("got %d errors, expected 1", len(errs))
				} else if got := stripValidationLocations(errs[0].Error()); got != c.Error {
					t.Errorf("got `%s`, expected `%s`", got, c.Error)
				}
			}
		})
	}
}

func TestHTTPEndpointStreamingValidationCoverage(t *testing.T) {
	cases := map[string]struct {
		DSL      func()
		Contains string
	}{
		"mixed results require sse": {
			DSL:      mixedResultsWithoutSSEDsl,
			Contains: `Methods with both Result and StreamingResult defined with different types must use ServerSentEvents()`,
		},
		"sse disallows client stream": {
			DSL:      sseWithClientStreamDsl,
			Contains: `Server-Sent Events cannot be used with client-to-server streaming endpoints`,
		},
		"sse disallows bidirectional stream": {
			DSL:      sseWithBidirectionalStreamDsl,
			Contains: `Server-Sent Events cannot be used with bidirectional streaming endpoints`,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, tc.DSL)
			got := stripValidationLocations(err.Error())
			if !strings.Contains(got, tc.Contains) {
				t.Fatalf("got %q, expected substring %q", got, tc.Contains)
			}
		})
	}
}

func TestHTTPEndpointParentRequired(t *testing.T) {
	root := expr.RunDSL(t, testdata.EndpointHasParent)
	svc := root.Service("Child")
	if svc == nil {
		t.Fatal(`unexpected error, service "Child" not found`)
	}
	m := svc.Method("Method")
	if m == nil || m.Payload == nil {
		t.Fatal(`unexpected error, method "Method" or its payload not found`)
	}
	if !m.Payload.IsRequired("ancestor_id") {
		t.Errorf(`expected "ancestor_id" is required, but not so`)
	}
	if !m.Payload.IsRequired("parent_id") {
		t.Errorf(`expected "parent_id" is required, but not so`)
	}
}

func TestHTTPEndpointFinalization(t *testing.T) {
	cases := map[string]struct {
		DSL          func()
		ExpectedBody expr.DataType
	}{
		"body-as-extend-type": {
			DSL:          testdata.FinalizeEndpointBodyAsExtendedTypeDSL,
			ExpectedBody: testdata.FinalizeEndpointBodyAsExtendedType,
		},
		"body-as-prop-with-extend-type": {
			DSL:          testdata.FinalizeEndpointBodyAsPropWithExtendedTypeDSL,
			ExpectedBody: testdata.FinalizeEndpointBodyAsPropWithExtendedType,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			root := expr.RunDSL(t, tc.DSL)
			e := root.API.HTTP.Services[0].HTTPEndpoints[0]

			if tc.ExpectedBody != nil {
				if e.Body == nil {
					t.Errorf("got endpoint without body, expected endpoint with body")
					return
				}
				bodyObj := *expr.AsObject(e.Body.Type)
				expectedBodyObj := *expr.AsObject(tc.ExpectedBody)
				if len(bodyObj) != len(expectedBodyObj) {
					t.Errorf("got %d, expected %d attribute(s) in endpoint body", len(bodyObj), len(expectedBodyObj))
				} else {
					for i := range expectedBodyObj {
						if bodyObj[i].Name != expectedBodyObj[i].Name {
							t.Errorf("got %q, expected %q attribute in endpoint body", bodyObj[i].Name, expectedBodyObj[i].Name)
						}
					}
				}
			}
		})
	}
}

func TestHTTPEndpointPrepareAdditionalCoverage(t *testing.T) {
	t.Run("service sse inheritance", func(t *testing.T) {
		root := expr.RunDSL(t, serviceLevelSSEInheritanceDSL)
		e := root.API.HTTP.Services[0].HTTPEndpoints[0]
		if e.SSE == nil {
			t.Fatalf("expected SSE inheritance from service HTTP block")
		}
		if e.SSE.DataField != "event" {
			t.Fatalf("got SSE data field %q, expected %q", e.SSE.DataField, "event")
		}
	})

	t.Run("api sse inheritance", func(t *testing.T) {
		root := expr.RunDSL(t, apiLevelSSEInheritanceDSL)
		e := root.API.HTTP.Services[0].HTTPEndpoints[0]
		if e.SSE == nil {
			t.Fatalf("expected SSE inheritance from API HTTP block")
		}
		if e.SSE.DataField != "event" {
			t.Fatalf("got SSE data field %q, expected %q", e.SSE.DataField, "event")
		}
	})

	t.Run("service http errors inherited", func(t *testing.T) {
		root := expr.RunDSL(t, serviceLevelHTTPErrorsDSL)
		e := root.API.HTTP.Services[0].HTTPEndpoints[0]
		if len(e.HTTPErrors) != 1 {
			t.Fatalf("got %d HTTP errors, expected 1", len(e.HTTPErrors))
		}
		if e.HTTPErrors[0].Name != "bad_request" || e.HTTPErrors[0].Response.StatusCode != StatusBadRequest {
			t.Fatalf("unexpected inherited service error %#v", e.HTTPErrors[0])
		}
	})

	t.Run("api http errors inherited", func(t *testing.T) {
		root := expr.RunDSL(t, apiLevelHTTPErrorsDSL)
		e := root.API.HTTP.Services[0].HTTPEndpoints[0]
		if len(e.HTTPErrors) != 1 {
			t.Fatalf("got %d HTTP errors, expected 1", len(e.HTTPErrors))
		}
		if e.HTTPErrors[0].Name != "bad_request" || e.HTTPErrors[0].Response.StatusCode != StatusBadRequest {
			t.Fatalf("unexpected inherited API error %#v", e.HTTPErrors[0])
		}
	})

	t.Run("websocket route coerced to get", func(t *testing.T) {
		root := expr.RunDSL(t, websocketRouteMethodCoercionDSL)
		e := root.API.HTTP.Services[0].HTTPEndpoints[0]
		if len(e.Routes) != 1 {
			t.Fatalf("got %d routes, expected 1", len(e.Routes))
		}
		if e.Routes[0].Method != "GET" {
			t.Fatalf("got route method %q, expected GET", e.Routes[0].Method)
		}
	})

	t.Run("default no content response", func(t *testing.T) {
		root := expr.RunDSL(t, defaultNoContentResponseDSL)
		e := root.API.HTTP.Services[0].HTTPEndpoints[0]
		if len(e.Responses) != 1 {
			t.Fatalf("got %d responses, expected 1", len(e.Responses))
		}
		if e.Responses[0].StatusCode != StatusNoContent {
			t.Fatalf("got response status %d, expected %d", e.Responses[0].StatusCode, StatusNoContent)
		}
	})

	t.Run("default redirect response", func(t *testing.T) {
		root := expr.RunDSL(t, defaultRedirectResponseDSL)
		e := root.API.HTTP.Services[0].HTTPEndpoints[0]
		if len(e.Responses) != 1 {
			t.Fatalf("got %d responses, expected 1", len(e.Responses))
		}
		if e.Responses[0].StatusCode != StatusMovedPermanently {
			t.Fatalf("got response status %d, expected %d", e.Responses[0].StatusCode, StatusMovedPermanently)
		}
	})

	t.Run("jsonrpc websocket payload migration", func(t *testing.T) {
		root := expr.RunDSL(t, jsonrpcWebSocketPayloadMigrationDSL)
		e := root.API.JSONRPC.Services[0].HTTPEndpoints[0]
		if e.MethodExpr.Stream != expr.BidirectionalStreamKind {
			t.Fatalf("got stream kind %v, expected bidirectional", e.MethodExpr.Stream)
		}
		if e.MethodExpr.Payload == nil || e.MethodExpr.Payload.Type == expr.Empty {
			t.Fatalf("expected finalized payload to remain addressable for JSON-RPC WebSocket")
		}
		if e.MethodExpr.StreamingPayload == nil || e.MethodExpr.StreamingPayload.Type == expr.Empty {
			t.Fatalf("expected streaming payload to be populated during JSON-RPC WebSocket finalize")
		}
		if e.Body == nil || e.Body.Type == expr.Empty {
			t.Fatalf("expected request body to remain addressable for migrated JSON-RPC payload")
		}
		if e.Body != e.StreamingBody {
			t.Fatalf("expected migrated JSON-RPC body to reuse streaming body")
		}
	})

	t.Run("implicit session cookie mapping", func(t *testing.T) {
		root := expr.RunDSL(t, sessionCookieMappingDSL)
		e := root.API.HTTP.Services[0].HTTPEndpoints[0]
		if e.MethodExpr.Payload.Find("browser_session_key") != nil {
			t.Fatalf("expected transport-only session cookie auth to avoid payload injection")
		}
		if e.Cookies.Find("browser_session_key") == nil {
			t.Fatalf("expected implicit session cookie mapping for browser_session_key")
		}
		if e.Cookies.ElemName("browser_session_key") != "__Host-browser_session" {
			t.Fatalf("got cookie name %q, expected %q", e.Cookies.ElemName("browser_session_key"), "__Host-browser_session")
		}
	})

	t.Run("jsonrpc id projection captured on endpoint", func(t *testing.T) {
		root := expr.RunDSL(t, jsonrpcEndpointIDProjectionDSL)
		e := root.API.JSONRPC.Services[0].HTTPEndpoints[0]
		if e.PayloadIDAttribute != "id" {
			t.Fatalf("got payload ID attribute %q, expected %q", e.PayloadIDAttribute, "id")
		}
		if e.ResultIDAttribute != "id" {
			t.Fatalf("got result ID attribute %q, expected %q", e.ResultIDAttribute, "id")
		}
	})
}

func TestHTTPEndpointValidationAdditionalCoverage(t *testing.T) {
	cases := map[string]struct {
		DSL      func()
		Contains string
	}{
		"all tagged responses rejected": {
			DSL:      allTaggedResponsesDSL,
			Contains: "All responses define a Tag, at least one response must define no Tag.",
		},
		"tagged responses require object result": {
			DSL:      taggedPrimitiveResultDSL,
			Contains: "Some responses define a Tag but the method Result type is not an object.",
		},
		"duplicate response status code": {
			DSL:      testdata.EndpointDuplicateResponseStatusCode,
			Contains: "Multiple response definitions with status code 200",
		},
		"sse on non streaming endpoint rejected": {
			DSL:      nonStreamingSSEDsl,
			Contains: "Server-Sent Events can only be used with endpoints that have a streaming result or mixed results",
		},
		"jsonrpc result id requires request id": {
			DSL:      jsonrpcResultIDWithoutRequestIDDSL,
			Contains: `JSON-RPC method "show" result defines an ID field but the request (payload) does not`,
		},
		"redirect rejects mismatched response status": {
			DSL:      redirectWithMismatchedResponseDSL,
			Contains: "Endpoint cannot use Response when using Redirect.",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, tc.DSL)
			got := stripValidationLocations(err.Error())
			if !strings.Contains(got, tc.Contains) {
				t.Fatalf("got %q, expected substring %q", got, tc.Contains)
			}
		})
	}
}

func TestHTTPStreamingSessionSecurityRequestBodyInference(t *testing.T) {
	cases := map[string]struct {
		DSL                func()
		ExpectedRoute      string
		ExpectedParamField string
		ExpectStreamingIn  bool
	}{
		"server stream path param": {
			DSL:                websocketSessionCookiePathStreamDSL,
			ExpectedRoute:      "/ws/projects/{project_id}",
			ExpectedParamField: "project_id",
		},
		"bidirectional path param": {
			DSL:                websocketSessionCookiePathBidirectionalDSL,
			ExpectedRoute:      "/ws/projects/{project_id}",
			ExpectedParamField: "project_id",
			ExpectStreamingIn:  true,
		},
		"server stream query param": {
			DSL:                websocketSessionCookieQueryStreamDSL,
			ExpectedRoute:      "/ws/projects",
			ExpectedParamField: "project_id",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			root := expr.RunDSL(t, tc.DSL)
			e := root.API.HTTP.Services[0].HTTPEndpoints[0]

			if e.Body == nil || e.Body.Type != expr.Empty {
				t.Fatalf("expected websocket handshake request body to be empty, got %#v", e.Body)
			}
			if e.Cookies.Find("browser_session_cookie") == nil {
				t.Fatalf("expected inferred browser_session_cookie cookie mapping")
			}
			if e.Params.Find(tc.ExpectedParamField) == nil {
				t.Fatalf("expected %q handshake param mapping", tc.ExpectedParamField)
			}
			if len(e.Routes) != 1 || e.Routes[0].Path != tc.ExpectedRoute {
				t.Fatalf("got route %#v, expected %q", e.Routes, tc.ExpectedRoute)
			}
			if tc.ExpectStreamingIn && (e.StreamingBody == nil || e.StreamingBody.Type == expr.Empty) {
				t.Fatalf("expected bidirectional websocket endpoint to retain streaming body")
			}
		})
	}
}

func TestHTTPAuthorizationMapping(t *testing.T) {
	cases := []struct {
		Name           string
		DSL            func()
		ExpectedHeader string
	}{{
		Name:           "explicit",
		DSL:            testdata.ExplicitAuthHeaderDSL,
		ExpectedHeader: "token",
	}, {
		Name:           "implicit",
		DSL:            testdata.ImplicitAuthHeaderDSL,
		ExpectedHeader: "Authorization",
	},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			root := expr.RunDSL(t, tc.DSL)
			e := root.API.HTTP.Services[0].HTTPEndpoints[0]
			if e.Headers == nil {
				t.Errorf("got endpoint without header, expected endpoint with HTTP header")
				return
			}
			if len(*expr.AsObject(e.Headers.Type)) != 1 {
				t.Errorf("got %d, expected 1 attribute in endpoint headers", len(*expr.AsObject(e.Headers.Type)))
				return
			}
			n := e.Headers.ElemName("token")
			if n != tc.ExpectedHeader {
				t.Errorf("got %q, expected %q attribute in endpoint headers", n, tc.ExpectedHeader)
			}
		})
	}
}

func websocketSessionCookieAuth() any {
	browserSession := APIKeySecurity("browser_session_cookie", func() {
		Description("Browser session cookie")
	})
	return SessionAuth("app_session", func() {
		CookieTransport(browserSession, "", func() {
			CookieName("__Host-ak_session")
		})
	})
}

var inconsistentRouteParamsDSL = func() {
	Service("RouteMismatch", func() {
		Method("Show", func() {
			Payload(func() {
				Attribute("id", String)
				Attribute("slug", String)
			})
			Result(String)
			HTTP(func() {
				GET("/{id}")
				GET("/{slug}")
			})
		})
	})
}

var websocketSessionCookiePathStreamDSL = websocketSessionCookieStreamingDSL(
	"WebSocketSessionCookiePathStream",
	"/ws/projects/{project_id}",
	false,
)

var websocketSessionCookiePathBidirectionalDSL = websocketSessionCookieStreamingDSL(
	"WebSocketSessionCookiePathBidirectional",
	"/ws/projects/{project_id}",
	true,
)

var websocketSessionCookieQueryStreamDSL = websocketSessionCookieStreamingDSL(
	"WebSocketSessionCookieQueryStream",
	"/ws/projects",
	false,
)

func websocketSessionCookieStreamingDSL(serviceName string, route string, bidirectional bool) func() {
	return func() {
		appSession := websocketSessionCookieAuth()
		Service(serviceName, func() {
			Method("connect", func() {
				SessionSecurity(appSession)
				Payload(func() {
					Attribute("project_id", String)
					Required("project_id")
				})
				if bidirectional {
					StreamingPayload(String)
				}
				StreamingResult(func() {
					Attribute("event", String)
					Required("event")
				})
				HTTP(func() {
					GET(route)
					Param("project_id")
					Response(StatusOK)
				})
			})
		})
	}
}

var mixedResultsWithoutSSEDsl = func() {
	Service("MixedResults", func() {
		Method("Watch", func() {
			Result(func() {
				Attribute("done", Boolean)
			})
			StreamingResult(func() {
				Attribute("event", String)
			})
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var sseWithClientStreamDsl = func() {
	Service("ClientStreamSSE", func() {
		Method("Watch", func() {
			StreamingPayload(func() {
				Attribute("value", String)
			})
			Result(func() {
				Attribute("done", Boolean)
			})
			HTTP(func() {
				GET("/")
				ServerSentEvents()
			})
		})
	})
}

var sseWithBidirectionalStreamDsl = func() {
	Service("BidirectionalStreamSSE", func() {
		Method("Watch", func() {
			StreamingPayload(func() {
				Attribute("value", String)
			})
			StreamingResult(func() {
				Attribute("event", String)
			})
			HTTP(func() {
				GET("/")
				ServerSentEvents()
			})
		})
	})
}

var serviceLevelSSEInheritanceDSL = func() {
	Service("ServiceLevelSSE", func() {
		HTTP(func() {
			ServerSentEvents("event")
		})
		Method("watch", func() {
			StreamingResult(func() {
				Attribute("event", String)
			})
			HTTP(func() {
				GET("/watch")
			})
		})
	})
}

var apiLevelSSEInheritanceDSL = func() {
	API("APISSE", func() {
		HTTP(func() {
			ServerSentEvents("event")
		})
	})
	Service("APILevelSSE", func() {
		Method("watch", func() {
			StreamingResult(func() {
				Attribute("event", String)
			})
			HTTP(func() {
				GET("/watch")
			})
		})
	})
}

var serviceLevelHTTPErrorsDSL = func() {
	Service("ServiceLevelErrors", func() {
		Error("bad_request")
		HTTP(func() {
			Response("bad_request", StatusBadRequest)
		})
		Method("show", func() {
			Error("bad_request")
			Result(String)
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var apiLevelHTTPErrorsDSL = func() {
	API("APILevelErrors", func() {
		Error("bad_request")
		HTTP(func() {
			Response("bad_request", StatusBadRequest)
		})
	})
	Service("APIInheritedErrors", func() {
		Method("show", func() {
			Error("bad_request")
			Result(String)
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var websocketRouteMethodCoercionDSL = func() {
	Service("WebSocketRouteMethodCoercion", func() {
		Method("stream", func() {
			StreamingResult(String)
			HTTP(func() {
				POST("/stream")
			})
		})
	})
}

var defaultNoContentResponseDSL = func() {
	Service("DefaultNoContent", func() {
		Method("create", func() {
			HTTP(func() {
				POST("/")
			})
		})
	})
}

var defaultRedirectResponseDSL = func() {
	Service("DefaultRedirect", func() {
		Method("show", func() {
			HTTP(func() {
				GET("/")
				Redirect("/dest", StatusMovedPermanently)
			})
		})
	})
}

var jsonrpcWebSocketPayloadMigrationDSL = func() {
	Service("JSONRPCPayloadMigration", func() {
		Method("stream", func() {
			Payload(func() {
				ID("id", String)
				Attribute("message", String)
				Required("id")
			})
			StreamingResult(func() {
				Attribute("event", String)
			})
			JSONRPC(func() {})
		})
		JSONRPC(func() {
			GET("/rpc")
		})
	})
}

var sessionCookieMappingDSL = func() {
	var browserSession = APIKeySecurity("browser_session_key")
	var appSession = SessionAuth("app_session", func() {
		CookieTransport(browserSession, "", func() {
			CookieName("__Host-browser_session")
		})
	})
	Service("SessionCookieMapping", func() {
		Method("show", func() {
			SessionSecurity(appSession)
			Payload(func() {
				Attribute("message", String)
			})
			Result(String)
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var jsonrpcEndpointIDProjectionDSL = func() {
	Service("JSONRPCEndpointIDProjection", func() {
		Method("show", func() {
			Payload(func() {
				ID("id", String)
				Attribute("query", String)
				Required("id")
			})
			Result(func() {
				ID("id", String)
				Attribute("value", String)
				Required("id")
			})
			JSONRPC(func() {})
		})
		JSONRPC(func() {
			Path("/rpc")
		})
	})
}

var allTaggedResponsesDSL = func() {
	Service("AllTaggedResponses", func() {
		Method("show", func() {
			Result(func() {
				Attribute("kind", String)
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Tag("kind", "ok")
				})
				Response(StatusAccepted, func() {
					Tag("kind", "accepted")
				})
			})
		})
	})
}

var taggedPrimitiveResultDSL = func() {
	Service("TaggedPrimitiveResult", func() {
		Method("show", func() {
			Result(String)
			HTTP(func() {
				GET("/")
				Response(StatusOK)
				Response(StatusAccepted, func() {
					Tag("kind", "accepted")
				})
			})
		})
	})
}

var nonStreamingSSEDsl = func() {
	Service("NonStreamingSSE", func() {
		Method("show", func() {
			Result(func() {
				Attribute("event", String)
			})
			HTTP(func() {
				GET("/")
				ServerSentEvents("event")
			})
		})
	})
}

var jsonrpcResultIDWithoutRequestIDDSL = func() {
	Service("JSONRPCIDValidation", func() {
		Method("show", func() {
			Payload(func() {
				Attribute("query", String)
			})
			Result(func() {
				ID("id", String)
				Attribute("value", String)
				Required("id")
			})
			JSONRPC(func() {})
		})
		JSONRPC(func() {
			Path("/rpc")
		})
	})
}

var redirectWithMismatchedResponseDSL = func() {
	Service("RedirectMismatch", func() {
		Method("show", func() {
			HTTP(func() {
				GET("/")
				Redirect("/dest", StatusMovedPermanently)
				Response(StatusOK)
			})
		})
	})
}
