package codegen

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/CaliLuke/loom/internal/loomsource"
)

func TestServerResponseContractCases(t *testing.T) {
	root := RunHTTPDSL(t, responseContractServerDSL)
	services := CreateHTTPServices(root)
	file := findFileWithSuffix(t, ServerFiles("gen", services), filepath.Join("server", "server.go"))
	sections := file.Section("server-response-contract")
	require.Len(t, sections, 1)

	generated := codegen.SectionCode(t, sections[0])
	require.Contains(t, generated, "// ShowResponseContractCases returns the declared HTTP wire-response contracts")
	require.Contains(t, generated, "func ShowResponseContractCases() []loomhttp.ResponseContractCase")
	require.Contains(t, generated, `ID: "widgets.show.success.202.outcome=accepted"`)
	require.Contains(t, generated, "Kind: loomhttp.ResponseContractSuccess")
	require.Contains(t, generated, "StatusCode: 202")
	require.Contains(t, generated, `ContentTypes: []string{"application/json"}`)
	require.Contains(t, generated, `RequiredHeaders: []string{"X-Version"}`)
	require.Contains(t, generated, `RequiredCookies: []string{"widget_session"}`)
	require.Contains(t, generated, `ID: "widgets.show.error.not_found.404"`)
	require.Contains(t, generated, "Kind: loomhttp.ResponseContractError")
	require.Contains(t, generated, `ErrorName: "not_found"`)
}

func TestServerResponseContractCasesOmitUnsupportedEndpoints(t *testing.T) {
	root := RunHTTPDSL(t, testdata.StreamingResultDSL)
	services := CreateHTTPServices(root)
	files := ServerFiles("gen", services)

	for _, file := range files {
		require.Empty(t, file.Section("server-response-contract"))
	}
}

func TestServerResponseContractCasesCompile(t *testing.T) {
	const modulePath = "example.com/responsecontractcompile"
	root := RunHTTPDSL(t, responseContractServerDSL)
	dir := t.TempDir()
	repoRoot, err := loomsource.RepositoryRoot(".")
	require.NoError(t, err)
	t.Setenv("LOOM_DIR", repoRoot)
	renderHTTPModule(t, dir, modulePath, root)

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

func responseContractServerDSL() {
	Service("widgets", func() {
		Method("show", func() {
			Result(func() {
				Attribute("body", String)
				Attribute("outcome", String)
				Attribute("version", String)
				Attribute("session", String)
				Required("body", "outcome", "version", "session")
			})
			Error("not_found", func() {
				Attribute("reason", String)
				Required("reason")
			})
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
					Header("reason:X-Reason")
				})
			})
		})
	})
}
