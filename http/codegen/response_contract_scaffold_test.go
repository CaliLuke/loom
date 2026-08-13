package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/loomsource"
)

func TestResponseContractTestFiles(t *testing.T) {
	root := RunHTTPDSL(t, responseContractServerDSL)
	services := CreateHTTPServices(root)
	files := ResponseContractTestFiles("example.com/widgets/gen", services)
	require.Len(t, files, 1)
	require.Equal(t, filepath.Join("internal", "contracttest", "widgets_http_test.go"), files[0].Path)
	require.True(t, files[0].SkipExist)

	generated := codegen.SectionCode(t, files[0].Section("response-contract-test")[0])
	require.Contains(t, generated, "type widgetsResponseContractScenario func(*testing.T) *http.Response")
	require.Contains(t, generated, "func widgetsResponseContractScenarios() map[string]widgetsResponseContractScenario")
	require.Contains(t, generated, "func TestWidgetsHTTPResponseContracts(t *testing.T)")
	require.Contains(t, generated, "for _, contract := range widgetssvr.ResponseContractCases()")
	require.Contains(t, generated, `t.Errorf("missing response contract scenario %q", contract.ID)`)
	require.Contains(t, generated, "loomhttp.ValidateResponseContract(response, contract)")
	require.NotContains(t, generated, "t.Skip")
}

func TestResponseContractTestFilesPreserveExistingScaffold(t *testing.T) {
	root := RunHTTPDSL(t, responseContractServerDSL)
	file := ResponseContractTestFiles("example.com/widgets/gen", CreateHTTPServices(root))[0]
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

func TestResponseContractTestFilesPassRealGeneratedScenarios(t *testing.T) {
	const modulePath = "example.com/responsecontractprovider"
	root := RunHTTPDSL(t, responseContractProviderDSL)
	dir := t.TempDir()
	repoRoot, err := loomsource.RepositoryRoot(".")
	require.NoError(t, err)
	t.Setenv("LOOM_DIR", repoRoot)
	renderHTTPModule(t, dir, modulePath, root)
	renderGeneratedFiles(t, dir, ResponseContractTestFiles(modulePath+"/gen", CreateHTTPServices(root)))

	scaffoldPath := filepath.Join(dir, "internal", "contracttest", "widgets_http_test.go")
	scaffold, err := os.ReadFile(scaffoldPath)
	require.NoError(t, err)
	populated := strings.Replace(
		string(scaffold),
		"return map[string]widgetsResponseContractScenario{}",
		`return map[string]widgetsResponseContractScenario{
		"widgets.show.success.202.outcome=accepted": acceptedResponse,
		"widgets.show.success.200": fallbackResponse,
		"widgets.show.error.not_found.404": notFoundResponse,
	}`,
		1,
	)
	require.NotEqual(t, string(scaffold), populated)
	require.NoError(t, os.WriteFile(scaffoldPath, []byte(populated), 0o600))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "internal", "contracttest", "widgets_provider_test.go"),
		[]byte(responseContractProviderHarness),
		0o600,
	))

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./internal/contracttest")
}

func responseContractProviderDSL() {
	var NotFound = Type("NotFound", func() {
		Attribute("body", String)
		Attribute("reason", String)
		Required("body", "reason")
	})
	Service("widgets", func() {
		Method("show", func() {
			Result(func() {
				Attribute("body", String)
				Attribute("outcome", String)
				Attribute("version", String)
				Attribute("session", String)
				Required("body", "outcome", "version", "session")
			})
			Error("not_found", NotFound)
			HTTP(func() {
				GET("/widgets")
				Response(StatusAccepted, func() {
					Tag("outcome", "accepted")
					Body("body")
					Header("version:X-Version")
					SessionCookie("session:widget_session")
				})
				Response(StatusOK, func() {
					Body("body")
				})
				Response("not_found", StatusNotFound, func() {
					Body("body")
					Header("reason:X-Reason")
				})
			})
		})
	})
}

