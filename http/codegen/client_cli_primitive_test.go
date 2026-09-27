package codegen

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestClientCLIPrimitiveImports(t *testing.T) {
	for _, tc := range []struct {
		name   string
		design func()
	}{
		{"integer body", testdata.IntValidationDSL},
		{"boolean body", testdata.PayloadBodyPrimitiveBoolValidateDSL},
		{"boolean path", testdata.PayloadPathPrimitiveBoolValidateDSL},
		{"boolean query", testdata.PayloadQueryPrimitiveBoolValidateDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := ClientCLIFiles("gen", CreateHTTPServices(RunHTTPDSL(t, tc.design)))
			parser := findFileWithSection(t, files, "parse-endpoint")
			path, err := parser.Render(t.TempDir())
			require.NoError(t, err)
			source, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Contains(t, string(source), "strconv.")
			require.Contains(t, string(source), `"strconv"`)
		})
	}
}
