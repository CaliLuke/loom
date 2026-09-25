package codegen

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
)

// TestGeneratedUnionVariantTagRoundTrip compiles a generated module whose
// message has a union with a branch that sets its variant tag with
// oneof:type:tag, and round-trips both branches through the generated
// protobuf conversions in both directions. The conversions to protocol
// buffers switch on the union kind, which is the variant tag.
func TestGeneratedUnionVariantTagRoundTrip(t *testing.T) {
	runGeneratedRoundTrip(t, "example.com/grpcuniontag", unionVariantTagDSL, unionVariantTagHarness)
}

func unionVariantTagDSL() {
	var Envelope = Type("Envelope", func() {
		OneOf("pick", func() {
			Field(1, "alpha", String, func() {
				Meta("oneof:type:tag", "A")
			})
			Field(2, "beta", Int)
		})
		Required("pick")
	})
	Service("uniontag", func() {
		Method("echo", func() {
			Payload(Envelope)
			Result(Envelope)
			GRPC(func() {})
		})
	})
}

const unionVariantTagHarness = `package roundtrip

import (
	"testing"

	"github.com/stretchr/testify/require"

	"%[1]s/gen/grpc/uniontag/client"
	"%[1]s/gen/grpc/uniontag/server"
	uniontag "%[1]s/gen/uniontag"
)

func TestRoundTrip(t *testing.T) {
	alpha := &uniontag.Envelope{}
	alpha.Pick.SetAlpha("a")
	require.Equal(t, uniontag.PickKindAlpha, alpha.Pick.Kind())
	require.Equal(t, "A", string(alpha.Pick.Kind()))
	beta := &uniontag.Envelope{}
	beta.Pick.SetBeta(2)
	for _, envelope := range []*uniontag.Envelope{alpha, beta} {
		request := client.NewProtoEchoRequest(envelope)
		require.NotNil(t, request.Pick, "request oneof of %%s", envelope.Pick.Kind())
		require.Equal(t, envelope, server.NewEchoPayload(request))
		response := server.NewProtoEchoResponse(envelope)
		require.NotNil(t, response.Pick, "response oneof of %%s", envelope.Pick.Kind())
		require.Equal(t, envelope, client.NewEchoResult(response))
		require.NoError(t, server.ValidateEchoRequest(request))
		require.NoError(t, client.ValidateEchoResponse(response))
	}
	require.Equal(t, "a", client.NewProtoEchoRequest(alpha).GetAlpha())
	require.Equal(t, int64(2), server.NewProtoEchoResponse(beta).GetBeta())
}
`
