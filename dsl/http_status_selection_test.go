package dsl_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestStatusSelectionDSL(t *testing.T) {
	for _, name := range []string{"final and informational", "error only", "stream handshake", "duplicate informational name"} {
		t.Run(name, func(t *testing.T) {
			design := func() {
				failure := Type("Failure", func() {
					ErrorName("name", String)
					Attribute("HTTPStatus", Int)
					Attribute("message", String)
					Required("HTTPStatus", "name")
				})
				hints := Type("EarlyHints", func() {
					Attribute("links", ArrayOf(String))
				})
				progress := Type("Progress", func() {
					Attribute("Code", Int)
					Required("Code")
				})
				handshake := Type("StreamResponse", func() {
					Attribute("HTTPStatus", Int)
					Attribute("trace", String)
					Required("HTTPStatus")
				})
				Service("selection", func() {
					Method("call", func() {
						Error("client", failure)
						Error("unexpected", failure)
						if name == "stream handshake" {
							StreamingResult(String)
						} else if name != "error only" {
							Result(String)
						}
						HTTP(func() {
							GET("/call")
							Response("client", Status4XX, func() {
								StatusField("HTTPStatus")
							})
							if name == "error only" {
								NoSuccessResponse()
								return
							}
							Response("unexpected", StatusFallback, func() {
								StatusField("HTTPStatus")
							})
							if name == "stream handshake" {
								ServerSentEvents()
								Response(Status2XX, handshake, func() {
									StatusField("HTTPStatus")
									Header("trace:X-Trace")
								})
								return
							}
							Response(StatusOK)
							Informational("hints", StatusEarlyHints, hints, func() {
								Description("Links available before the final response")
								Header("links:Link")
							})
							Informational("progress", Status1XX, progress, func() {
								StatusField("Code")
							})
							if name == "duplicate informational name" {
								Informational("hints", StatusEarlyHints, hints, func() {
									Header("links:Link")
								})
							}
						})
					})
				})
			}
			if name == "duplicate informational name" {
				err := expr.RunInvalidDSL(t, design)
				require.Contains(t, strings.ToLower(err.Error()), "informational")
				require.ErrorContains(t, err, "hints")
			} else {
				expr.RunDSL(t, design)
			}
		})
	}
}
