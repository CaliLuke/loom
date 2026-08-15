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
	require.Contains(t, generated, `t.Errorf("response contract scenario %q has no declared contract", id)`)
	require.Contains(t, generated, "loomhttp.ValidateResponseContract(response, contract)")
	require.NotContains(t, generated, "t.Skip")
}

func TestResponseContractTestFilesIncludeSSEScenarios(t *testing.T) {
	root := RunHTTPDSL(t, responseContractSSEDSL)
	services := CreateHTTPServices(root)
	files := ResponseContractTestFiles("example.com/events/gen", services)
	require.Len(t, files, 1)

	generated := codegen.SectionCode(t, files[0].Section("response-contract-test")[0])
	require.Contains(t, generated, "type eventsSSEResponseContractScenario func(*testing.T) *loomhttp.SSEResponseContractObservation")
	require.Contains(t, generated, "func eventsSSEResponseContractScenarios() map[string]eventsSSEResponseContractScenario")
	require.Contains(t, generated, "contract.Transport == loomhttp.ResponseContractSSE")
	require.Contains(t, generated, "loomhttp.ValidateSSEResponseContract(observation, contract)")
	require.Contains(t, generated, `missing SSE response contract scenario %q`)
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

func TestResponseContractTestFilesUseDistinctServiceAliasPrefixes(t *testing.T) {
	root := RunHTTPDSL(t, responseContractAliasCollisionDSL)
	files := ResponseContractTestFiles("example.com/contracts/gen", CreateHTTPServices(root))
	require.Len(t, files, 2)

	generated := make([]string, 0, len(files))
	for _, file := range files {
		generated = append(generated, codegen.SectionCode(t, file.Section("response-contract-test")[0]))
	}
	require.Contains(t, generated[0], "type bodysvcResponseContractScenario")
	require.Contains(t, generated[1], "type bodysvcsvcResponseContractScenario")
}

func TestResponseContractTestFilesPassRealGeneratedScenarios(t *testing.T) {
	runResponseContractScaffold(t, responseContractScaffoldTest{
		modulePath:   "example.com/responsecontractprovider",
		design:       responseContractProviderDSL,
		scaffoldName: "widgets_http_test.go",
		emptyMap:     "return map[string]widgetsResponseContractScenario{}",
		scenarios: `return map[string]widgetsResponseContractScenario{
		"widgets.show.success.202.outcome=accepted": acceptedResponse,
		"widgets.show.success.200": fallbackResponse,
		"widgets.show.error.not_found.404": notFoundResponse,
	}`,
		harnessName: "widgets_provider_test.go",
		harness:     responseContractProviderHarness,
	})
}

func TestResponseContractTestFilesPassFileAndBodylessGeneratedScenarios(t *testing.T) {
	runResponseContractScaffold(t, responseContractScaffoldTest{
		modulePath:   "example.com/responsecontractfilebodyless",
		design:       responseContractFileAndBodylessDSL,
		scaffoldName: "files_http_test.go",
		emptyMap:     "return map[string]filesResponseContractScenario{}",
		scenarios: `return map[string]filesResponseContractScenario{
		"files.download.success.200": downloadResponse,
		"files.no_content.success.204": noContentResponse,
	}`,
		harnessName: "files_provider_test.go",
		harness:     responseContractFileAndBodylessHarness,
	})
}

func TestResponseContractTestFilesPassSSESuccessAndPreStreamErrorScenarios(t *testing.T) {
	const modulePath = "example.com/responsecontractsse"
	root := RunHTTPDSL(t, responseContractSSEDSL)
	dir := t.TempDir()
	repoRoot, err := loomsource.RepositoryRoot(".")
	require.NoError(t, err)
	t.Setenv("LOOM_DIR", repoRoot)
	renderHTTPModule(t, dir, modulePath, root)
	renderGeneratedFiles(t, dir, ResponseContractTestFiles(modulePath+"/gen", CreateHTTPServices(root)))

	scaffoldPath := filepath.Join(dir, "internal", "contracttest", "events_http_test.go")
	scaffold, err := os.ReadFile(scaffoldPath)
	require.NoError(t, err)
	populated := strings.Replace(
		string(scaffold),
		"return map[string]eventsResponseContractScenario{}",
		`return map[string]eventsResponseContractScenario{
		"events.watch.error.unauthorized.401": unauthorizedResponse,
	}`,
		1,
	)
	populated = strings.Replace(
		populated,
		"return map[string]eventsSSEResponseContractScenario{}",
		`return map[string]eventsSSEResponseContractScenario{
		"events.watch.success.200": watchSSEObservation,
	}`,
		1,
	)
	require.NotEqual(t, string(scaffold), populated)
	require.NoError(t, os.WriteFile(scaffoldPath, []byte(populated), 0o600))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "internal", "contracttest", "events_provider_test.go"),
		[]byte(responseContractSSEProviderHarness),
		0o600,
	))

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./internal/contracttest")
}

