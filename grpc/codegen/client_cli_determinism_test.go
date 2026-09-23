package codegen

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

const clientCLIDeterminismHelperPath = "LOOM_GRPC_CLIENT_CLI_DETERMINISM_HELPER_PATH"

var clientCLIHelperFuncPattern = regexp.MustCompile(`(?m)^func (protobuf\w+)\(`)

func TestClientCLIFilesAreByteStableAcrossIndependentProcesses(t *testing.T) {
	if path := os.Getenv(clientCLIDeterminismHelperPath); path != "" {
		require.NoError(t, os.WriteFile(path, renderClientCLIDeterminismFile(t), 0o600))
		return
	}

	outputs := make([][]byte, 0, 3)
	for range 3 {
		path := filepath.Join(t.TempDir(), "cli.go")
		cmd := exec.Command(os.Args[0], "-test.run=^TestClientCLIFilesAreByteStableAcrossIndependentProcesses$")
		cmd.Env = append(os.Environ(), clientCLIDeterminismHelperPath+"="+path)
		output, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "isolated gRPC client CLI generation failed:\n%s", output)
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		outputs = append(outputs, content)
	}
	for i, content := range outputs[1:] {
		require.Equal(t, string(outputs[0]), string(content), "client CLI differs in isolated generation %d", i+2)
	}

	var helpers []string
	for _, match := range clientCLIHelperFuncPattern.FindAllSubmatch(outputs[0], -1) {
		helpers = append(helpers, string(match[1]))
	}
	require.Equal(t, []string{
		"protobufDeterminismpbAlphaToDeterminismAlpha",
		"protobufDeterminismpbBravoToDeterminismBravo",
		"protobufDeterminismpbCharlieToDeterminismCharlie",
		"protobufDeterminismpbDeltaToDeterminismDelta",
		"protobufDeterminismpbEchoToDeterminismEcho",
		"protobufDeterminismpbFoxtrotToDeterminismFoxtrot",
	}, helpers)
}

func renderClientCLIDeterminismFile(t *testing.T) []byte {
	t.Helper()
	root := RunGRPCDSL(t, clientCLIDeterminismDSL)
	files := ClientCLIFiles("example.com/determinism/gen", CreateGRPCServices(root))
	require.Greater(t, len(files), 1, "expected the service client CLI file")
	var buf bytes.Buffer
	for _, section := range files[1].AllSections() {
		require.NoError(t, section.Write(&buf))
	}
	return buf.Bytes()
}

func clientCLIDeterminismDSL() {
	nested := func(name string) any {
		return Type(name, func() {
			Field(1, "value", String)
			Required("value")
		})
	}
	alpha := nested("Alpha")
	bravo := nested("Bravo")
	charlie := nested("Charlie")
	delta := nested("Delta")
	echo := nested("Echo")
	foxtrot := nested("Foxtrot")
	Service("Determinism", func() {
		Method("Run", func() {
			Payload(func() {
				Field(1, "alpha", alpha)
				Field(2, "bravo", bravo)
				Field(3, "charlie", charlie)
				Field(4, "delta", delta)
				Field(5, "echo", echo)
				Field(6, "foxtrot", foxtrot)
				Required("alpha", "bravo", "charlie", "delta", "echo", "foxtrot")
			})
			GRPC(func() {})
		})
	})
}
