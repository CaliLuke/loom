package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestHTTPRequestContentType(t *testing.T) {
	for _, test := range []struct {
		name      string
		media     string
		configure func()
		want      string
	}{
		{name: "JSON", media: "application/json"},
		{name: "vendor", media: "application/merge-patch+json"},
		{name: "parameters", media: "Application/Vnd.example+JSON; charset=utf-8"},
		{name: "empty", media: "", want: "concrete JSON media type"},
		{name: "malformed", media: "application/json;bad", want: "concrete JSON media type"},
		{name: "wildcard", media: "application/*+json", want: "concrete JSON media type"},
		{name: "incompatible", media: "application/xml", want: "concrete JSON media type"},
		{name: "unknown", media: "application/unknown", want: "concrete JSON media type"},
		{name: "form", media: "application/json", configure: FormRequest, want: "typed JSON request body"},
		{name: "multipart", media: "application/json", configure: MultipartRequest, want: "typed JSON request body"},
		{name: "raw", media: "application/json", configure: SkipRequestBodyEncodeDecode, want: "typed JSON request body"},
		{name: "mapped", media: "application/json", configure: func() {
			Header("value:cOnTeNt-TyPe")
		}, want: "mapped Content-Type"},
		{name: "bodyless", media: "application/json", configure: func() {
			Param("value")
		}, want: "typed JSON request body"},
	} {
		t.Run(test.name, func(t *testing.T) {
			design := func() {
				Service("media", func() {
					Method("send", func() {
						Payload(func() {
							Attribute("value", String)
						})
						HTTP(func() {
							POST("/send")
							ContentType(test.media)
							if test.configure != nil {
								test.configure()
							}
						})
					})
				})
			}
			if test.want != "" {
				err := expr.RunInvalidDSL(t, design)
				require.Contains(t, err.Error(), test.want)
				return
			}
			expr.RunDSL(t, design)
		})
	}
}

func TestHTTPRequestContentTypeTransportScope(t *testing.T) {
	for _, mode := range []string{"inherited header", "JSON-RPC", "WebSocket", "SSE"} {
		t.Run(mode, func(t *testing.T) {
			design := func() {
				Service("scopes", func() {
					if mode == "inherited header" {
						HTTP(func() {
							Header("value:Content-Type")
						})
					}
					if mode == "JSON-RPC" {
						JSONRPC(func() {
							POST("/rpc")
						})
					}
					Method("send", func() {
						Payload(func() {
							Attribute("value", String)
							Attribute("body", String)
						})
						if mode == "WebSocket" || mode == "SSE" {
							StreamingResult(String)
						}
						transport := func() {
							ContentType("application/merge-patch+json")
							if mode != "JSON-RPC" {
								POST("/send")
							}
							if mode == "SSE" {
								ServerSentEvents()
							}
						}
						if mode == "JSON-RPC" {
							JSONRPC(transport)
						} else {
							HTTP(transport)
						}
					})
				})
			}
			if mode == "SSE" {
				expr.RunDSL(t, design)
				return
			}
			err := expr.RunInvalidDSL(t, design)
			if mode == "inherited header" {
				require.Contains(t, err.Error(), "mapped Content-Type")
			} else {
				require.Contains(t, err.Error(), "typed JSON request body")
			}
		})
	}
}
