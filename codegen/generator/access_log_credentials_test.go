package generator

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestAccessLogCredentialExampleWiring(t *testing.T) {
	dir := t.TempDir()
	renderAccessLogExample(t, dir, testdata.AccessLogCredentialsDSL)
	for _, host := range []struct {
		name, file string
		web, rpc   bool
	}{
		{"mixed", "jsonrpc.go", true, true},
		{"webonly", "http.go", true, false},
		{"rpconly", "jsonrpc.go", false, true},
		{"ordinary", "http.go", false, false},
	} {
		source, err := os.ReadFile(filepath.Join(dir, "cmd", host.name, host.file))
		require.NoError(t, err)
		parsed, err := parser.ParseFile(token.NewFileSet(), host.file, source, 0)
		require.NoError(t, err)
		var imports []string
		for _, imp := range parsed.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			require.NoError(t, err)
			if path != "example.com/diagnostics/gen/http/web" && path != "example.com/diagnostics/gen/jsonrpc/rpc" {
				continue
			}
			require.NotNil(t, imp.Name)
			imports = append(imports, imp.Name.Name+" "+imp.Path.Value)
		}
		require.Len(t, imports, boolCount(host.web, host.rpc), host.name)
		var configuration string
		for line := range strings.SplitSeq(string(source), "\n") {
			if strings.Contains(line, "handler = log.HTTP(ctx") {
				require.Empty(t, configuration)
				configuration = strings.TrimSpace(line)
			}
		}
		require.NotEmpty(t, configuration)
		if host.web || host.rpc {
			require.Contains(t, configuration, ".Fields()")
		} else {
			require.Equal(t, "handler = log.HTTP(ctx)(handler)", configuration)
		}
		// Compile and execute the exact emitted middleware configuration with its
		// generated metadata imports. The controlled handler observes untouched URLs.
		harness := fmt.Sprintf(accessLogExampleHarness, strings.Join(imports, "\n"), configuration, host.web, host.web, host.rpc)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "cmd", host.name, "access_test.go"), []byte(harness), 0o600))
	}
	for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}, {"test", "-race", "./cmd/..."}} {
		logs, err := testingx.RunCmd(dir, "go", args...)
		require.NoError(t, err, logs)
	}
}

func boolCount(values ...bool) int {
	count := 0
	for _, value := range values {
		if value {
			count++
		}
	}
	return count
}

const accessLogExampleHarness = `package main
import (
 "bytes"
 "context"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
 "github.com/CaliLuke/loom/clue/log"
 %s
)
func TestAccessURL(t *testing.T) {
 var output bytes.Buffer
 ctx := log.Context(context.Background(), log.WithOutput(&output))
 original := "http://example.test/private-path?ticket=private-web&proof=private-rpc&key=private-key&ordinary=visible"
 req := httptest.NewRequest(http.MethodGet, original, nil)
 calls := 0
 var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  calls++
  if r.URL.String() != original {
   t.Errorf("wire URL changed: %%s", r.URL)
  }
  w.WriteHeader(http.StatusNoContent)
 })
 %s
 handler.ServeHTTP(httptest.NewRecorder(), req)
 if calls != 1 || req.URL.String() != original {
  t.Errorf("request changed or invocation lost: %%d %%s", calls, req.URL)
 }
 text := output.String()
 for _, marker := range []string{"start", "end", "ordinary=visible"} {
  if !strings.Contains(text, marker) {
   t.Errorf("missing %%q in %%s", marker, text)
  }
 }
 for _, item := range []struct {
  secret string
  masked bool
 }{{"private-path", %t}, {"private-web", %t}, {"private-rpc", %t}, {"private-key", true}} {
  if strings.Contains(text, item.secret) == item.masked {
   t.Errorf("incorrect mask for %%s (masked=%%t): %%s", item.secret, item.masked, text)
  }
 }
}
`
