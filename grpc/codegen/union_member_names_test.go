package codegen

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
)

func TestUnionMemberNamesGeneratedGRPC(t *testing.T) {
	runGeneratedRoundTrip(t, "example.com/grpcmembers", func() {
		envelope := Type("Envelope", func() {
			OneOf("pick", func() {
				Field(1, "Kind", String)
				Field(2, "Kind2", String)
				Field(3, "a-b", String)
				Field(4, "a_b", String)
				Field(5, "AB2", String)
			})
			Required("pick")
		})
		Service("members", func() {
			Method("echo", func() {
				Payload(envelope)
				Result(envelope)
				GRPC(func() {})
			})
		})
	}, unionMemberNamesGRPCHarness)
}

const unionMemberNamesGRPCHarness = `package roundtrip

import (
	"testing"
	"github.com/stretchr/testify/require"
	"%[1]s/gen/grpc/members/client"
	"%[1]s/gen/grpc/members/server"
	service "%[1]s/gen/members"
)

func TestBranches(t *testing.T) {
	for _, test := range []struct {
		tag string
		set func(*service.Envelope)
	}{
		{"Kind", func(v *service.Envelope) { v.Pick.SetKind("kind") }},
		{"Kind2", func(v *service.Envelope) { v.Pick.SetKind2("sibling") }},
		{"a-b", func(v *service.Envelope) { v.Pick.SetAB("first") }},
		{"a_b", func(v *service.Envelope) { v.Pick.SetAB3("second") }},
		{"AB2", func(v *service.Envelope) { v.Pick.SetAB2("reserved") }},
	} {
		v := &service.Envelope{}
		test.set(v)
		require.Equal(t, test.tag, string(v.Pick.Kind()))
		request := client.NewProtoEchoRequest(v)
		require.Equal(t, v, server.NewEchoPayload(request))
		response := server.NewProtoEchoResponse(v)
		require.Equal(t, v, client.NewEchoResult(response))
		require.NoError(t, server.ValidateEchoRequest(request))
		require.NoError(t, client.ValidateEchoResponse(response))
	}
}
`
