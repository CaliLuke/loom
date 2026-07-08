package codegen

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestSSE(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"string", testdata.SSEStringDSL},
		{"int", testdata.SSEIntDSL},
		{"bool", testdata.SSEBoolDSL},
		{"object", testdata.SSEObjectDSL},
		{"data-field", testdata.SSEDataFieldDSL},
		{"data-id-field", testdata.SSEDataIDFieldDSL},
		{"request-id", testdata.SSERequestIDDSL},
		{"all-fields", testdata.SSEAllFieldsDSL},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.DSL)
			services := CreateHTTPServices(root)
			fs := ServerFiles("", services)
			require.Len(t, fs, 3)
			sections := fs[1].AllSections()
			require.Greater(t, len(sections), 1)
			code := codegen.SectionCode(t, sections[1])
			golden := filepath.Join("testdata", "golden", "sse-"+c.Name+".golden")
			testutil.CompareOrUpdateGolden(t, code, golden)
		})
	}
}

func TestSSEServerStreamCommitsHeadersOnSend(t *testing.T) {
	root := RunHTTPDSL(t, testdata.SSEObjectDSL)
	services := CreateHTTPServices(root)
	files := ServerFiles("", services)

	var sseFile *codegen.File
	for _, f := range files {
		if strings.HasSuffix(f.Path, filepath.Join("server", "sse.go")) {
			sseFile = f
			break
		}
	}
	require.NotNil(t, sseFile)

	code := codegen.SectionCode(t, sseFile.Section("server-sse")[0])
	require.Contains(t, code, "// Send streams instances of")
	require.Contains(t, code, "// SendWithContext streams instances of")
	require.NotContains(t, code, "// Send Send streams")
	require.NotContains(t, code, "// SendWithContext SendWithContext streams")
	require.Contains(t, code, "s.initHeaders()")
	require.NotContains(t, code, "func (s *SSEObjectMethodServerStream) open() error")
	require.Contains(t, code, "return s.SendWithContext(s.r.Context(), v)")
	require.NotContains(t, code, "return s.SendWithContext(context.Background(), v)")
	require.Contains(t, code, "data, err := loomhttp.EncodeSSEData(payload)")
	require.NotContains(t, code, "switch v := payload.(type)")
	require.Contains(t, code, "if err := loomhttp.WriteSSEEvent(s.w, msg); err != nil {")
	require.Contains(t, code, "if err := ctx.Err(); err != nil {")
	require.Contains(t, code, "if err := s.r.Context().Err(); err != nil {")
}

func TestSSEHandlerDefersStreamCommitUntilEndpointAccepts(t *testing.T) {
	root := RunHTTPDSL(t, testdata.SSEObjectDSL)
	services := CreateHTTPServices(root)
	files := ServerFiles("", services)

	var serverFile *codegen.File
	for _, f := range files {
		if strings.HasSuffix(f.Path, filepath.Join("server", "server.go")) {
			serverFile = f
			break
		}
	}
	require.NotNil(t, serverFile)

	code := codegen.SectionCode(t, serverFile.Section("server-handler-init")[0])
	require.Contains(t, code, "stream := &SSEObjectMethodServerStream{")
	require.Contains(t, code, "Stream: stream,")
	require.NotContains(t, code, "stream.open()")
	decl := strings.Index(code, "encodeError =")
	require.NotEqual(t, -1, decl, "raw SSE handlers must declare encodeError before using it")
	use := strings.Index(code, "if err := encodeError(ctx, w, err); err != nil && errhandler != nil {")
	require.NotEqual(t, -1, use, "raw SSE handlers must encode pre-stream endpoint failures")
	require.Less(t, decl, use, "encodeError must be declared before the endpoint failure path uses it")
	require.Contains(t, code, "if err := encodeError(ctx, w, err); err != nil && errhandler != nil {")
}

func TestSSEHandlerDoesNotEncodeErrorAfterStreamStarts(t *testing.T) {
	root := RunHTTPDSL(t, testdata.SSEObjectDSL)
	services := CreateHTTPServices(root)
	files := ServerFiles("", services)

	var serverFile *codegen.File
	for _, f := range files {
		if strings.HasSuffix(f.Path, filepath.Join("server", "server.go")) {
			serverFile = f
			break
		}
	}
	require.NotNil(t, serverFile)

	code := codegen.SectionCode(t, serverFile.Section("server-handler-init")[0])
	require.Contains(t, code, "if stream.started() {")
	require.Contains(t, code, "errhandler(ctx, w, err)")

	started := strings.Index(code, "if stream.started() {")
	encode := strings.Index(code[started:], "if err := encodeError(ctx, w, err); err != nil && errhandler != nil {")
	require.NotEqual(t, -1, started)
	require.NotEqual(t, -1, encode)
}

func TestSSEHandlerUsesTypedLastEventIDContextKey(t *testing.T) {
	root := RunHTTPDSL(t, testdata.SSERequestIDDSL)
	services := CreateHTTPServices(root)
	files := ServerFiles("", services)

	var serverFile *codegen.File
	for _, f := range files {
		if strings.HasSuffix(f.Path, filepath.Join("server", "server.go")) {
			serverFile = f
			break
		}
	}
	require.NotNil(t, serverFile)

	code := codegen.SectionCode(t, serverFile.Section("server-handler-init")[0])
	require.Contains(t, code, "context.WithValue(ctx, loomhttp.LastEventIDKey, lastEventID)")
	require.NotContains(t, code, `context.WithValue(ctx, "last-event-id", lastEventID)`)
}