type responseContractScaffoldTest struct {
	modulePath   string
	design       func()
	scaffoldName string
	emptyMap     string
	scenarios    string
	harnessName  string
	harness      string
}

func runResponseContractScaffold(t *testing.T, test responseContractScaffoldTest) {
	t.Helper()
	root := RunHTTPDSL(t, test.design)
	dir := t.TempDir()
	repoRoot, err := loomsource.RepositoryRoot(".")
	require.NoError(t, err)
	t.Setenv("LOOM_DIR", repoRoot)
	renderHTTPModule(t, dir, test.modulePath, root)
	renderGeneratedFiles(t, dir, ResponseContractTestFiles(test.modulePath+"/gen", CreateHTTPServices(root)))

	scaffoldPath := filepath.Join(dir, "internal", "contracttest", test.scaffoldName)
	scaffold, err := os.ReadFile(scaffoldPath)
	require.NoError(t, err)
	require.Contains(t, string(scaffold), "// Generated once by Loom. Edit freely; regeneration never overwrites this file.")
	require.NotContains(t, string(scaffold), "DO NOT EDIT")
	populated := strings.Replace(string(scaffold), test.emptyMap, test.scenarios, 1)
	require.NotEqual(t, string(scaffold), populated)
	require.NoError(t, os.WriteFile(scaffoldPath, []byte(populated), 0o600))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "internal", "contracttest", test.harnessName),
		[]byte(test.harness),
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

func responseContractFileAndBodylessDSL() {
	Service("files", func() {
		Method("download", func() {
			Result(func() {
				Attribute("etag", String)
				Required("etag")
			})
			HTTP(func() {
				GET("/files/download")
				FileResponse()
				Response(func() {
					Header("etag:ETag")
				})
			})
		})
		Method("no_content", func() {
			Result(Empty)
			HTTP(func() {
				GET("/files/empty")
				Response(StatusNoContent)
			})
		})
	})
}

func responseContractAliasCollisionDSL() {
	for _, serviceName := range []string{"body", "bodysvc"} {
		Service(serviceName, func() {
			Method("show", func() {
				Result(String)
				HTTP(func() {
					GET("/items")
					Response(StatusOK)
				})
			})
		})
	}
}

