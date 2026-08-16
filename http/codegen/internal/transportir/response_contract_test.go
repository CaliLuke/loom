package transportir_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	testcodegen "github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

type (
	responseContractSnapshot struct {
		Cases       []responseContractCaseSnapshot           `json:"cases,omitempty"`
		Limitations []transportir.ResponseContractLimitation `json:"limitations,omitempty"`
	}

	responseContractCaseSnapshot struct {
		ID           string                                        `json:"id"`
		Kind         transportir.ResponseContractCaseKind          `json:"kind"`
		StatusCode   int                                           `json:"status_code"`
		ErrorName    string                                        `json:"error_name,omitempty"`
		TagName      string                                        `json:"tag_name,omitempty"`
		TagValue     string                                        `json:"tag_value,omitempty"`
		ContentTypes []string                                      `json:"content_types"`
		Headers      []transportir.ResponseContractHeader          `json:"headers,omitempty"`
		Cookies      []transportir.ResponseContractCookie          `json:"cookies,omitempty"`
		Multipart    *transportir.ResponseContractMultipartRequest `json:"multipart,omitempty"`
	}
)

func TestAnalyzeResponseContractCases(t *testing.T) {
	root := testcodegen.RunDSL(t, responseContractDSL)
	endpoint := transportir.BuildEndpoint(root.API.HTTP.Services[0].HTTPEndpoints[0])

	analysis := transportir.AnalyzeResponseContractCases(endpoint)
	require.True(t, analysis.Supported())
	require.Empty(t, analysis.Limitations)
	require.Len(t, analysis.Cases, 4)

	accepted := analysis.Cases[0]
	require.Equal(t, "widgets.show.success.202.outcome=accepted", accepted.ID)
	require.Equal(t, transportir.ResponseContractSuccess, accepted.Kind)
	require.Equal(t, expr.StatusAccepted, accepted.StatusCode)
	require.Equal(t, "outcome", accepted.TagName)
	require.Equal(t, "accepted", accepted.TagValue)
	require.Equal(t, []string{"application/json"}, accepted.ContentTypes)
	require.True(t, accepted.HasBody)
	require.Len(t, accepted.Headers, 1)
	require.Equal(t, "X-Version", accepted.Headers[0].HTTPName)
	require.Len(t, accepted.Cookies, 1)
	require.Equal(t, "widget_session", accepted.Cookies[0].HTTPName)
	require.Equal(t, "/", accepted.Cookies[0].Path)
	require.True(t, accepted.Cookies[0].Secure)
	require.True(t, accepted.Cookies[0].HTTPOnly)
	require.Equal(t, expr.CookieSameSiteLax, accepted.Cookies[0].SameSite)
	fallback := analysis.Cases[1]
	require.Equal(t, "widgets.show.success.200", fallback.ID)
	require.Empty(t, fallback.TagName)
	require.Empty(t, fallback.TagValue)

	notFound := analysis.Cases[2]
	gone := analysis.Cases[3]
	require.Equal(t, "widgets.show.error.not_found.404", notFound.ID)
	require.Equal(t, "widgets.show.error.gone.404", gone.ID)
	require.Equal(t, expr.StatusNotFound, notFound.StatusCode)
	require.Equal(t, expr.StatusNotFound, gone.StatusCode)
	require.Equal(t, "not_found", notFound.ErrorName)
	require.Equal(t, "gone", gone.ErrorName)
	require.Equal(t, "X-Reason", notFound.Headers[0].HTTPName)

	encoded, err := json.Marshal(responseContractSnapshotFor(analysis))
	require.NoError(t, err)
	testutil.AssertJSON(t, "testdata/golden/response_contract_cases.json.golden", encoded)
}

