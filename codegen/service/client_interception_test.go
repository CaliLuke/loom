package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

// TestClientInterceptorDispatch checks endpoint-local applicability and the
// shared declaration identities consumed by construction and wrapper rendering.
func TestClientInterceptorDispatch(t *testing.T) {
	for _, profile := range []string{"stream", "combined", "payload", "generic"} {
		t.Run(profile, func(t *testing.T) {
			root := runDSL(t, func() {
				audit := dsl.Interceptor("audit", func() {
					if profile == "stream" || profile == "combined" {
						dsl.ReadStreamingResult(func() {
							dsl.Attribute("value")
						})
					}
					if profile == "payload" || profile == "combined" {
						dsl.ReadPayload(func() {
							dsl.Attribute("id")
						})
					}
				})
				summary := dsl.Type("Summary", func() {
					dsl.Attribute("value", dsl.String)
				})
				chunk := dsl.Type("Chunk", func() {
					dsl.Attribute("value", dsl.String)
				})
				dsl.Service("events", func() {
					dsl.Method("watch", func() {
						dsl.Payload(func() {
							dsl.Attribute("id", dsl.String)
						})
						dsl.Result(summary)
						dsl.StreamingResult(chunk)
						dsl.ClientInterceptor(audit)
					})
				})
			})
			data := NewServicesData(root).Get("events")
			method := endpointData(data).Methods[0]
			require.True(t, method.HasMixedResults)
			require.Len(t, data.ClientInterceptors[0].Methods, 1, "endpoint plans must not duplicate accessor records")
			plan := method.clientInterception
			require.Len(t, plan.stream.bindings, 1)
			require.Equal(t, "Audit", plan.stream.bindings[0].name)
			require.Equal(t, profile == "stream" || profile == "combined", plan.stream.bindings[0].messages)
			if profile == "stream" {
				require.Empty(t, plan.primary.bindings)
			} else {
				require.Len(t, plan.primary.bindings, 1)
				require.True(t, plan.primary.bindings[0].opening)
				require.False(t, plan.primary.bindings[0].messages)
			}
			if profile == "payload" || profile == "generic" {
				require.Same(t, plan.primary, plan.stream, "identical chains have one declaration owner")
			} else {
				require.NotSame(t, plan.primary, plan.stream)
			}

			// Equalize only the endpoint argument to compare selected behavior.
			method.ArgName = "endpoint"
			method.StreamArgName = "endpoint"
			ordinary := renderJenniferExpr(t, wrappedClientEndpointExpr(method, false))
			stream := renderJenniferExpr(t, wrappedClientEndpointExpr(method, true))
			if profile == "stream" || profile == "combined" {
				require.NotEqual(t, ordinary, stream, "ordinary and streaming boundaries require different interception")
			} else {
				require.Equal(t, ordinary, stream, "a profile without message access retains the same opening behavior")
			}
			if profile == "stream" {
				require.Equal(t, "var _ = endpoint", strings.TrimSpace(ordinary))
			}
		})
	}
}
