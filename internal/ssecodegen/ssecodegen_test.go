package ssecodegen

import (
	"bytes"
	"testing"

	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
)

func TestInitHeadersSourceVariants(t *testing.T) {
	t.Run("preserve existing headers", func(t *testing.T) {
		code := InitHeadersSource("s.w", HeaderOptions{PreserveExisting: true})
		require.Contains(t, code, `if header.Get("Content-Type") == "" {`)
		require.Contains(t, code, `header.Set("Content-Type", "text/event-stream")`)
		require.NotContains(t, code, `header.Set("X-Accel-Buffering", "no")`)
	})

	t.Run("force set headers with accel buffering", func(t *testing.T) {
		code := InitHeadersSource("s.w", HeaderOptions{IncludeAccelBuffering: true})
		require.NotContains(t, code, `if header.Get("Content-Type") == "" {`)
		require.Contains(t, code, `header.Set("X-Accel-Buffering", "no")`)
		require.Contains(t, code, `s.w.WriteHeader(http.StatusOK)`)
	})
}

func TestWriteAndFlushSource(t *testing.T) {
	code := WriteAndFlushSource("loomhttp.WriteSSEEvent(s.w, msg)", "s.w")
	require.Contains(t, code, `if err := loomhttp.WriteSSEEvent(s.w, msg); err != nil {`)
	require.Contains(t, code, `return http.NewResponseController(s.w).Flush()`)
}

func TestInitHeadersBodyAndWriteAndFlushBody(t *testing.T) {
	stmt := jen.Func().Id("render").Params().Error().Block(
		append(
			InitHeadersBody("s.w", HeaderOptions{IncludeAccelBuffering: true}),
			WriteAndFlushBody(
				jen.Id("loomhttp").Dot("WriteJSONSSEEvent").Call(
					jen.Id("s").Dot("w"),
					jen.Id("loomhttp").Dot("SSEMessage").Values(jen.Dict{jen.Id("Type"): jen.Lit("message")}),
					jen.Id("v"),
				),
				"s.w",
			)...,
		)...,
	)

	var buf bytes.Buffer
	require.NoError(t, stmt.Render(&buf))
	code := codegen.FormatTestCode(t, "package foo\n"+buf.String())
	require.Contains(t, code, `header.Set("Content-Type", "text/event-stream")`)
	require.Contains(t, code, `header.Set("X-Accel-Buffering", "no")`)
	require.Contains(t, code, `if err := loomhttp.WriteJSONSSEEvent(s.w, loomhttp.SSEMessage{Type: "message"}, v); err != nil {`)
	require.Contains(t, code, `return http.NewResponseController(s.w).Flush()`)
}