func TestResponseContractTestFilesFailMissingScenarios(t *testing.T) {
	const modulePath = "example.com/responsecontractscaffold"
	root := RunHTTPDSL(t, responseContractServerDSL)
	dir := t.TempDir()
	repoRoot, err := loomsource.RepositoryRoot(".")
	require.NoError(t, err)
	t.Setenv("LOOM_DIR", repoRoot)
	renderHTTPModule(t, dir, modulePath, root)
	renderGeneratedFiles(t, dir, ResponseContractTestFiles(modulePath+"/gen", CreateHTTPServices(root)))

	runGoCommand(t, dir, "mod", "tidy")
	output, err := runCommandAllowFailure(dir, "go", "test", "./internal/contracttest")
	require.Error(t, err)
	for _, id := range []string{
		"widgets.show.success.202.outcome=accepted",
		"widgets.show.success.200",
		"widgets.show.error.not_found.404",
	} {
		require.Contains(t, output, `missing response contract scenario "`+id+`"`)
	}
	require.NotContains(t, output, "build failed")

	path := filepath.Join(dir, "internal", "contracttest", "widgets_http_test.go")
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	withNilScenario := strings.Replace(
		string(content),
		"return map[string]widgetsResponseContractScenario{}",
		`return map[string]widgetsResponseContractScenario{
		"widgets.show.success.202.outcome=accepted": func(*testing.T) *http.Response { return nil },
	}`,
		1,
	)
	require.NotEqual(t, string(content), withNilScenario)
	require.NoError(t, os.WriteFile(path, []byte(withNilScenario), 0o600))

	output, err = runCommandAllowFailure(dir, "go", "test", "./internal/contracttest")
	require.Error(t, err)
	require.Contains(t, output, `response contract "widgets.show.success.202.outcome=accepted": response is nil`)
	require.NotContains(t, output, `missing response contract scenario "widgets.show.success.202.outcome=accepted"`)
}

const responseContractProviderHarness = `package contracttest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	widgets "example.com/responsecontractprovider/gen/widgets"
	widgetserver "example.com/responsecontractprovider/gen/http/widgets/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

type widgetScenario int

const (
	acceptedScenario widgetScenario = iota
	fallbackScenario
	notFoundScenario
)

type widgetsService struct {
	scenario widgetScenario
}

func acceptedResponse(t *testing.T) *http.Response {
	return requestWidgetScenario(t, acceptedScenario)
}

func fallbackResponse(t *testing.T) *http.Response {
	return requestWidgetScenario(t, fallbackScenario)
}

func notFoundResponse(t *testing.T) *http.Response {
	return requestWidgetScenario(t, notFoundScenario)
}

func requestWidgetScenario(t *testing.T, scenario widgetScenario) *http.Response {
	t.Helper()
	endpoints := widgets.NewEndpoints(&widgetsService{scenario: scenario})
	mux := loomhttp.NewMuxer()
	server := widgetserver.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	widgetserver.Mount(mux, server)
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, httpServer.URL+"/widgets", nil)
	if err != nil {
		t.Fatalf("create widgets request: %v", err)
	}
	response, err := httpServer.Client().Do(request)
	if err != nil {
		t.Fatalf("request widgets scenario: %v", err)
	}
	t.Cleanup(func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close widgets response: %v", err)
		}
	})
	return response
}

func (s *widgetsService) Show(context.Context) (*widgets.ShowResult, error) {
	switch s.scenario {
	case acceptedScenario:
		return &widgets.ShowResult{
			Body:    "accepted",
			Outcome: "accepted",
			Version: "v2",
			Session: "session-accepted",
		}, nil
	case fallbackScenario:
		return &widgets.ShowResult{
			Body:    "fallback",
			Outcome: "fallback",
			Version: "v2",
			Session: "session-fallback",
		}, nil
	case notFoundScenario:
		return nil, &widgets.NotFound{Body: "widget not found", Reason: "missing widget"}
	default:
		return nil, nil
	}
}
`
