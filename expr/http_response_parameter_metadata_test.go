package expr_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestResponseBodyRejectsParameterMetadata(t *testing.T) {
	for _, kind := range []string{"success", "error"} {
		for _, header := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/header=%t", kind, header), func(t *testing.T) {
				design := func() {
					record := Type("Record", func() {
						Attribute("value", String, func() {
							Meta("openapi:style", "simple")
						})
					})
					Service("metadata", func() {
						Method("send", func() {
							if kind == "error" {
								Error("failure", record)
							} else {
								Result(record)
							}
							HTTP(func() {
								POST("/")
								mapping := func() {
									if header {
										Header("value")
									}
								}
								if kind == "error" {
									Response("failure", StatusBadRequest, mapping)
								} else {
									Response(StatusOK, mapping)
								}
							})
						})
					})
				}
				if header {
					expr.RunDSL(t, design)
				} else {
					require.ErrorContains(t, expr.RunInvalidDSL(t, design), "cannot declare parameter metadata")
				}
			})
		}
	}
}
