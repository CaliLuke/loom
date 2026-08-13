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
