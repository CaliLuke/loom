package expr_test

import (
	"fmt"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

// TestHTTPWireNameAdmission rejects invalid effective names across HTTP consumers.
func TestHTTPWireNameAdmission(t *testing.T) {
	for _, rpc := range []bool{false, true} {
		transport := "http"
		if rpc {
			transport = "jsonrpc"
		}
		for _, location := range []string{"request header", "request cookie", "response header", "response cookie", "error header", "error cookie"} {
			for _, name := range []string{"X Space", "X:Colon", "semi;colon", "café", "line\r\n"} {
				t.Run(fmt.Sprintf("%s/%s/%q", transport, location, name), func(t *testing.T) {
					err := expr.RunInvalidDSL(t, func() {
						Service("names", func() {
							if rpc {
								JSONRPC(func() {
									POST("/names")
								})
							}
							Method("show", func() {
								Payload(func() {
									Attribute("value", String)
								})
								Result(func() {
									Attribute("value", String)
								})
								Error("failed", func() {
									Attribute("value", String)
								})
								mapping := func() {
									switch location {
									case "request header", "response header", "error header":
										Header("value:" + name)
									default:
										Cookie("value:" + name)
									}
								}
								transportDSL := func() {
									if !rpc {
										POST("/names")
									}
									switch location {
									case "request header", "request cookie":
										mapping()
									case "response header", "response cookie":
										Response(StatusOK, mapping)
									default:
										Response("failed", StatusBadRequest, mapping)
									}
								}
								if rpc {
									JSONRPC(transportDSL)
								} else {
									HTTP(transportDSL)
								}
							})
						})
					})
					require.Error(t, err)
					require.Contains(t, err.Error(), "token")
				})
			}
		}
	}
}
