package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

const primitiveCLIHarness = `package primitivecli_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	cli %q
)

func TestPrimitivePayload(t *testing.T) {
	args := os.Args
	t.Cleanup(func() {
		os.Args = args
	})
	commands := cli.UsageCommands()
	require.Len(t, commands, 1)
	for _, tc := range []struct {
		name string
		value string
		valid bool
	}{
		{"valid", %q, true},
		{"invalid", "not-a-value", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			os.Args = append([]string{"primitive-cli"}, strings.Fields(commands[0])...)
			os.Args = append(os.Args, "--p", tc.value)
			endpoint, payload, err := cli.ParseEndpoint("http", "localhost", nil, nil, nil, false)
			if tc.valid {
				require.NoError(t, err)
				require.NotNil(t, endpoint)
				require.Equal(t, %s, payload)
			} else {
				require.ErrorContains(t, err, "invalid value")
				require.Nil(t, endpoint)
				require.Nil(t, payload)
			}
		})
	}
}
`

func TestCLIPrimitivePayloadCompile(t *testing.T) {
	for _, tc := range []struct {
		name   string
		design func()
		value  string
	}{
		{"integer body", testdata.IntValidationDSL, "1"},
		{"boolean body", testdata.PayloadBodyPrimitiveBoolValidateDSL, "true"},
		{"boolean path", testdata.PayloadPathPrimitiveBoolValidateDSL, "true"},
		{"boolean query", testdata.PayloadQueryPrimitiveBoolValidateDSL, "true"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			codegen.RunDSL(t, tc.design)
			dir := t.TempDir()
			goMod := fmt.Sprintf("module example.com/primitivecli\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", loomModuleSource(t))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))
			_, err := Generate(dir, "gen", false)
			require.NoError(t, err)
			_, err = Generate(dir, "example", false)
			require.NoError(t, err)
			parsers, err := filepath.Glob(filepath.Join(dir, "gen", "http", "cli", "*", "cli.go"))
			require.NoError(t, err)
			require.Len(t, parsers, 1)
			parserPath, err := filepath.Rel(dir, filepath.Dir(parsers[0]))
			require.NoError(t, err)
			harness := fmt.Sprintf(primitiveCLIHarness, "example.com/primitivecli/"+filepath.ToSlash(parserPath), tc.value, tc.value)
			require.NoError(t, os.MkdirAll(filepath.Join(dir, "probes"), 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "probes", "cli_test.go"), []byte(harness), 0o600))
			for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}, {"test", "-count=1", "./probes"}} {
				output, err := testingx.RunCmd(dir, "go", args...)
				require.NoError(t, err, output)
			}
		})
	}
}