func responseContractSSEDSL() {
	Service("events", func() {
		Method("watch", func() {
			Error("unauthorized")
			StreamingResult(func() {
				Attribute("id", String)
				Attribute("event", String)
				Attribute("data", String)
				Required("id", "event", "data")
			})
			HTTP(func() {
				GET("/events")
				Response("unauthorized", StatusUnauthorized)
				ServerSentEvents(func() {
					SSEEventID("id")
					SSEEventType("event")
					SSEEventData("data")
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

	withStaleScenario := strings.Replace(
		string(content),
		"return map[string]widgetsResponseContractScenario{}",
		`return map[string]widgetsResponseContractScenario{
		"widgets.show.success.299": func(*testing.T) *http.Response { return nil },
	}`,
		1,
	)
	require.NotEqual(t, string(content), withStaleScenario)
	require.NoError(t, os.WriteFile(path, []byte(withStaleScenario), 0o600))

	output, err = runCommandAllowFailure(dir, "go", "test", "./internal/contracttest")
	require.Error(t, err)
	require.Contains(t, output, `response contract scenario "widgets.show.success.299" has no declared contract`)
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

const responseContractSSEProviderHarness = `package contracttest

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	events "example.com/responsecontractsse/gen/events"
	eventsserver "example.com/responsecontractsse/gen/http/events/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

type eventsService struct {
	unauthorized bool
}

func watchSSEObservation(t *testing.T) *loomhttp.SSEResponseContractObservation {
	t.Helper()
	response := requestEventsScenario(t, false)
	observed, err := loomhttp.ParseSSEStream(response.Body)
	if closeErr := response.Body.Close(); closeErr != nil {
		t.Errorf("close SSE response: %v", closeErr)
	}
	return &loomhttp.SSEResponseContractObservation{
		Response:      response,
		Events:        observed,
		TerminalError: err,
	}
}

func unauthorizedResponse(t *testing.T) *http.Response {
	return requestEventsScenario(t, true)
}

func requestEventsScenario(t *testing.T, unauthorized bool) *http.Response {
	t.Helper()
	endpoints := events.NewEndpoints(&eventsService{unauthorized: unauthorized})
	mux := loomhttp.NewMuxer()
	server := eventsserver.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	eventsserver.Mount(mux, server)
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, httpServer.URL+"/events", nil)
	if err != nil {
		t.Fatalf("create events request: %v", err)
	}
	request.Header.Set("Accept", "text/event-stream")
	response, err := httpServer.Client().Do(request)
	if err != nil {
		t.Fatalf("request events scenario: %v", err)
	}
	if unauthorized {
		t.Cleanup(func() {
			if err := response.Body.Close(); err != nil {
				t.Errorf("close unauthorized response: %v", err)
			}
		})
	}
	return response
}

func (s *eventsService) Watch(_ context.Context, stream events.WatchServerStream) error {
	if s.unauthorized {
		return events.MakeUnauthorized(errors.New("unauthorized"))
	}
	return stream.Send(&events.WatchResult{
		ID:    "event-1",
		Event: "created",
		Data:  "payload",
	})
}
`

const responseContractFileAndBodylessHarness = `package contracttest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	files "example.com/responsecontractfilebodyless/gen/files"
	filesserver "example.com/responsecontractfilebodyless/gen/http/files/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func downloadResponse(t *testing.T) *http.Response {
	return requestFilesScenario(t, "/files/download")
}

func noContentResponse(t *testing.T) *http.Response {
	return requestFilesScenario(t, "/files/empty")
}

func requestFilesScenario(t *testing.T, path string) *http.Response {
	t.Helper()
	endpoints := files.NewEndpoints(&filesService{})
	mux := loomhttp.NewMuxer()
	server := filesserver.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	filesserver.Mount(mux, server)
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, httpServer.URL+path, nil)
	if err != nil {
		t.Fatalf("create files request: %v", err)
	}
	response, err := httpServer.Client().Do(request)
	if err != nil {
		t.Fatalf("request files scenario: %v", err)
	}
	t.Cleanup(func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close files response: %v", err)
		}
	})
	return response
}

type filesService struct{}

func (*filesService) Download(context.Context) (*files.DownloadResult, *loomhttp.FileResponse, error) {
	return &files.DownloadResult{Etag: "\"example\""}, &loomhttp.FileResponse{
		Name:    "example.bin",
		Content: strings.NewReader("file bytes"),
	}, nil
}

func (*filesService) NoContent(context.Context) error {
	return nil
}
`
