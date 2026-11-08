package transportir_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	testcodegen "github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func TestResponseOccurrenceFacts(t *testing.T) {
	root := testcodegen.RunDSL(t, func() {
		opening := Type("Opening", func() {
			Attribute("status", Int)
			Attribute("session", String)
			Required("status", "session")
		})
		Service("facts", func() {
			Method("read", func() {
				Result(func() {
					Attribute("status", Int)
					Attribute("message", String)
					Required("status", "message")
				})
				HTTP(func() {
					GET("/read")
					Response(StatusOK, func() {
						StatusField("status")
					})
					Response(Status2XX, func() {
						StatusField("status")
					})
				})
			})
			Method("watch", func() {
				StreamingResult(func() {
					Attribute("status", String)
					Attribute("session", Int)
					Required("status", "session")
				})
				HTTP(func() {
					GET("/watch")
					ServerSentEvents()
					Response(StatusCreated, opening, func() {
						StatusField("status")
						Header("session:X-Session")
					})
				})
			})
		})
	})
	for _, endpoint := range root.API.HTTP.Services[0].HTTPEndpoints {
		t.Run(endpoint.Name(), func(t *testing.T) {
			ir := transportir.BuildEndpoint(endpoint)
			require.Equal(t, endpoint.FinalStatusKeys(), ir.Response.FinalKeys)
			require.Equal(t, endpoint.StatusSelected(), ir.Response.StatusSelected)
			for i, r := range endpoint.Responses {
				actual := ir.Response.Responses[i]
				require.Same(t, r, actual.Occurrence)
				carrier := endpoint.MethodExpr.Result
				if r.Handshake != nil {
					carrier = r.Handshake
				}
				require.Same(t, carrier, actual.Carrier)
				require.Equal(t, r.Handshake != nil, actual.Handshake)
				require.NotNil(t, actual.StatusField)
				require.Equal(t, "status", actual.StatusField.Name)
				require.Same(t, carrier.Find("status"), actual.StatusField.Attribute)
				require.ElementsMatch(t, r.RequiredFields(""), actual.Required[""])
				actual.Required[""][0] = "corrupt"
				require.NotContains(t, r.RequiredFields(""), "corrupt")
				if actual.Handshake {
					require.Equal(t, expr.Empty, actual.Body.Type)
					require.Equal(t, expr.String, ir.Stream.ResponseMessage.Find("status").Type)
					require.Equal(t, expr.Int, ir.Stream.ResponseMessage.Find("session").Type)
				}
			}
			ir.Response.FinalKeys[0] = expr.HTTPStatusSelector{Code: 599}
			require.NotEqual(t, ir.Response.FinalKeys, endpoint.FinalStatusKeys())
		})
	}
}

func TestInformationalOccurrenceFacts(t *testing.T) {
	root := testcodegen.RunDSL(t, func() {
		hints := Type("Hints", func() {
			Attribute("status", Int)
			Attribute("link", String)
			Required("status", "link")
		})
		Service("facts", func() {
			Method("read", func() {
				HTTP(func() {
					GET("/")
					Informational("early", StatusEarlyHints, hints, func() {
						StatusField("status")
						Header("link:Link")
					})
					Informational("progress", Status1XX, hints, func() {
						StatusField("status")
						Header("link:Link")
					})
				})
			})
		})
	})
	endpoint := root.API.HTTP.Services[0].HTTPEndpoints[0]
	ir := transportir.BuildEndpoint(endpoint)
	require.Equal(t, endpoint.InformationalStatusKeys(), ir.Response.InformationalKeys)
	require.Len(t, ir.Response.Informational, 2)
	for i, info := range endpoint.Informational {
		actual := ir.Response.Informational[i]
		require.Equal(t, info.Name, actual.Name)
		require.Same(t, info.HTTPResponseExpr, actual.Response.Occurrence)
		require.Same(t, info.Value, actual.Response.Carrier)
		require.Equal(t, info.StatusDomain(), actual.Response.Domain)
		require.ElementsMatch(t, info.RequiredFields(""), actual.Response.Required[""])
		require.False(t, actual.Response.Handshake)
		require.Nil(t, actual.Response.Body)
		require.NotNil(t, actual.Response.StatusField)
	}
	domain := ir.Response.Informational[1].Response.Domain
	require.Contains(t, domain, 102)
	for _, excluded := range []int{100, 101, 103} {
		require.NotContains(t, domain, excluded)
	}
	ir.Response.InformationalKeys[0] = expr.HTTPStatusSelector{Code: 199}
	require.NotEqual(t, ir.Response.InformationalKeys, endpoint.InformationalStatusKeys())
}

func TestResponseOccurrenceViewsAndErrors(t *testing.T) {
	root := testcodegen.RunDSL(t, func() {
		viewed := ResultType("application/vnd.occurrence", func() {
			Attributes(func() {
				Attribute("status", Int)
				Attribute("message", String)
				Required("status", "message")
			})
			View("default", func() {
				Attribute("status")
				Attribute("message")
			})
			View("summary", func() {
				Attribute("status")
				Attribute("message")
			})
		})
		Service("facts", func() {
			Method("read", func() {
				Result(viewed)
				HTTP(func() {
					GET("/")
					Response(StatusOK, func() {
						StatusField("status")
					})
				})
			})
			Method("gone", func() {
				Error("gone", String)
				HTTP(func() {
					GET("/gone")
					NoSuccessResponse()
					Response("gone", StatusGone)
				})
			})
		})
	})
	read := root.API.HTTP.Services[0].Endpoint("read")
	response := transportir.BuildEndpoint(read).Response.Responses[0]
	for _, view := range []string{"", "default", "summary"} {
		require.ElementsMatch(t, []string{"status", "message"}, response.Required[view])
		response.Required[view][0] = "changed"
		require.NotContains(t, read.Responses[0].RequiredFields(view), "changed")
	}
	gone := root.API.HTTP.Services[0].Endpoint("gone")
	ir := transportir.BuildEndpoint(gone).Response
	require.True(t, ir.NoSuccess)
	require.Empty(t, ir.Responses)
	require.Same(t, gone.HTTPErrors[0].AttributeExpr, ir.ErrorResponses[0].Carrier)
	require.Equal(t, []expr.HTTPStatusSelector{{Code: 410}}, ir.FinalKeys)
}
