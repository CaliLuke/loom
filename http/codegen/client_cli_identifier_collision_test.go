package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/service"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestAggregateClientCLIIdentifiersDoNotCollide(t *testing.T) {
	const modulePath = "example.com/cli-identifiers"
	tests := []struct {
		name        string
		design      func()
		expressions func(*expr.RootExpr) *expr.HTTPExpr
		transport   ClientCLITransport
	}{
		{
			name:   "http",
			design: aggregateHTTPClientCLIIdentifierDSL,
			expressions: func(root *expr.RootExpr) *expr.HTTPExpr {
				return root.API.HTTP
			},
			transport: httpClientCLITransport(),
		},
		{
			name:   "jsonrpc",
			design: aggregateJSONRPCClientCLIIdentifierDSL,
			expressions: func(root *expr.RootExpr) *expr.HTTPExpr {
				return &root.API.JSONRPC.HTTPExpr
			},
			transport: ClientCLITransport{PathName: "jsonrpc", DisplayName: "JSON-RPC"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := RunHTTPDSL(t, test.design)
			data := NewServicesData(service.NewServicesData(root), test.expressions(root))
			files := ClientCLIFilesForTransport(modulePath+"/gen", data, test.transport)
			parser := findFileWithSection(t, files, "parse-endpoint")
			dir := t.TempDir()
			_, err := parser.Render(dir)
			require.NoError(t, err)
			source, err := os.ReadFile(filepath.Join(dir, parser.Path))
			require.NoError(t, err)
			code := string(source)

			assert.NotContains(t, code, "\n\tenc \""+modulePath+"/gen/"+test.transport.PathName+"/en/client\"")
			assert.Equal(t, 1, strings.Count(code, "func ckRawUsage("), code)
		})
	}
}

func TestAggregateClientCLIIdentifiersAreIsolatedPerServer(t *testing.T) {
	const modulePath = "example.com/cli-server-identifiers"
	root := RunHTTPDSL(t, aggregateHTTPClientCLIServerIsolationDSL)
	files := ClientCLIFiles(modulePath+"/gen", CreateHTTPServices(root))
	dir := t.TempDir()
	for _, file := range files {
		_, err := file.Render(dir)
		require.NoError(t, err)
	}

	withCompetitor, err := os.ReadFile(filepath.Join(dir, "gen", "http", "cli", "with_competitor", "cli.go"))
	require.NoError(t, err)
	withoutCompetitor, err := os.ReadFile(filepath.Join(dir, "gen", "http", "cli", "without_competitor", "cli.go"))
	require.NoError(t, err)

	assert.Contains(t, string(withCompetitor), `enc3 "`+modulePath+`/gen/http/en/client"`)
	assert.Contains(t, string(withCompetitor), "enc3.NewClient(")
	assert.Contains(t, string(withoutCompetitor), `enc2 "`+modulePath+`/gen/http/en/client"`)
	assert.Contains(t, string(withoutCompetitor), "enc2.NewClient(")
}

func aggregateHTTPClientCLIIdentifierDSL() {
	API("cli-identifiers", func() {
		Server("api", func() {
			Services("en", "ckRaw", "ck")
			Host("development", func() {
				URI("http://localhost:8080")
			})
		})
	})
	for _, serviceName := range []string{"en", "ckRaw", "ck"} {
		Service(serviceName, func() {
			methodName := "show"
			if serviceName == "ck" {
				methodName = "raw"
			}
			Method(methodName, func() {
				HTTP(func() {
					GET("/" + serviceName)
				})
			})
		})
	}
}

func aggregateJSONRPCClientCLIIdentifierDSL() {
	API("cli-identifiers", func() {
		JSONRPC(func() {})
	})
	for _, serviceName := range []string{"en", "ckRaw", "ck"} {
		Service(serviceName, func() {
			JSONRPC(func() {
				POST("/rpc/" + serviceName)
			})
			methodName := "show"
			if serviceName == "ck" {
				methodName = "raw"
			}
			Method(methodName, func() {
				JSONRPC(func() {})
			})
		})
	}
}

func aggregateHTTPClientCLIServerIsolationDSL() {
	audit := Interceptor("audit")
	API("cli-server-identifiers", func() {
		Server("with competitor", func() {
			Services("en", "enc2")
			Host("development", func() {
				URI("http://localhost:8080")
			})
		})
		Server("without competitor", func() {
			Services("en")
			Host("development", func() {
				URI("http://localhost:8081")
			})
		})
	})
	Service("en", func() {
		Method("show", func() {
			HTTP(func() {
				GET("/en")
			})
		})
	})
	Service("enc2", func() {
		ClientInterceptor(audit)
		Method("show", func() {
			HTTP(func() {
				GET("/enc2")
			})
		})
	})
}
