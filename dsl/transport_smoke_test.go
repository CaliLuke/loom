package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestTransportDSLComprehensiveSmoke(t *testing.T) {
	root := expr.RunDSL(t, func() {
		API("inventory", func() {
			Title("Inventory API")
			Description("Inventory management API")
			Version("1.2.3")
			TermsOfService("https://example.com/terms")
			Contact(func() {
				Name("Support")
				Email("support@example.com")
				URL("https://example.com/support")
			})
			License(func() {
				Name("MIT")
				URL("https://example.com/license")
			})
			Docs(func() {
				Description("Reference docs")
				URL("https://example.com/docs")
			})
			Randomizer(expr.NewDeterministicRandomizer())
			HTTP(func() {
				Path("/api")
				Consumes("application/json")
				Produces("application/json")
			})
		})

		Service("widgets", func() {
			Error("not_found", ErrorResult, "widget was not found", func() {
				Temporary()
				Timeout()
				Fault()
				Remedy(func() {
					RemedyCode("WIDGET_NOT_FOUND")
					SafeMessage("Widget not found")
					RetryHint("Retry after refresh")
				})
			})
			JSONRPC(func() {
				POST("/rpc")
				Response("not_found", RPCMethodNotFound)
			})

			Method("show", func() {
				Payload(func() {
					ID("requestID", String, "JSON-RPC request ID")
					Attribute("id", String)
					Attribute("traceID", String)
					Attribute("session", String)
					Attribute("verbose", Boolean)
					Required("id", "requestID")
				})
				Result(func() {
					ID("requestID", String, "JSON-RPC request ID")
					Attribute("id", String)
					Attribute("href", String)
					Attribute("state", String)
					Attribute("session", String)
				})
				HTTP(func() {
					GET("/widgets/{id}")
					Header("traceID:X-Trace-ID", String)
					Cookie("session:loom-session", String)
					Params(func() {
						Param("verbose", Boolean)
					})
					Response(StatusCreated, func() {
						Description("created response")
						ContentType("application/json")
						Header("href")
						SessionCookie("session:loom-session", String)
						CookieMaxAge(3600)
						CookieDomain("example.com")
						CookiePath("/")
						CookieSecure()
						CookieHTTPOnly()
						CookieSameSite(CookieSameSiteStrict)
						Link("self", func() {
							LinkOperation("show")
							LinkParam("id", "$response.body#/id")
							LinkRequestBody("$response.body")
						})
					})
					Response("not_found", StatusNotFound)
				})
			})

			Method("rpcShow", func() {
				Payload(func() {
					ID("requestID", String, "JSON-RPC request ID")
					Attribute("id", String)
					Required("id", "requestID")
				})
				Result(func() {
					ID("requestID", String, "JSON-RPC request ID")
					Attribute("id", String)
				})
				JSONRPC(func() {})
			})

			Method("legacy", func() {
				Payload(func() {
					Attribute("id", String)
				})
				Result(Empty)
				HTTP(func() {
					GET("/legacy/{id}")
					Redirect("/widgets/{id}", StatusMovedPermanently)
				})
			})
		})
	})

	require.Equal(t, "Inventory API", root.API.Title)
	require.Equal(t, "1.2.3", root.API.Version)
	require.Equal(t, "https://example.com/terms", root.API.TermsOfService)
	require.Equal(t, "Support", root.API.Contact.Name)
	require.Equal(t, "support@example.com", root.API.Contact.Email)
	require.Equal(t, "MIT", root.API.License.Name)
	require.Equal(t, "https://example.com/docs", root.API.Docs.URL)
	require.NotNil(t, root.API.ExampleGenerator)
	require.Equal(t, "/api", root.API.HTTP.Path)
	require.Equal(t, []string{"application/json"}, root.API.HTTP.Consumes)
	require.Equal(t, []string{"application/json"}, root.API.HTTP.Produces)

	svc := root.Service("widgets")
	require.NotNil(t, svc)
	require.Len(t, svc.Errors, 1)
	require.NotNil(t, svc.Errors[0].Remedy)
	require.Equal(t, "WIDGET_NOT_FOUND", svc.Errors[0].Remedy.Code)
	require.Equal(t, "Widget not found", svc.Errors[0].Remedy.SafeMessage)

	httpSvc := root.API.HTTP.Service("widgets")
	require.NotNil(t, httpSvc)

	showEndpoint := httpSvc.Endpoint("show")
	require.NotNil(t, showEndpoint)
	require.Len(t, showEndpoint.Routes, 1)
	require.Equal(t, "GET", showEndpoint.Routes[0].Method)
	require.Equal(t, "/widgets/{id}", showEndpoint.Routes[0].Path)
	require.Len(t, showEndpoint.Responses, 1)
	require.Len(t, showEndpoint.HTTPErrors, 1)
	require.Equal(t, expr.StatusCreated, showEndpoint.Responses[0].StatusCode)
	require.Equal(t, "created response", showEndpoint.Responses[0].Description)
	require.Equal(t, "application/json", showEndpoint.Responses[0].ContentType)
	require.Len(t, showEndpoint.Responses[0].Cookies, 1)
	require.Equal(t, "loom-session", showEndpoint.Responses[0].Cookies[0].HTTPName())
	require.Equal(t, "3600", showEndpoint.Responses[0].Cookies[0].MaxAge)
	require.Equal(t, "example.com", showEndpoint.Responses[0].Cookies[0].Domain)
	require.Equal(t, "/", showEndpoint.Responses[0].Cookies[0].Path)
	require.True(t, showEndpoint.Responses[0].Cookies[0].Secure)
	require.True(t, showEndpoint.Responses[0].Cookies[0].HTTPOnly)
	require.Equal(t, expr.CookieSameSiteStrict, showEndpoint.Responses[0].Cookies[0].SameSite)
	require.Len(t, showEndpoint.Responses[0].Links, 1)
	require.Equal(t, "show", showEndpoint.Responses[0].Links[0].Operation)
	require.Equal(t, "$response.body#/id", showEndpoint.Responses[0].Links[0].Parameters["id"])
	require.Equal(t, "$response.body", showEndpoint.Responses[0].Links[0].RequestBody)
	require.Equal(t, "not_found", showEndpoint.HTTPErrors[0].Name)
	require.Equal(t, expr.StatusNotFound, showEndpoint.HTTPErrors[0].Response.StatusCode)

	jsonrpcSvc := root.API.JSONRPC.Service("widgets")
	require.NotNil(t, jsonrpcSvc)
	jsonrpcShow := jsonrpcSvc.Endpoint("rpcShow")
	require.NotNil(t, jsonrpcShow)
	require.Len(t, jsonrpcShow.HTTPErrors, 1)
	require.Equal(t, expr.RPCMethodNotFound, jsonrpcShow.HTTPErrors[0].Response.StatusCode)

	legacyEndpoint := httpSvc.Endpoint("legacy")
	require.NotNil(t, legacyEndpoint)
	require.NotNil(t, legacyEndpoint.Redirect)
	require.Equal(t, "/widgets/{id}", legacyEndpoint.Redirect.URL)
	require.Equal(t, expr.StatusMovedPermanently, legacyEndpoint.Redirect.StatusCode)
}

