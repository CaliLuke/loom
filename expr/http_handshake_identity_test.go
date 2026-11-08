package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func TestHandshakeNativeCarrierIdentity(t *testing.T) {
	for _, mode := range []string{"shared", "copied", "distinct", "alias"} {
		for _, reverse := range []bool{false, true} {
			name := mode
			if reverse {
				name += "_reverse"
			}
			t.Run(name, func(t *testing.T) {
				root := expr.SetupTestDSL(t)
				require.True(t, eval.Execute(func() {
					fields := func() {
						Attribute("status", Int)
						Attribute("session", String)
						Required("status")
					}
					first := Type("Opening", fields)
					second := first
					if mode == "distinct" {
						second = Type("OtherOpening", fields)
					} else if mode == "alias" {
						second = Type("OpeningAlias", first)
					}
					Service("events", func() {
						Method("watch", func() {
							StreamingResult(String)
							HTTP(func() {
								GET("/events")
								ServerSentEvents()
								if mode == "copied" {
									second = expr.Dup(first).(expr.UserType)
								}
								values := []expr.UserType{first, second}
								if reverse {
									values[0], values[1] = values[1], values[0]
								}
								for i, value := range values {
									Response(StatusOK+i, value, func() {
										StatusField("status")
										Header("session:X-Session")
									})
								}
							})
						})
					})
				}, nil), eval.Context.Error())
				err := eval.RunDSL()
				if mode == "distinct" || mode == "alias" {
					require.ErrorContains(t, err, "SSE success responses must use the same native handshake type")
					return
				}
				require.NoError(t, err)
				responses := root.API.HTTP.Services[0].HTTPEndpoints[0].Responses
				require.NotSame(t, responses[0].Handshake, responses[1].Handshake)
				require.Equal(t, responses[0].Handshake.Type.(expr.UserType).ID(), responses[1].Handshake.Type.(expr.UserType).ID())
			})
		}
	}
}

func TestHandshakeAnonymousCarrierIdentity(t *testing.T) {
	for _, mode := range []string{"singleton", "shared", "distinct"} {
		t.Run(mode, func(t *testing.T) {
			root := expr.SetupTestDSL(t)
			first := &expr.Object{{Name: "status", Attribute: &expr.AttributeExpr{Type: expr.Int}}}
			second := first
			if mode == "distinct" {
				second = expr.Dup(first).(*expr.Object)
			}
			require.True(t, eval.Execute(func() {
				Service("events", func() {
					Method("watch", func() {
						StreamingResult(String)
						HTTP(func() {
							GET("/events")
							ServerSentEvents()
							if mode == "singleton" {
								Response(StatusCreated, first)
								return
							}
							Response(StatusOK, first, func() {
								StatusField("status")
							})
							Response(StatusCreated, second, func() {
								StatusField("status")
							})
						})
					})
				})
			}, nil), eval.Context.Error())
			err := eval.RunDSL()
			responses := root.API.HTTP.Services[0].HTTPEndpoints[0].Responses
			if mode == "singleton" {
				require.NoError(t, err)
				require.Equal(t, first, expr.AsObject(responses[0].Handshake.Type))
				return
			}
			// Raw DataType arguments carry no root Required declaration. These
			// rows test identity through Prepare/Validate, not admitted multi-status.
			require.ErrorContains(t, err, "required non-nullable native Int")
			require.Same(t, first, responses[0].Handshake.Type)
			require.Same(t, second, responses[1].Handshake.Type)
			if mode == "distinct" {
				require.ErrorContains(t, err, "SSE success responses must use the same native handshake type")
			} else {
				require.NotContains(t, err.Error(), "same native handshake type")
			}
		})
	}
}
