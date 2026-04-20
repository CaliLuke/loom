package codegen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestRenderSSEParseAssignment(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		code := renderSSEParseAssignment("event", "string")
		require.Contains(t, code, "\tevent = dataContent\n")
	})

	t.Run("bytes", func(t *testing.T) {
		code := renderSSEParseAssignment("event", "[]byte")
		require.Contains(t, code, "\tevent = []byte(dataContent)\n")
	})

	t.Run("int", func(t *testing.T) {
		code := renderSSEParseAssignment("event", "int")
		require.Contains(t, code, "strconv.Atoi(dataContent)")
		require.Contains(t, code, "\tevent = v\n")
	})

	t.Run("default decode", func(t *testing.T) {
		code := renderSSEParseAssignment("event", "*Example")
		require.Contains(t, code, "strings.NewReader(dataContent)")
		require.Contains(t, code, "s.decoder(respBody).Decode(&event)")
	})
}

func TestSSEClientEmitterSectionStillRendersCoreHelpers(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		testdata.SSEObjectDSL()
	})
	services := CreateHTTPServices(root)
	files := ClientFiles("gen", services)
	sseFile := findFileWithSection(t, files, "client-sse")
	code := renderedSectionSource(t, sseFile.Section("client-sse")[0])

	require.Contains(t, code, "func (s *SSEObjectMethodStreamImpl) readEvent(ctx context.Context) ([]byte, error) {")
	require.Contains(t, code, "func (s *SSEObjectMethodStreamImpl) checkBuffer() ([]byte, bool) {")
	require.Contains(t, code, "func (s *SSEObjectMethodStreamImpl) processEvent(eventData []byte)")
	require.True(t, strings.Index(code, "func (s *SSEObjectMethodStreamImpl) readEvent") < strings.Index(code, "func (s *SSEObjectMethodStreamImpl) processEvent"))
}
