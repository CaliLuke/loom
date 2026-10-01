package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
)

func TestLowerInitial(t *testing.T) {
	cases := []struct {
		Name string
		In   string
		Want string
	}{
		{Name: "empty", In: "", Want: ""},
		{Name: "ASCII", In: "Widget", Want: "widget"},
		{Name: "acronym", In: "URLValue", Want: "uRLValue"},
		{Name: "BMP uppercase", In: "Éclair", Want: "éclair"},
		{Name: "caseless", In: "日本", Want: "日本"},
		{Name: "astral without lowercase", In: "𝒜streamer", Want: "𝒜streamer"},
		{Name: "astral with lowercase", In: "𐐀streamer", Want: "𐐨streamer"},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			require.Equal(t, c.Want, lowerInitial(c.In))
		})
	}
}

func TestUnicodeWebSocketPrivateIdentifiers(t *testing.T) {
	root := RunJSONRPCDSL(t, unicodeWebSocketIdentifiersDSL)
	data := CreateJSONRPCServices(root).Get("𝒜streamer")
	require.NotNil(t, data)

	streamCode := codegen.SectionCode(t, jsonrpcWebSocketServerStructSection(data))
	require.Contains(t, streamCode, "type 𝒜streamerStream struct")
	require.Contains(t, streamCode, "𐐨xchange func(")
	require.Contains(t, streamCode, "𐐨xchangeEndpoint loom.Endpoint")

	serverCode := codegen.SectionCode(t, jsonrpcWebSocketServerHandlerSection(data))
	require.Contains(t, serverCode, "stream := &𝒜streamerStream{")
	require.Contains(t, serverCode, "𐐨xchange:         s.𐐨xchange,")
	require.Contains(t, serverCode, "𐐨xchangeEndpoint: s.𐐨xchangeEndpoint,")
}

func unicodeWebSocketIdentifiersDSL() {
	var Frame = dsl.Type("Frame", func() {
		dsl.Attribute("text", dsl.String)
	})
	dsl.API("unicode-identifiers", func() {
		dsl.JSONRPC(func() {})
	})
	dsl.Service("𝒜streamer", func() {
		dsl.JSONRPC(func() {
			dsl.GET("/rpc")
		})
		dsl.Method("𐐀xchange", func() {
			dsl.StreamingPayload(Frame)
			dsl.StreamingResult(Frame)
			dsl.JSONRPC(func() {})
		})
	})
}
