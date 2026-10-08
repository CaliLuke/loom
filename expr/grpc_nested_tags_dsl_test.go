package expr_test

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/expr/testdata"
	"github.com/stretchr/testify/require"
)

func TestGRPCNestedTagMessageRoles(t *testing.T) {
	for _, role := range []string{"request", "response", "stream", "array", "map", "error", "metadata", "partial-message"} {
		t.Run(role, func(t *testing.T) {
			tagged := false
			design := func() {
				nested := Type("Nested", func() {
					if tagged {
						Field(1, "value", Any)
					} else {
						Attribute("value", Any)
					}
				})
				Service("svc", func() {
					Method("method", func() {
						switch role {
						case "response":
							Result(nested)
						case "stream":
							StreamingPayload(nested)
						case "array":
							Payload(ArrayOf(nested))
						case "map":
							Payload(MapOf(String, nested))
						case "error":
							Error("broken", nested)
						case "metadata", "partial-message":
							Payload(func() { Field(1, "item", nested); Attribute("header", String) })
						default:
							Payload(nested)
						}
						GRPC(func() {
							if role == "metadata" || role == "partial-message" {
								Metadata(func() { Attribute("header") })
							}
							if role == "partial-message" {
								Message(func() { Attribute("item") })
							}
							if role == "error" {
								Response("broken", CodeInternal)
							}
						})
					})
				})
			}
			require.ErrorContains(t, expr.RunInvalidDSL(t, design), `attribute "value" does not have "rpc:tag"`)
			tagged = true
			expr.RunDSL(t, design)
		})
	}
	expr.RunDSL(t, testdata.GRPCEndpointWithTaggedUnionContainingAny)
}
