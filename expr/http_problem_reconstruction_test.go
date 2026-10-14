package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestProblemReconstructionUsesServiceRepresentation(t *testing.T) {
	for _, transport := range []string{"http", "jsonrpc"} {
		for _, shape := range []string{"default", "error result", "problem result", "custom"} {
			t.Run(transport+"/"+shape, func(t *testing.T) {
				design := func() {
					Service("coverage", func() {
						if transport == "jsonrpc" {
							JSONRPC(func() {
								POST("/coverage")
							})
						}
						Method("send", func() {
							switch shape {
							case "default":
								Error("oops")
							case "error result":
								Error("oops", ErrorResult)
							case "problem result":
								Error("oops", expr.ProblemResult)
							case "custom":
								Error("oops", func() {
									Attribute("detail", String)
									Attribute("timeout", Boolean)
									Required("detail", "timeout")
								})
							}
							mapping := func() {
								if transport == "http" {
									POST("/coverage")
								}
								Response("oops", StatusBadRequest, func() {
									Body(func() {
										Attribute("detail")
									})
								})
							}
							if transport == "http" {
								HTTP(mapping)
							} else {
								JSONRPC(mapping)
							}
						})
					})
				}
				if shape == "custom" {
					err := expr.RunInvalidDSL(t, design)
					require.ErrorContains(t, err, `required error oops attribute "timeout" has no reconstruction source`)
				} else {
					expr.RunDSL(t, design)
				}
			})
		}
	}
}
