package expr_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestSSEWireStatusDomains(t *testing.T) {
	for _, tc := range []struct {
		name     string
		selector any
		want     []int
	}{
		{"success class", Status2XX, statusRange(200, 299, 204)},
		{"redirect class", Status3XX, statusRange(300, 399, 304)},
		{"fallback", StatusFallback, statusRange(200, 599, 204, 304)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := expr.RunDSL(t, wireDomainDesign(tc.selector, true, true, true, 0))
			e := root.API.HTTP.Service("wire_domain").Endpoint("read")
			r := e.Responses[0]
			require.Equal(t, tc.want, r.StatusDomain())
			require.Equal(t, expr.Empty, r.Body.Type, "handshake metadata body remains empty")
			require.Equal(t, expr.String, e.MethodExpr.StreamingResult.Type, "event source is unchanged")
			require.True(t, r.Handshake.IsRequired("status"))
		})
	}
}

func TestSSEWireStatusRejection(t *testing.T) {
	for _, tc := range []struct {
		name        string
		selector    any
		typed       bool
		statusField bool
		constraint  int
		message     string
	}{
		{"typed exact204", 204, true, false, 0, "does not allow response body"},
		{"typed exact304", 304, true, false, 0, "does not allow response body"},
		{"legacy exact204", 204, false, false, 0, "does not allow response body"},
		{"legacy exact304", 304, false, false, 0, "does not allow response body"},
		{"class only204", Status2XX, true, true, 204, "constraints exclude every owned status"},
		{"class only304", Status3XX, true, true, 304, "constraints exclude every owned status"},
		{"fallback only204", StatusFallback, true, true, 204, "constraints exclude every owned status"},
		{"fallback only304", StatusFallback, true, true, 304, "constraints exclude every owned status"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, wireDomainDesign(tc.selector, true, tc.typed, tc.statusField, tc.constraint))
			require.Contains(t, strings.ToLower(err.Error()), tc.message)
		})
	}
}

func TestOrdinaryBodylessWireStatusDomains(t *testing.T) {
	for _, tc := range []struct {
		name       string
		selector   any
		field      bool
		constraint int
		want       []int
	}{
		{"exact204", 204, false, 0, []int{204}},
		{"exact304", 304, false, 0, []int{304}},
		{"class204", Status2XX, true, 204, statusRange(200, 299)},
		{"class304", Status3XX, true, 304, statusRange(300, 399)},
		{"fallback", StatusFallback, true, 204, statusRange(200, 599)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := expr.RunDSL(t, wireDomainDesign(tc.selector, false, false, tc.field, tc.constraint))
			r := root.API.HTTP.Service("wire_domain").Endpoint("read").Responses[0]
			require.Equal(t, tc.want, r.StatusDomain())
			require.Equal(t, expr.Empty, r.Body.Type)
		})
	}
}

func TestSSEErrorWireStatusDomains(t *testing.T) {
	for _, body := range []bool{false, true} {
		name := "bodyless error"
		if body {
			name = "body-bearing error"
		}
		t.Run(name, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				failure := Type("Failure", func() {
					ErrorName("name", String)
					Attribute("status", Int)
					Required("name", "status")
					if body {
						Attribute("message", String)
					}
				})
				Service("wire_errors", func() {
					Method("read", func() {
						StreamingResult(String)
						Error("failure", failure)
						HTTP(func() {
							GET("/")
							ServerSentEvents()
							Response(StatusOK)
							Response("failure", StatusFallback, func() {
								StatusField("status")
								Header("name:X-Error")
							})
						})
					})
				})
			})
			e := root.API.HTTP.Service("wire_errors").Endpoint("read")
			want := statusRange(201, 599)
			if body {
				want = statusRange(201, 599, 204, 304)
			}
			require.Equal(t, want, e.HTTPErrors[0].Response.StatusDomain())
			require.Equal(t, []int{200}, e.Responses[0].StatusDomain())
		})
	}
}

