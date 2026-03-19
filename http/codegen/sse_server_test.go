package codegen

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/codegen/testutil"
	"goa.design/goa/v3/http/codegen/testdata"
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
			sections := fs[1].SectionTemplates
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
	require.Contains(t, code, "s.initHeaders()")
	require.NotContains(t, code, "func (s *SSEObjectMethodServerStream) open() error")
	require.Contains(t, code, "if err := goahttp.WriteSSEEvent(s.w, msg); err != nil {")
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
	require.Contains(t, code, "if err := encodeError(ctx, w, err); err != nil && errhandler != nil {")
	require.NotContains(t, code, "if errhandler != nil {\n\t\t\t\t\terrhandler(ctx, w, err)\n\t\t\t\t}")
}
