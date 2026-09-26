package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestRenderSSEParseAssignment(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		code := renderSSEParseAssignment("event", "string", false)
		require.Contains(t, code, "\tevent = dataContent\n")
	})

	t.Run("bytes", func(t *testing.T) {
		code := renderSSEParseAssignment("event", "[]byte", false)
		require.Contains(t, code, "\tevent = []byte(dataContent)\n")
	})

	t.Run("int", func(t *testing.T) {
		code := renderSSEParseAssignment("event", "int", false)
		require.Contains(t, code, "strconv.Atoi(dataContent)")
		require.Contains(t, code, "\tevent = v\n")
	})

	t.Run("default decode", func(t *testing.T) {
		code := renderSSEParseAssignment("event", "*Example", false)
		require.Contains(t, code, "strings.NewReader(dataContent)")
		require.Contains(t, code, "s.decoder(respBody).Decode(&event)")
	})

	t.Run("optional string", func(t *testing.T) {
		code := renderSSEParseAssignment("event", "string", true)
		require.Equal(t, "\tif dataContent != \"\" {\n\t\tevent = &dataContent\n\t}\n", code)
	})

	t.Run("optional int decode", func(t *testing.T) {
		code := renderSSEParseAssignment("event", "int", true)
		require.NotContains(t, code, "strconv.Atoi")
		require.Contains(t, code, "s.decoder(respBody).Decode(&event)")
	})
}

func TestSSEParseAssignmentNeedsDecoder(t *testing.T) {
	cases := []struct {
		TypeRef string
		Pointer bool
		Want    bool
	}{
		{"string", false, false},
		{"string", true, false},
		{"[]byte", false, false},
		{"int", false, false},
		{"int", true, true},
		{"bool", true, true},
		{"float64", false, true},
	}
	for _, c := range cases {
		require.Equal(t, c.Want, sseParseAssignmentNeedsDecoder(c.TypeRef, c.Pointer), "%s pointer=%t", c.TypeRef, c.Pointer)
	}
}

func TestSSEClientEmitterDelegatesCoreReader(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		testdata.SSEObjectDSL()
	})
	services := CreateHTTPServices(root)
	files := ClientFiles("gen", services)
	sseFile := findFileWithSection(t, files, "client-sse")
	code := renderedSectionSource(t, sseFile.Section("client-sse")[0])

	require.Contains(t, code, "SSEStreamReader")
	require.Contains(t, code, "reader:  loomhttp.NewSSEStreamReader(resp.Body)")
	require.Contains(t, code, "byts, err = s.reader.ReadEvent(ctx)")
	require.Contains(t, code, "return s.reader.Close()")
	require.NotContains(t, code, "func (s *SSEObjectMethodStreamImpl) readEvent")
	require.NotContains(t, code, "func (s *SSEObjectMethodStreamImpl) checkBuffer")
	require.Contains(t, code, "func (s *SSEObjectMethodStreamImpl) processEvent(eventData []byte)")
}

func TestSSEClientEmitterDoesNotEmitBlockingReadLoop(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		testdata.SSEObjectDSL()
	})
	services := CreateHTTPServices(root)
	files := ClientFiles("gen", services)
	sseFile := findFileWithSection(t, files, "client-sse")
	code := renderedSectionSource(t, sseFile.Section("client-sse")[0])

	require.NotContains(t, code, "readLock sync.Mutex")
	require.NotContains(t, code, "s.readLock.Lock()")
	require.NotContains(t, code, "body := s.resp.Body")
	require.NotContains(t, code, "readc := make(chan readResult")
	require.NotContains(t, code, "n, err := s.resp.Body.Read(buf)")
}

func TestSSEClientEmitterReturnsEOFFromRecv(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		testdata.SSEObjectDSL()
	})
	services := CreateHTTPServices(root)
	files := ClientFiles("gen", services)
	sseFile := findFileWithSection(t, files, "client-sse")
	code := renderedSectionSource(t, sseFile.Section("client-sse")[0])

	require.Contains(t, code, "if errors.Is(err, io.EOF) {\n\t\t\tif closeErr := s.Close(); closeErr != nil {\n\t\t\t\treturn event, errors.Join(io.EOF, closeErr)\n\t\t\t}\n\t\t\treturn event, io.EOF\n\t\t}")
	require.NotContains(t, code, "err = nil")
}