func wireDomainDesign(selector any, sse, typed, statusField bool, constraint int) func() {
	return func() {
		value := Type("Opening", func() {
			if statusField {
				Attribute("status", Int, func() {
					if constraint != 0 {
						Enum(constraint)
					}
				})
				Required("status")
			}
		})
		Service("wire_domain", func() {
			Method("read", func() {
				if sse {
					StreamingResult(String)
				} else if statusField {
					Result(value)
				}
				HTTP(func() {
					GET("/")
					if sse {
						ServerSentEvents()
					}
					bindings := func() {
						if statusField {
							StatusField("status")
						}
					}
					if typed {
						Response(selector, value, bindings)
					} else {
						Response(selector, bindings)
					}
				})
			})
		})
	}
}

func TestDefaultResponseWireStatus(t *testing.T) {
	for _, tc := range []struct {
		name string
		want int
	}{
		{"endpoint SSE", 200},
		{"service SSE", 200},
		{"API SSE", 200},
		{"empty unary", 204},
		{"empty object unary", 200},
		{"headers only unary", 200},
		{"raw unary", 200},
		{"file unary", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				emptyObject := Type("EmptyObject", func() {
					Meta("openapi:additionalProperties", "false")
				})
				API("default_wire", func() {
					if tc.name == "API SSE" {
						HTTP(func() {
							ServerSentEvents()
						})
					}
				})
				Service("default_wire", func() {
					if tc.name == "service SSE" {
						HTTP(func() {
							ServerSentEvents()
						})
					}
					Method("read", func() {
						switch tc.name {
						case "empty object unary":
							Result(emptyObject)
						case "headers only unary":
							Result(func() {
								Attribute("trace", String)
								Required("trace")
							})
						default:
							Result(Empty)
						}
						if strings.HasSuffix(tc.name, "SSE") {
							StreamingResult(String)
						}
						HTTP(func() {
							GET("/")
							switch tc.name {
							case "endpoint SSE":
								ServerSentEvents()
							case "headers only unary":
								Response(func() {
									Header("trace:X-Trace")
								})
							case "raw unary":
								SkipResponseBodyEncodeDecode()
							case "file unary":
								FileResponse()
							}
						})
					})
				})
			})
			e := root.API.HTTP.Service("default_wire").Endpoint("read")
			require.Len(t, e.Responses, 1)
			r := e.Responses[0]
			require.Equal(t, expr.HTTPStatusSelector{Code: tc.want}, r.Status)
			require.Equal(t, []int{tc.want}, r.StatusDomain())
			require.Nil(t, r.AuthoredBody())
			if tc.name == "empty object unary" || tc.name == "headers only unary" {
				require.NotEqual(t, expr.Empty, e.MethodExpr.Result.Type)
				require.Equal(t, expr.Empty, r.Body.Type)
			} else {
				require.Equal(t, expr.Empty, e.MethodExpr.Result.Type)
			}
			if strings.HasSuffix(tc.name, "SSE") {
				require.NotNil(t, e.SSE)
				require.True(t, e.MethodExpr.HasMixedResults())
				require.Equal(t, expr.String, e.MethodExpr.StreamingResult.Type)
			}
		})
	}
}

func TestDefaultRedirectBodyStatus(t *testing.T) {
	for _, body := range []bool{false, true} {
		name := "bodyless304"
		if body {
			name = "object304"
		}
		t.Run(name, func(t *testing.T) {
			design := func() {
				Service("redirect_wire", func() {
					Method("read", func() {
						if body {
							Result(func() {
								Attribute("message", String)
							})
						}
						HTTP(func() {
							GET("/")
							Redirect("/x", StatusNotModified)
						})
					})
				})
			}
			if body {
				err := expr.RunInvalidDSL(t, design)
				require.Contains(t, err.Error(), "status code 304 which does not allow response body")
				return
			}
			root := expr.RunDSL(t, design)
			e := root.API.HTTP.Service("redirect_wire").Endpoint("read")
			require.Equal(t, expr.HTTPStatusSelector{Code: 304}, e.Responses[0].Status)
			require.Equal(t, []int{304}, e.Responses[0].StatusDomain())
		})
	}
}