func TestAnalyzeResponseContractCasesRejectsUnsupportedScopes(t *testing.T) {
	base := func() *transportir.Endpoint {
		return &transportir.Endpoint{
			Service:    &transportir.Service{Name: "widgets"},
			MethodName: "show",
			Request:    &transportir.Request{},
			Response:   &transportir.Response{},
		}
	}

	cases := []struct {
		name     string
		endpoint func() *transportir.Endpoint
		code     transportir.ResponseContractLimitationCode
	}{
		{
			name:     "missing endpoint",
			endpoint: func() *transportir.Endpoint { return nil },
			code:     transportir.ResponseContractMissingEndpoint,
		},
		{
			name: "missing identity",
			endpoint: func() *transportir.Endpoint {
				endpoint := base()
				endpoint.Service = nil
				return endpoint
			},
			code: transportir.ResponseContractMissingIdentity,
		},
		{
			name: "JSON-RPC",
			endpoint: func() *transportir.Endpoint {
				endpoint := base()
				endpoint.IsJSONRPC = true
				return endpoint
			},
			code: transportir.ResponseContractJSONRPC,
		},
		{
			name: "streaming",
			endpoint: func() *transportir.Endpoint {
				endpoint := base()
				endpoint.Stream = &transportir.Stream{IsStreaming: true}
				return endpoint
			},
			code: transportir.ResponseContractStreaming,
		},
		{
			name: "redirect",
			endpoint: func() *transportir.Endpoint {
				endpoint := base()
				endpoint.Redirect = &transportir.Redirect{StatusCode: expr.StatusFound}
				return endpoint
			},
			code: transportir.ResponseContractRedirect,
		},
		{
			name: "unrepresentable multipart",
			endpoint: func() *transportir.Endpoint {
				endpoint := base()
				endpoint.Request.Multipart = true
				endpoint.Request.Body = &expr.AttributeExpr{Type: expr.String}
				return endpoint
			},
			code: transportir.ResponseContractMultipart,
		},
		{
			name: "raw request body",
			endpoint: func() *transportir.Endpoint {
				endpoint := base()
				endpoint.Request.SkipBodyEncode = true
				return endpoint
			},
			code: transportir.ResponseContractRawRequestBody,
		},
		{
			name: "raw response body",
			endpoint: func() *transportir.Endpoint {
				endpoint := base()
				endpoint.Response.SkipBodyEncode = true
				return endpoint
			},
			code: transportir.ResponseContractRawResponseBody,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			analysis := transportir.AnalyzeResponseContractCases(tc.endpoint())
			require.False(t, analysis.Supported())
			require.Empty(t, analysis.Cases)
			require.Len(t, analysis.Limitations, 1)
			require.Equal(t, tc.code, analysis.Limitations[0].Code)
			require.NotEmpty(t, analysis.Limitations[0].Detail)
		})
	}
}

func TestAnalyzeResponseContractCasesSupportsMultipart(t *testing.T) {
	root := testcodegen.RunDSL(t, responseContractMultipartDSL)
	endpoint := transportir.BuildEndpoint(root.API.HTTP.Services[0].HTTPEndpoints[0])

	analysis := transportir.AnalyzeResponseContractCases(endpoint)
	require.True(t, analysis.Supported())
	require.Empty(t, analysis.Limitations)
	require.Len(t, analysis.Cases, 2)

	want := &transportir.ResponseContractMultipartRequest{
		ContentType: "multipart/form-data",
		Parts: []transportir.ResponseContractMultipartPart{
			{Name: "file", MediaType: "application/octet-stream", Required: true},
			{Name: "label", MediaType: "text/plain", Required: true},
			{Name: "note", MediaType: "text/plain"},
		},
	}
	require.Equal(t, "imports.create.success.202", analysis.Cases[0].ID)
	require.Equal(t, want, analysis.Cases[0].Multipart)
	require.Equal(t, "imports.create.error.bad_request.400", analysis.Cases[1].ID)
	require.Equal(t, want, analysis.Cases[1].Multipart)
}

