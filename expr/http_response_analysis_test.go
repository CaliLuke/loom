package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestCapturedHTTPStatusDomains(t *testing.T) {
	for _, tc := range []struct {
		name       string
		selector   any
		body       string
		exactFirst bool
		error404   bool
		constrain  bool
		want       []int
	}{
		{"literal without status field", StatusOK, "literal", false, false, false, []int{200}},
		{"body class", Status2XX, "object", false, false, false, statusRange(200, 299, 204)},
		{"bodyless class", Status2XX, "implicit empty", false, false, false, statusRange(200, 299)},
		{"bodyless redirect", Status3XX, "implicit empty", false, false, false, statusRange(300, 399)},
		{"explicit empty object", Status2XX, "explicit object", false, false, false, statusRange(200, 299, 204)},
		{"typed additional object", Status3XX, "additional", false, false, false, statusRange(300, 399, 304)},
		{"explicit Empty", Status2XX, "empty", false, false, false, statusRange(200, 299)},
		{"selected native body", Status2XX, "selected", false, false, false, statusRange(200, 299, 204)},
		{"exact shadows class", Status2XX, "object", true, false, false, statusRange(201, 299, 204)},
		{"error shadows fallback", StatusFallback, "object", false, true, false, statusRange(200, 599, 204, 304, 404)},
		{"constraints do not narrow ownership", Status2XX, "object", false, false, true, statusRange(200, 299, 204)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				empty := Type("EmptyBody", func() {
					Meta("openapi:additionalProperties", "false")
				})
				value := Type("Value", func() {
					Attribute("status", Int, func() {
						if tc.constrain {
							Enum(201)
						}
					})
					Required("status")
					if tc.body == "object" || tc.body == "selected" || tc.body == "literal" {
						Attribute("message", String)
					}
					if tc.body == "additional" {
						AdditionalAttributes(String)
					}
				})
				Service("domains", func() {
					Method("read", func() {
						Payload(value)
						Result(value)
						if tc.error404 {
							Error("missing")
						}
						HTTP(func() {
							POST("/")
							if tc.exactFirst {
								Response(StatusOK, func() {
									StatusField("status")
								})
							}
							Response(tc.selector, func() {
								if tc.body != "literal" {
									StatusField("status")
								}
								switch tc.body {
								case "explicit object":
									Body(empty)
								case "empty":
									Body(Empty)
								case "selected":
									Body("message")
								}
							})
							if tc.error404 {
								Response("missing", StatusNotFound)
							}
						})
					})
				})
			})
			e := root.API.HTTP.Service("domains").Endpoint("read")
			r := e.Responses[len(e.Responses)-1]
			require.Equal(t, tc.want, r.StatusDomain())
			got := r.StatusDomain()
			got[0] = 999
			require.Equal(t, tc.want, r.StatusDomain())
			require.NotNil(t, e.Body.Find("status"))
			require.True(t, e.MethodExpr.Payload.IsRequired("status"))
			if tc.exactFirst {
				require.Equal(t, []int{200}, e.Responses[0].StatusDomain())
			}
			if tc.error404 {
				require.Equal(t, []int{404}, e.HTTPErrors[0].Response.StatusDomain())
			}
		})
	}
}

func TestCapturedInformationalDomains(t *testing.T) {
	root := expr.RunDSL(t, func() {
		empty := Type("Hints", func() {
		})
		status := Type("Status", func() {
			Attribute("status", Int)
			Required("status")
		})
		Service("hints", func() {
			Method("read", func() {
				HTTP(func() {
					GET("/")
					Informational("Exact", StatusEarlyHints, empty, func() {
					})
					Informational("Class", Status1XX, status, func() {
						StatusField("status")
					})
					Response(StatusNoContent)
				})
			})
		})
	})
	e := root.API.HTTP.Service("hints").Endpoint("read")
	wantKeys := []expr.HTTPStatusSelector{{Code: 103}, {Class: expr.Status1XX}}
	require.Equal(t, wantKeys, e.InformationalStatusKeys())
	require.Equal(t, []int{103}, e.Informational[0].StatusDomain())
	require.Equal(t, statusRange(102, 199, 103), e.Informational[1].StatusDomain())
	selected, ok := expr.SelectInformational(e.InformationalStatusKeys(), 100)
	require.True(t, ok)
	require.Equal(t, expr.HTTPStatusSelector{Class: expr.Status1XX}, selected)
	require.NotContains(t, e.Informational[1].StatusDomain(), 100)
	require.NotContains(t, e.Informational[1].StatusDomain(), 101)
	keys := e.InformationalStatusKeys()
	keys[0] = expr.HTTPStatusSelector{Code: 199}
	require.Equal(t, wantKeys, e.InformationalStatusKeys())
}

func TestResponseAnalysisProtocolDomains(t *testing.T) {
	root := expr.RunDSL(t, func() {
		Service("rpc", func() {
			JSONRPC(func() {
				POST("/rpc")
			})
			Method("call", func() {
				Result(String)
				Error("zero")
				JSONRPC(func() {
					Response("zero", 0)
				})
			})
		})
	})
	e := root.API.JSONRPC.Service("rpc").Endpoint("call")
	require.Equal(t, []expr.HTTPStatusSelector{{Code: 200}}, e.FinalStatusKeys())
	require.Equal(t, []int{200}, e.Responses[0].StatusDomain())
	require.Zero(t, e.HTTPErrors[0].Response.JSONRPCError.Code)
	require.Nil(t, e.HTTPErrors[0].Response.StatusDomain())
}

func statusRange(first, last int, excluded ...int) []int {
	codes := make([]int, 0, last-first+1)
	for code := first; code <= last; code++ {
		skip := false
		for _, value := range excluded {
			if code == value {
				skip = true
			}
		}
		if !skip {
			codes = append(codes, code)
		}
	}
	return codes
}

func TestErrorOnlyStatusDomains(t *testing.T) {
	root := expr.RunDSL(t, func() {
		failure := Type("Failure", func() {
			ErrorName("name", String)
			Attribute("status", Int)
			Required("name", "status")
		})
		Service("errors", func() {
			Method("read", func() {
				Error("class", failure)
				Error("fallback", failure)
				HTTP(func() {
					GET("/")
					NoSuccessResponse()
					Response("class", Status4XX, func() {
						StatusField("status")
					})
					Response("fallback", StatusFallback, func() {
						StatusField("status")
					})
				})
			})
		})
	})
	e := root.API.HTTP.Service("errors").Endpoint("read")
	require.Empty(t, e.Responses)
	require.Equal(t, []expr.HTTPStatusSelector{{Class: expr.Status4XX}, {Class: expr.StatusFallback}}, e.FinalStatusKeys())
	require.Equal(t, statusRange(400, 499), e.HTTPErrors[0].Response.StatusDomain())
	want := append(statusRange(200, 399, 204, 304), statusRange(500, 599)...)
	require.Equal(t, want, e.HTTPErrors[1].Response.StatusDomain())
}
