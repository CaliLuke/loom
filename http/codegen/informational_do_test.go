package codegen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
)

func TestInformationalClientDoBoundary(t *testing.T) {
	for _, streaming := range []bool{false, true} {
		for _, observed := range []bool{false, true} {
			name := "unary"
			if streaming {
				name = "stream"
			}
			if observed {
				name += "/observed"
			} else {
				name += "/ordinary"
			}
			t.Run(name, func(t *testing.T) {
				root := RunHTTPDSL(t, func() {
					hints := Type("Hints", func() {
						Attribute("trace", String)
					})
					Service("boundary", func() {
						Method("read", func() {
							if streaming {
								StreamingResult(String)
							} else {
								Result(String)
							}
							HTTP(func() {
								GET("/read")
								if streaming {
									ServerSentEvents()
								}
								if observed {
									Informational("hints", StatusEarlyHints, hints, func() {
										Header("trace:X-Trace")
									})
								}
								Response(StatusOK)
							})
						})
					})
				})
				endpoint := CreateHTTPServices(root).Get("boundary").Endpoint("read")
				actual := codegen.SectionCode(t, clientEndpointSection(endpoint))
				require.Equal(t, 1, strings.Count(actual, ".Do(req)"))
				if !observed {
					require.NotContains(t, actual, "ObserveInformational")
					require.NotContains(t, actual, "finishInformational")
					return
				}
				require.NotEmpty(t, endpoint.InformationalObservers)
				prepare := "req, finishInformational := loomhttp.ObserveInformational(req, " + endpoint.InformationalObservers + "(ctx))"
				finish := "resp, err = finishInformational(resp, err)"
				require.Contains(t, actual, prepare)
				require.Contains(t, actual, finish)
				require.Equal(t, 1, strings.Count(actual, "ObserveInformational("))
				require.Equal(t, 1, strings.Count(actual, "finishInformational(resp, err)"))
				require.Less(t, strings.Index(actual, prepare), strings.Index(actual, ".Do(req)"))
				require.Less(t, strings.Index(actual, ".Do(req)"), strings.Index(actual, finish))
				require.Less(t, strings.Index(actual, finish), strings.Index(actual, "loomhttp.ErrRequestError("))
				require.Less(t, strings.Index(actual, finish), strings.Index(actual, "decodeResponse(resp)"))
			})
		}
	}
}
