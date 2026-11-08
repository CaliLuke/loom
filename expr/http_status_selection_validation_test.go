package expr_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestHTTPStatusSelectorAdmission(t *testing.T) {
	for _, tc := range []struct {
		name string
		want []string
	}{
		{name: "exact informational"},
		{name: "class informational"},
		{name: "exact and class success"},
		{name: "fallback success"},
		{name: "error only"},
		{name: "SSE exact created"},
		{name: "SSE two exact"},
		{name: "SSE class"},
		{name: "native class error"},
		{name: "missing status field", want: []string{"statusfield"}},
		{name: "optional status field", want: []string{"statusfield", "required"}},
		{name: "nullable status field", want: []string{"statusfield", "nullable"}},
		{name: "string status field", want: []string{"statusfield", "int"}},
		{name: "defaulted status field", want: []string{"statusfield", "default"}},
		{name: "excluded status domain", want: []string{"statusfield", "status"}},
		{name: "status header collision", want: []string{"statusfield", "header"}},
		{name: "mixed class key", want: []string{"success", "error"}},
		{name: "mixed exact key", want: []string{"success", "error"}},
		{name: "built in class error", want: []string{"statusfield"}},
		{name: "tag with class", want: []string{"tag"}},
		{name: "informational as final", want: []string{"informational"}},
		{name: "upgrade as final", want: []string{"101"}},
		{name: "JSONRPC class", want: []string{"json-rpc"}},
		{name: "file class", want: []string{"fileresponse"}},
		{name: "SSE tag", want: []string{"tag"}},
		{name: "error only with result", want: []string{"nosuccessresponse", "result"}},
		{name: "informational content", want: []string{"body"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := statusSelectorAdmissionDesign(tc.name)
			if len(tc.want) == 0 {
				expr.RunDSL(t, design)
				return
			}
			err := expr.RunInvalidDSL(t, design)
			for _, fragment := range tc.want {
				require.Contains(t, strings.ToLower(err.Error()), fragment)
			}
		})
	}
}

func statusSelectorAdmissionDesign(mode string) func() {
	return func() {
		carrier := Type("StatusValue", func() {
			if mode == "string status field" {
				Attribute("status", String)
			} else {
				Attribute("status", Int, func() {
					switch mode {
					case "nullable status field":
						Nullable()
					case "defaulted status field":
						Default(400)
					case "excluded status domain":
						Enum(600)
					}
				})
			}
			Attribute("message", String)
			Attribute("kind", String)
			if mode != "optional status field" {
				Required("status")
			}
		})
		hints := Type("Hints", func() {
			Attribute("link", String)
		})
		handshake := Type("Handshake", func() {
			Attribute("status", Int)
			Attribute("subscription", String)
			Attribute("resume", String)
			Attribute("kind", String)
			Required("status", "subscription")
		})
		exactHandshake := Type("CreatedHandshake", func() {
			Attribute("subscription", String)
			Attribute("resume", String)
			Required("subscription")
		})
		Service("status_selection", func() {
			if mode == "JSONRPC class" {
				JSONRPC(func() {
					POST("/rpc")
				})
			}
			Method("call", func() {
				switch {
				case strings.HasPrefix(mode, "SSE "):
					StreamingResult(String)
				case mode == "error only", mode == "file class":
				case mode == "JSONRPC class":
					Result(String)
				default:
					Result(carrier)
				}
				if mode == "built in class error" {
					Error("failure")
				} else {
					Error("failure", carrier)
				}
				if mode == "JSONRPC class" {
					JSONRPC(func() {
						Response("failure", Status5XX, func() {
							StatusField("status")
						})
					})
					return
				}
				HTTP(func() {
					GET("/call")
					switch mode {
					case "exact informational", "informational content":
						Response(StatusOK)
						Informational("hints", StatusEarlyHints, hints, func() {
							Header("link:Link")
							if mode == "informational content" {
								Body("link")
							}
						})
					case "class informational":
						Response(StatusOK)
						Informational("progress", Status1XX, carrier, func() {
							StatusField("status")
							Header("message:X-Progress")
							Header("kind:X-Kind")
						})
					case "exact and class success":
						Response(StatusOK, func() {
							StatusField("status")
						})
						Response(Status2XX, func() {
							StatusField("status")
						})
					case "fallback success":
						Response(StatusFallback, func() {
							StatusField("status")
						})
					case "error only", "error only with result":
						NoSuccessResponse()
						Response("failure", Status4XX, func() {
							StatusField("status")
						})
					case "mixed class key":
						Response(Status4XX, func() {
							StatusField("status")
						})
						Response("failure", Status4XX, func() {
							StatusField("status")
						})
					case "mixed exact key":
						Response(StatusNotFound)
						Response("failure", StatusNotFound)
					case "tag with class":
						Response(StatusOK, func() {
							StatusField("status")
						})
						Response(Status2XX, func() {
							StatusField("status")
							Tag("kind", "created")
						})
					case "informational as final":
						Response(StatusEarlyHints)
					case "upgrade as final":
						Response(101)
					case "file class":
						FileResponse()
						Response(Status2XX)
					case "SSE exact created":
						ServerSentEvents()
						Response(StatusCreated, exactHandshake, func() {
							Header("subscription:X-Subscription-ID")
							Header("resume:X-Resume-Token")
						})
					case "SSE two exact", "SSE class", "SSE tag":
						ServerSentEvents()
						if mode == "SSE two exact" || mode == "SSE tag" {
							Response(StatusOK, handshake, func() {
								StatusField("status")
								Header("subscription:X-Subscription-ID")
								Header("resume:X-Resume-Token")
								Header("kind:X-Kind")
							})
						}
						status := any(Status2XX)
						if mode != "SSE class" {
							status = StatusCreated
						}
						Response(status, handshake, func() {
							StatusField("status")
							Header("subscription:X-Subscription-ID")
							Header("resume:X-Resume-Token")
							Header("kind:X-Kind")
							if mode == "SSE tag" {
								Tag("kind", "created")
							}
						})
					case "built in class error":
						Response(StatusOK)
						Response("failure", Status4XX, func() {
							StatusField("status")
						})
					default:
						Response(StatusOK)
						Response("failure", Status4XX, func() {
							if mode != "missing status field" {
								StatusField("status")
							}
							if mode == "status header collision" {
								Header("status:X-Status")
							}
						})
					}
				})
			})
		})
	}
}