func TestAnalyzeResponseContractCasesSupportsSSE(t *testing.T) {
	root := testcodegen.RunDSL(t, responseContractSSEDSL)
	endpoint := transportir.BuildEndpoint(root.API.HTTP.Services[0].HTTPEndpoints[0])

	analysis := transportir.AnalyzeResponseContractCases(endpoint)
	require.True(t, analysis.Supported())
	require.Empty(t, analysis.Limitations)
	require.Len(t, analysis.Cases, 2)
	require.Equal(t, "events.watch.success.200", analysis.Cases[0].ID)
	require.Equal(t, []string{"text/event-stream"}, analysis.Cases[0].ContentTypes)
	require.Equal(t, transportir.ResponseContractSSETransport, analysis.Cases[0].Transport)
	require.NotNil(t, analysis.Cases[0].SSE)
	require.Equal(t, "server", analysis.Cases[0].SSE.Direction)
	require.Equal(t, "id", analysis.Cases[0].SSE.IDField)
	require.Equal(t, "event", analysis.Cases[0].SSE.EventField)
	require.Equal(t, "data", analysis.Cases[0].SSE.DataField)
	require.Equal(t, "text", analysis.Cases[0].SSE.DataEncoding)
	require.True(t, analysis.Cases[0].SSE.IDRequired)
	require.True(t, analysis.Cases[0].SSE.EventTypeRequired)
	require.Equal(t, "eof", analysis.Cases[0].SSE.Terminal)
	require.Equal(t, "events.watch.error.unauthorized.401", analysis.Cases[1].ID)
	require.Equal(t, transportir.ResponseContractHTTP, analysis.Cases[1].Transport)
	require.Nil(t, analysis.Cases[1].SSE)
}

func TestAnalyzeResponseContractCasesEscapesStableIDSegments(t *testing.T) {
	endpoint := &transportir.Endpoint{
		Service:    &transportir.Service{Name: "catalog.v2"},
		MethodName: "show/item",
		Request:    &transportir.Request{},
		Response: &transportir.Response{Responses: []*transportir.ResponseStatus{{
			StatusCode: expr.StatusAccepted,
			TagName:    "state=kind",
			TagValue:   "ready.now",
		}}},
	}

	analysis := transportir.AnalyzeResponseContractCases(endpoint)
	require.True(t, analysis.Supported())
	require.Len(t, analysis.Cases, 1)
	require.Equal(t, "catalog%2Ev2.show%2Fitem.success.202.state%3Dkind=ready%2Enow", analysis.Cases[0].ID)
}

func TestAnalyzeResponseContractCasesTracksBodyApplicability(t *testing.T) {
	endpoint := &transportir.Endpoint{
		Service:    &transportir.Service{Name: "files"},
		MethodName: "download",
		Request:    &transportir.Request{},
		Response: &transportir.Response{
			Responses: []*transportir.ResponseStatus{{
				StatusCode:   expr.StatusNoContent,
				ContentTypes: []string{"application/json"},
				Body:         &expr.AttributeExpr{Type: expr.Empty},
			}},
		},
	}

	analysis := transportir.AnalyzeResponseContractCases(endpoint)
	require.True(t, analysis.Supported())
	require.Len(t, analysis.Cases, 1)
	require.False(t, analysis.Cases[0].HasBody)
}

func TestAnalyzeResponseContractCasesTreatsOnlyFileSuccessAsBody(t *testing.T) {
	endpoint := &transportir.Endpoint{
		Service:    &transportir.Service{Name: "files"},
		MethodName: "download",
		Request:    &transportir.Request{},
		Response: &transportir.Response{
			FileResponse: true,
			Responses: []*transportir.ResponseStatus{{
				StatusCode:   expr.StatusOK,
				ContentTypes: []string{"*/*"},
				Body:         &expr.AttributeExpr{Type: expr.Empty},
			}},
			ErrorResponses: []*transportir.ResponseStatus{{
				Error:        &transportir.Error{Name: "not_found"},
				StatusCode:   expr.StatusNotFound,
				ContentTypes: []string{"application/json"},
				Body:         &expr.AttributeExpr{Type: expr.Empty},
				IsError:      true,
			}},
		},
	}

	analysis := transportir.AnalyzeResponseContractCases(endpoint)
	require.True(t, analysis.Supported())
	require.Len(t, analysis.Cases, 2)
	require.True(t, analysis.Cases[0].HasBody)
	require.False(t, analysis.Cases[1].HasBody)
}

