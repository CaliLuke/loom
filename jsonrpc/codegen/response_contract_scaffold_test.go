package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
)

func TestResponseContractTestFiles(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonRPCTypedErrorBodyDSL)
	files := ResponseContractTestFiles("example.com/tools/gen", CreateJSONRPCServices(root))
	require.Len(t, files, 1)
	require.Equal(t, filepath.Join("internal", "contracttest", "tools_jsonrpc_test.go"), files[0].Path)
	require.True(t, files[0].SkipExist)
	generated := codegen.SectionCode(t, files[0].Section("response-contract-test")[0])
	require.Contains(t, generated, "type toolsJSONRPCResponseContractScenario func(*testing.T, loomjsonrpc.ResponseContractCase) *loomjsonrpc.ResponseContractObservation")
	require.Contains(t, generated, "func TestToolsJSONRPCResponseContracts(t *testing.T)")
	require.Contains(t, generated, "for _, contract := range toolsjssvr.ResponseContractCases()")
	require.Contains(t, generated, "loomjsonrpc.ValidateResponseContract(observation, contract)")
}

func TestResponseContractTestFilesPreserveExistingScaffold(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonRPCTypedErrorBodyDSL)
	file := ResponseContractTestFiles("example.com/tools/gen", CreateJSONRPCServices(root))[0]
	dir := t.TempDir()
	path := filepath.Join(dir, file.Path)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, []byte("consumer owned\n"), 0o600))
	written, err := file.Render(dir)
	require.NoError(t, err)
	require.Empty(t, written)
}

func TestResponseContractTestFilesPassRealGeneratedHandler(t *testing.T) {
	const modulePath = "example.com/jsonrpcresponsecontract"
	root := RunJSONRPCDSL(t, responseContractHandlerDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, modulePath, root)
	renderCodegenFiles(t, dir, ResponseContractTestFiles(modulePath+"/gen", CreateJSONRPCServices(root)))

	scaffoldPath := filepath.Join(dir, "internal", "contracttest", "tools_jsonrpc_test.go")
	scaffold, err := os.ReadFile(scaffoldPath)
	require.NoError(t, err)
	populated := strings.Replace(string(scaffold),
		"return map[string]toolsJSONRPCResponseContractScenario{}",
		`return map[string]toolsJSONRPCResponseContractScenario{
		"tools.call.success": observeSuccess,
		"tools.call.notification": observeNotification,
	}`,
		1,
	)
	require.NotEqual(t, string(scaffold), populated)
	require.NoError(t, os.WriteFile(scaffoldPath, []byte(populated), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "internal", "contracttest", "jsonrpc_provider_test.go"), []byte(jsonrpcResponseContractHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "./internal/contracttest")
}

func responseContractHandlerDSL() {
	dsl.Service("tools", func() {
		dsl.JSONRPC(func() { dsl.POST("/rpc") })
		dsl.Method("call", func() {
			dsl.Result(dsl.String)
			dsl.JSONRPC(func() {})
		})
	})
}

func responseContractUnsupportedWebSocketDSL() {
	dsl.Service("events", func() {
		dsl.JSONRPC(func() { dsl.GET("/rpc") })
		dsl.Method("watch", func() {
			dsl.StreamingResult(dsl.String)
			dsl.JSONRPC(func() {})
		})
	})
}

const jsonrpcResponseContractHarness = `package contracttest

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	tools "example.com/jsonrpcresponsecontract/gen/tools"
	toolsserver "example.com/jsonrpcresponsecontract/gen/jsonrpc/tools/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loomjsonrpc "github.com/CaliLuke/loom/jsonrpc"
)

type toolsService struct{}

func (*toolsService) Call(context.Context) (string, error) {
	return "ok", nil
}

func observeSuccess(t *testing.T, _ loomjsonrpc.ResponseContractCase) *loomjsonrpc.ResponseContractObservation {
	return observeCall(t, []byte("{\"jsonrpc\":\"2.0\",\"method\":\"call\",\"id\":1}"))
}

func observeNotification(t *testing.T, _ loomjsonrpc.ResponseContractCase) *loomjsonrpc.ResponseContractObservation {
	return observeCall(t, []byte("{\"jsonrpc\":\"2.0\",\"method\":\"call\"}"))
}

func observeCall(t *testing.T, body []byte) *loomjsonrpc.ResponseContractObservation {
	t.Helper()
	mux := loomhttp.NewMuxer()
	server := toolsserver.New(
		tools.NewEndpoints(&toolsService{}),
		mux,
		loomhttp.RequestDecoder,
		loomhttp.ResponseEncoder,
		func(_ context.Context, _ http.ResponseWriter, err error) { t.Error(err) },
	)
	toolsserver.Mount(mux, server)
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)
	request, err := http.NewRequest(http.MethodPost, httpServer.URL+"/rpc", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := httpServer.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return &loomjsonrpc.ResponseContractObservation{Response: response}
}
`
