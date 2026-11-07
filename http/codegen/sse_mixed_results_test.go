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

func TestSSE_MixedResults(t *testing.T) {
	root := RunHTTPDSL(t, testdata.MixedResultsDSL)
	services := CreateHTTPServices(root)

	t.Run("server", func(t *testing.T) {
		files := ServerFiles("", services)
		var sseFile *codegen.File
		for _, f := range files {
			if strings.HasSuffix(f.Path, filepath.Join("server", "sse.go")) {
				sseFile = f
				break
			}
		}
		require.NotNil(t, sseFile)

		sections := sseFile.Section("server-sse")
		require.NotEmpty(t, sections)
		code := codegen.SectionCode(t, sections[0])

		require.Contains(t, code, "payload = res")
		require.NotContains(t, code, "NewCreateResponseBody")
		testutil.NewGoldenFile(t, filepath.Join("testdata", "golden")).
			StringContent(code).
			Path("sse-mixed-results-server.golden").
			CompareContent()
	})

	t.Run("handler failure", func(t *testing.T) {
		files := ServerFiles("", services)
		for _, f := range files {
			for _, section := range f.Section("server-handler-init") {
				code := codegen.SectionCode(t, section)
				require.Contains(t, code, "lifecycle.HandlerFailed(err, stream.writer.ClaimFailure(), encodeError, errhandler)")
				require.Contains(t, code, "lifecycle.HandlerFailed(err, false, encodeError, errhandler)")
				require.Contains(t, code, "Stream:  &discardCreateServerStream{}")
				require.NotContains(t, code, "stream.started()")
				return
			}
		}
		t.Error("missing mixed-result handler")
	})

	t.Run("client", func(t *testing.T) {
		files := ClientFiles("", services)
		var sseFile *codegen.File
		for _, f := range files {
			if strings.HasSuffix(f.Path, filepath.Join("client", "sse.go")) {
				sseFile = f
				break
			}
		}
		require.NotNil(t, sseFile)

		sections := sseFile.Section("client-sse")
		require.NotEmpty(t, sections)
		code := codegen.SectionCode(t, sections[0])

		require.Contains(t, code, "event = new(")
		testutil.NewGoldenFile(t, filepath.Join("testdata", "golden")).
			StringContent(code).
			Path("sse-mixed-results-client.golden").
			CompareContent()
	})
}
