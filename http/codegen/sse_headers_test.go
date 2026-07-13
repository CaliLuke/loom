package codegen

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestGeneratedSSEHandlerUsesSharedHeaderRuntime(t *testing.T) {
	root := RunHTTPDSL(t, testdata.SSEObjectDSL)
	files := ServerFiles("", CreateHTTPServices(root))
	var serverFile *codegen.File
	for _, file := range files {
		if strings.HasSuffix(file.Path, filepath.Join("server", "server.go")) {
			serverFile = file
			break
		}
	}
	require.NotNil(t, serverFile)
	code := codegen.SectionCode(t, serverFile.Section("server-handler-init")[0])

	require.Contains(t, code, "loomhttp.NewSSEStreamWriter(")
	require.Contains(t, code, "loomtransport.TransportHTTP")
}