func TestOptionalRequestBodyDSL(t *testing.T) {
	root := expr.RunDSL(t, func() {
		var LinkStart = Type("LinkStart", func() {
			Attribute("continue", String)
		})

		Service("optional", func() {
			Method("begin", func() {
				Payload(LinkStart)
				Result(Empty)
				HTTP(func() {
					POST("/optional")
					Body(LinkStart)
					OptionalRequestBody()
				})
			})
		})
	})

	endpoint := root.API.HTTP.Service("optional").Endpoint("begin")
	require.NotNil(t, endpoint)
	require.True(t, endpoint.OptionalRequestBody)
	require.NotNil(t, endpoint.Body)
	require.True(t, expr.IsObject(endpoint.Body.Type))
}

func TestMultipartAndFormRequestDSL(t *testing.T) {
	root := expr.RunDSL(t, func() {
		Service("uploads", func() {
			Method("multipart", func() {
				Payload(func() {
					Attribute("blob", Bytes)
				})
				Result(func() {
					Attribute("ok", Boolean)
				})
				HTTP(func() {
					POST("/uploads/multipart")
					MultipartRequest()
					Response(StatusAccepted)
				})
			})

			Method("form", func() {
				Payload(func() {
					Attribute("name", String)
					Attribute("size", Int)
				})
				Result(func() {
					Attribute("ok", Boolean)
				})
				HTTP(func() {
					POST("/uploads/form")
					FormRequest()
					Response(StatusCreated)
				})
			})
		})
	})

	httpSvc := root.API.HTTP.Service("uploads")
	require.NotNil(t, httpSvc)

	multipartEndpoint := httpSvc.Endpoint("multipart")
	require.NotNil(t, multipartEndpoint)
	require.True(t, multipartEndpoint.MultipartRequest)
	require.False(t, multipartEndpoint.FormRequest)
	require.Equal(t, expr.StatusAccepted, multipartEndpoint.Responses[0].StatusCode)

	formEndpoint := httpSvc.Endpoint("form")
	require.NotNil(t, formEndpoint)
	require.True(t, formEndpoint.FormRequest)
	require.False(t, formEndpoint.MultipartRequest)
	require.Equal(t, expr.StatusCreated, formEndpoint.Responses[0].StatusCode)
}

func TestSkipEncodeDecodeDSL(t *testing.T) {
	root := expr.RunDSL(t, func() {
		Service("passthrough", func() {
			Method("ingest", func() {
				Payload(func() {
					Attribute("traceID", String)
					Attribute("session", String)
					Required("traceID", "session")
				})
				Result(func() {
					Attribute("requestID", String)
					Required("requestID")
				})
				HTTP(func() {
					POST("/passthrough")
					Header("traceID:X-Trace-ID", String)
					Cookie("session:loom-session", String)
					SkipRequestBodyEncodeDecode()
					SkipResponseBodyEncodeDecode()
					Response(StatusAccepted, func() {
						Header("requestID:X-Request-ID", String)
					})
				})
			})
		})
	})

	endpoint := root.API.HTTP.Service("passthrough").Endpoint("ingest")
	require.NotNil(t, endpoint)
	require.True(t, endpoint.SkipRequestBodyEncodeDecode)
	require.True(t, endpoint.SkipResponseBodyEncodeDecode)
	require.NotNil(t, endpoint.Body)
	require.Equal(t, expr.StatusAccepted, endpoint.Responses[0].StatusCode)
	require.NotNil(t, endpoint.Responses[0].Body)
	require.NotNil(t, endpoint.Responses[0].Headers)
	require.False(t, endpoint.Responses[0].Headers.IsEmpty())
	headerName, ok := endpoint.Responses[0].Headers.FindKey("requestID")
	require.True(t, ok)
	require.Equal(t, "X-Request-ID", headerName)
}
