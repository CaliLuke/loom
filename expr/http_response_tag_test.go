package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestHTTPResponseTagAdmission rejects missing and non-string semantic fields.
func TestHTTPResponseTagAdmission(t *testing.T) {
	for _, rpc := range []bool{false, true} {
		for _, kind := range []string{"missing", "integer", "boolean"} {
			t.Run(kind+map[bool]string{false: "/http", true: "/jsonrpc"}[rpc], func(t *testing.T) {
				err := expr.RunInvalidDSL(t, func() {
					Service("tagged", func() {
						if rpc {
							JSONRPC(func() {
								POST("/tagged")
							})
						}
						Method("show", func() {
							Result(func() {
								carrier := String
								if kind == "integer" {
									carrier = Int
								} else if kind == "boolean" {
									carrier = Boolean
								}
								Attribute("kind", carrier)
							})
							mapping := func() {
								if !rpc {
									GET("/tagged")
								}
								Response(StatusOK)
								Response(StatusAccepted, func() {
									field := "kind"
									if kind == "missing" {
										field = "absent"
									}
									Tag(field, "accepted")
								})
							}
							if rpc {
								JSONRPC(mapping)
							} else {
								HTTP(mapping)
							}
						})
					})
				})
				require.ErrorContains(t, err, "Tag")
				require.ErrorContains(t, err, "HTTP response")
				require.ErrorContains(t, err, "string result field")
			})
		}
	}
}