func responseContractSnapshotFor(analysis *transportir.ResponseContractAnalysis) responseContractSnapshot {
	snapshot := responseContractSnapshot{Limitations: analysis.Limitations}
	for _, contractCase := range analysis.Cases {
		snapshot.Cases = append(snapshot.Cases, responseContractCaseSnapshot{
			ID:           contractCase.ID,
			Kind:         contractCase.Kind,
			StatusCode:   contractCase.StatusCode,
			ErrorName:    contractCase.ErrorName,
			TagName:      contractCase.TagName,
			TagValue:     contractCase.TagValue,
			ContentTypes: contractCase.ContentTypes,
			Headers:      contractCase.Headers,
			Cookies:      contractCase.Cookies,
			Multipart:    contractCase.Multipart,
		})
	}
	return snapshot
}

func responseContractMultipartDSL() {
	dsl.Service("imports", func() {
		dsl.Method("create", func() {
			dsl.Payload(func() {
				dsl.Attribute("file", dsl.Bytes)
				dsl.Attribute("label", dsl.String)
				dsl.Attribute("note", dsl.String)
				dsl.Required("file", "label")
			})
			dsl.Error("bad_request")
			dsl.HTTP(func() {
				dsl.POST("/imports")
				dsl.MultipartRequest()
				dsl.Response(expr.StatusAccepted)
				dsl.Response("bad_request", expr.StatusBadRequest)
			})
		})
	})
}

func responseContractDSL() {
	dsl.Service("widgets", func() {
		dsl.Method("show", func() {
			dsl.Result(func() {
				dsl.Attribute("body", dsl.String)
				dsl.Attribute("outcome", dsl.String)
				dsl.Attribute("version", dsl.String)
				dsl.Attribute("session", dsl.String)
				dsl.Required("body", "outcome", "version", "session")
			})
			dsl.Error("not_found", func() {
				dsl.Attribute("reason", dsl.String)
				dsl.Required("reason")
			})
			dsl.Error("gone")
			dsl.HTTP(func() {
				dsl.GET("/widgets")
				dsl.Response(expr.StatusAccepted, func() {
					dsl.Tag("outcome", "accepted")
					dsl.Body("body")
					dsl.Header("version:X-Version")
					dsl.SessionCookie("session:widget_session")
				})
				dsl.Response(expr.StatusOK, func() {
					dsl.Body("body")
				})
				dsl.Response("not_found", expr.StatusNotFound, func() {
					dsl.Header("reason:X-Reason")
				})
				dsl.Response("gone", expr.StatusNotFound)
			})
		})
	})
}

func responseContractSSEDSL() {
	dsl.Service("events", func() {
		dsl.Method("watch", func() {
			dsl.Error("unauthorized")
			dsl.StreamingResult(func() {
				dsl.Attribute("id", dsl.String)
				dsl.Attribute("event", dsl.String)
				dsl.Attribute("data", dsl.String)
				dsl.Required("id", "event", "data")
			})
			dsl.HTTP(func() {
				dsl.GET("/events")
				dsl.Response("unauthorized", expr.StatusUnauthorized)
				dsl.ServerSentEvents(func() {
					dsl.SSEEventID("id")
					dsl.SSEEventType("event")
					dsl.SSEEventData("data")
				})
			})
		})
	})
}
