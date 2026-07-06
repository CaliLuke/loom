package example

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/example/testdata"
)

func TestExampleCLIFiles(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"no-server", testdata.NoServerDSL},
		{"single-server-single-host", testdata.SingleServerSingleHostDSL},
		{"single-server-single-host-with-variables", testdata.SingleServerSingleHostWithVariablesDSL},
		{"single-server-multiple-hosts", testdata.SingleServerMultipleHostsDSL},
		{"single-server-multiple-hosts-with-variables", testdata.SingleServerMultipleHostsWithVariablesDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := codegen.RunDSL(t, c.DSL)
			fs := CLIFiles("", root)
			require.Len(t, fs, 1)
			require.Greater(t, len(fs[0].AllSections()), 0)
			var buf bytes.Buffer
			for _, s := range fs[0].AllSections()[1:] {
				require.NoError(t, s.Write(&buf))
			}
			code := codegen.FormatTestCode(t, "package foo\n"+buf.String())
			golden := filepath.Join("testdata", "client-"+c.Name+".golden")
			compareOrUpdateGolden(t, code, golden)
		})
	}
}

func TestExampleCLIHelpUsesErrorsIs(t *testing.T) {
	root := codegen.RunDSL(t, testdata.NoServerDSL)
	fs := CLIFiles("", root)
	require.Len(t, fs, 1)
	require.Greater(t, len(fs[0].AllSections()), 0)
	var buf bytes.Buffer
	for _, s := range fs[0].AllSections()[1:] {
		require.NoError(t, s.Write(&buf))
	}
	code := codegen.FormatTestCode(t, "package foo\n"+buf.String())

	require.Contains(t, code, "errors.Is(err, flag.ErrHelp)")
	require.NotContains(t, code, "err == flag.ErrHelp")
}
