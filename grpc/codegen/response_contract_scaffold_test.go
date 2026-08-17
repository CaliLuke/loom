package codegen

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
	"github.com/CaliLuke/loom/internal/loomsource"
)

func TestResponseContractTestFiles(t *testing.T) {
	root := RunGRPCDSL(t, testdata.UnaryRPCWithErrorsDSL)
	files := ResponseContractTestFiles("example.com/widgets/gen", CreateGRPCServices(root))
	require.Len(t, files, 1)
	require.Equal(t, filepath.Join("internal", "contracttest", "service_unary_rpc_with_errors_grpc_test.go"), files[0].Path)
	require.True(t, files[0].SkipExist)

	generated := codegen.SectionCode(t, files[0].Section("response-contract-test")[0])
	require.Contains(t, generated, "type serviceunaryrpcwitherrorsGRPCResponseContractScenario func(*testing.T, loomgrpc.ResponseContractCase) *loomgrpc.ResponseContractObservation")
	require.Contains(t, generated, "func TestServiceunaryrpcwitherrorsGRPCResponseContracts(t *testing.T)")
	require.Contains(t, generated, "for _, contract := range serviceunaryrpcwitherrorssvr.ResponseContractCases()")
	require.Contains(t, generated, "loomgrpc.ValidateResponseContract(observation, contract)")
	require.Contains(t, generated, `t.Errorf("missing gRPC response contract scenario %q", contract.ID)`)
	require.Contains(t, generated, `t.Errorf("gRPC response contract scenario %q has no declared contract", id)`)
}

func TestResponseContractTestFilesPreserveExistingScaffold(t *testing.T) {
	root := RunGRPCDSL(t, testdata.UnaryRPCNoPayloadDSL)
	file := ResponseContractTestFiles("example.com/widgets/gen", CreateGRPCServices(root))[0]
	dir := t.TempDir()
	path := filepath.Join(dir, file.Path)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, []byte("consumer owned\n"), 0o600))

	written, err := file.Render(dir)
	require.NoError(t, err)
	require.Empty(t, written)
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "consumer owned\n", string(content))
}

func TestResponseContractTestFilesPassRealGeneratedGRPCScenario(t *testing.T) {
	const modulePath = "example.com/grpcresponsecontract"
	root := RunGRPCDSL(t, testdata.UnaryRPCNoPayloadDSL)
	dir := t.TempDir()
	renderGRPCResponseContractModule(t, dir, modulePath, root)

	scaffoldPath := filepath.Join(dir, "internal", "contracttest", "service_unary_rpc_no_payload_grpc_test.go")
	scaffold, err := os.ReadFile(scaffoldPath)
	require.NoError(t, err)
	populated := string(scaffold)
	populated = strings.Replace(populated,
		"return map[string]serviceunaryrpcnopayloadGRPCResponseContractScenario{}",
		`return map[string]serviceunaryrpcnopayloadGRPCResponseContractScenario{
		"ServiceUnaryRPCNoPayload.MethodUnaryRPCNoPayload.success.0": observeUnarySuccess,
	}`,
		1,
	)
	require.NotEqual(t, string(scaffold), populated)
	require.NoError(t, os.WriteFile(scaffoldPath, []byte(populated), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "internal", "contracttest", "grpc_provider_test.go"), []byte(grpcResponseContractHarness), 0o600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "test", "./internal/contracttest")
}

func renderGRPCResponseContractModule(t *testing.T, dir, modulePath string, root *expr.RootExpr) {
	t.Helper()
	genpkg := modulePath + "/gen"
	serviceData := servicecodegen.NewServicesData(root)
	grpcData := NewServicesData(serviceData)
	var files []*codegen.File
	userTypePkgs := make(map[string][]string)
	for _, service := range root.Services {
		files = append(files, servicecodegen.Files(genpkg, service, serviceData, userTypePkgs)...)
		if views := servicecodegen.ViewsFile(genpkg, service, serviceData); views != nil {
			files = append(files, views)
		}
		files = append(files, servicecodegen.EndpointFile(genpkg, service, serviceData))
	}
	files = append(files, ProtoFiles(genpkg, grpcData)...)
	files = append(files, ClientTypeFiles(genpkg, grpcData)...)
	files = append(files, ClientFiles(genpkg, grpcData)...)
	files = append(files, ServerTypeFiles(genpkg, grpcData)...)
	files = append(files, ServerFiles(genpkg, grpcData)...)
	files = append(files, ResponseContractTestFiles(genpkg, grpcData)...)
	for _, file := range files {
		_, err := file.Render(dir)
		require.NoError(t, err, file.Path)
	}

	repoRoot, err := loomsource.RepositoryRoot(".")
	require.NoError(t, err)
	goMod := fmt.Sprintf("module %s\n\ngo 1.27rc2\n\nrequire github.com/CaliLuke/loom v1.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", modulePath, repoRoot)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))
}

func runGRPCGoCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("go", args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	require.NoError(t, err, "%s", output)
}

const grpcResponseContractHarness = `package contracttest

import (
	"context"
	"net"
	"testing"

	loomgrpc "github.com/CaliLuke/loom/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	serviceunaryrpcnopayload "example.com/grpcresponsecontract/gen/service_unary_rpc_no_payload"
	serviceunaryrpcnopayloadpb "example.com/grpcresponsecontract/gen/grpc/service_unary_rpc_no_payload/pb"
	serviceunaryrpcnopayloadsvr "example.com/grpcresponsecontract/gen/grpc/service_unary_rpc_no_payload/server"
)

type unaryService struct{}

func (*unaryService) MethodUnaryRPCNoPayload(context.Context) (string, error) {
	return "ok", nil
}

func observeUnarySuccess(t *testing.T, _ loomgrpc.ResponseContractCase) *loomgrpc.ResponseContractObservation {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	serviceunaryrpcnopayloadpb.RegisterServiceUnaryRPCNoPayloadServer(
		server,
		serviceunaryrpcnopayloadsvr.New(serviceunaryrpcnopayload.NewEndpoints(&unaryService{}), nil),
	)
	go func() {
		if err := server.Serve(listener); err != nil {
			t.Error(err)
		}
	}()
	t.Cleanup(server.Stop)

	conn, err := grpc.NewClient("passthrough:///bufconn", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	})
	client := serviceunaryrpcnopayloadpb.NewServiceUnaryRPCNoPayloadClient(conn)
	message, callErr := client.MethodUnaryRPCNoPayload(context.Background(), &serviceunaryrpcnopayloadpb.MethodUnaryRPCNoPayloadRequest{})
	return &loomgrpc.ResponseContractObservation{Message: message, Error: callErr}
}
`
