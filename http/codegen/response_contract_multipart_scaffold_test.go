package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/internal/loomsource"
)

func TestResponseContractTestFilesIncludeMultipartScenarios(t *testing.T) {
	root := RunHTTPDSL(t, responseContractMultipartServerDSL)
	services := CreateHTTPServices(root)
	files := ResponseContractTestFiles("example.com/imports/gen", services)
	require.Len(t, files, 1)

	generated := codegen.SectionCode(t, files[0].Section("response-contract-test")[0])
	require.Contains(t, generated, "type importsMultipartResponseContractScenario func(*testing.T, loomhttp.MultipartRequestContract) *http.Response")
	require.Contains(t, generated, "func importsMultipartResponseContractScenarios() map[string]importsMultipartResponseContractScenario")
	require.Contains(t, generated, "contract.Multipart != nil")
	require.Contains(t, generated, "response := scenario(t, *contract.Multipart)")
	require.Contains(t, generated, `missing multipart response contract scenario %q`)
}

func TestResponseContractTestFilesPassMultipartSuccessAndErrorScenarios(t *testing.T) {
	const modulePath = "example.com/responsecontractmultipart"
	root := RunHTTPDSL(t, responseContractMultipartServerDSL)
	dir := t.TempDir()
	repoRoot, err := loomsource.RepositoryRoot(".")
	require.NoError(t, err)
	t.Setenv("LOOM_DIR", repoRoot)
	renderHTTPModule(t, dir, modulePath, root)
	renderGeneratedFiles(t, dir, ResponseContractTestFiles(modulePath+"/gen", CreateHTTPServices(root)))

	scaffoldPath := filepath.Join(dir, "internal", "contracttest", "imports_http_test.go")
	scaffold, err := os.ReadFile(scaffoldPath)
	require.NoError(t, err)
	populated := strings.Replace(
		string(scaffold),
		"return map[string]importsMultipartResponseContractScenario{}",
		`return map[string]importsMultipartResponseContractScenario{
		"imports.create.success.202": acceptedImportResponse,
		"imports.create.error.bad_request.400": badImportResponse,
	}`,
		1,
	)
	require.NotEqual(t, string(scaffold), populated)
	require.NoError(t, os.WriteFile(scaffoldPath, []byte(populated), 0o600))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "internal", "contracttest", "imports_provider_test.go"),
		[]byte(responseContractMultipartProviderHarness),
		0o600,
	))

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./internal/contracttest")
}

const responseContractMultipartProviderHarness = `package contracttest

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	imports "example.com/responsecontractmultipart/gen/imports"
	importsserver "example.com/responsecontractmultipart/gen/http/imports/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func acceptedImportResponse(t *testing.T, contract loomhttp.MultipartRequestContract) *http.Response {
	return requestImportScenario(t, contract, "accept")
}

func badImportResponse(t *testing.T, contract loomhttp.MultipartRequestContract) *http.Response {
	return requestImportScenario(t, contract, "reject")
}

func requestImportScenario(t *testing.T, contract loomhttp.MultipartRequestContract, label string) *http.Response {
	t.Helper()
	if contract.ContentType != "multipart/form-data" {
		t.Fatalf("multipart content type = %q", contract.ContentType)
	}
	if len(contract.Parts) != 2 {
		t.Fatalf("multipart parts = %d, want 2", len(contract.Parts))
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile(contract.Parts[0].Name, "sample.bin")
	if err != nil {
		t.Fatalf("create file part: %v", err)
	}
	if _, err := file.Write([]byte("payload")); err != nil {
		t.Fatalf("write file part: %v", err)
	}
	if err := writer.WriteField(contract.Parts[1].Name, label); err != nil {
		t.Fatalf("write label part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	endpoints := imports.NewEndpoints(&importsService{})
	mux := loomhttp.NewMuxer()
	server := importsserver.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	importsserver.Mount(mux, server)
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)

	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, httpServer.URL+"/imports", &body)
	if err != nil {
		t.Fatalf("create imports request: %v", err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := httpServer.Client().Do(request)
	if err != nil {
		t.Fatalf("request import scenario: %v", err)
	}
	t.Cleanup(func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close import response: %v", err)
		}
	})
	return response
}

type importsService struct{}

func (*importsService) Create(_ context.Context, payload *imports.CreatePayload) (*imports.CreateResult, error) {
	if payload.Label == "reject" {
		return nil, imports.MakeBadRequest(errors.New("rejected"))
	}
	return &imports.CreateResult{Receipt: "accepted", ImportID: "import-1"}, nil
}
`
