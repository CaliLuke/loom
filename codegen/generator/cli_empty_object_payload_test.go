package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/loomsource"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestCLIEmptyObjectPayloadCompile generates the service, transport, and
// example output for HTTP and JSON-RPC methods whose payload is an object user
// type without attributes, builds and vets the module, and runs the generated
// command parsers. Each parser returns the empty payload that the service
// method takes, without a payload flag.
func TestCLIEmptyObjectPayloadCompile(t *testing.T) {
	repoRoot, err := loomsource.RepositoryRoot(".")
	require.NoError(t, err)
	source, err := loomsource.Resolve(repoRoot, filepath.Join(t.TempDir(), "loom-pinned"))
	require.NoError(t, err)

	codegen.RunDSL(t, emptyObjectPayloadDesign)
	dir := t.TempDir()
	goMod := fmt.Sprintf("module example.com/emptycli\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", source)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))

	_, err = Generate(dir, "gen", false)
	require.NoError(t, err)
	_, err = Generate(dir, "example", false)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "empty_cli_test.go"), []byte(emptyObjectPayloadHarness), 0o600))

	_, err = testingx.RunCmd(dir, "go", "mod", "tidy")
	require.NoError(t, err)
	output, err := testingx.RunCmd(dir, "go", "build", "./...")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(dir, "go", "vet", "./...")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(dir, "go", "test", "-count=1", ".")
	require.NoError(t, err, output)
}

// emptyObjectPayloadDesign declares the empty object type E and the payload E
// of an HTTP method, of an HTTP method that streams its request body, and of a
// JSON-RPC method.
func emptyObjectPayloadDesign() {
	dsl.API("emptycli", func() {
		dsl.JSONRPC(func() {})
	})
	e := dsl.Type("E", func() {})
	dsl.Service("web", func() {
		dsl.Method("create", func() {
			dsl.Payload(e)
			dsl.HTTP(func() {
				dsl.POST("/")
			})
		})
		dsl.Method("upload", func() {
			dsl.Payload(e)
			dsl.HTTP(func() {
				dsl.POST("/upload")
				dsl.SkipRequestBodyEncodeDecode()
			})
		})
	})
	dsl.Service("rpc", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("create", func() {
			dsl.Payload(e)
			dsl.JSONRPC(func() {})
		})
	})
}

const emptyObjectPayloadHarness = `package emptycli

import (
	"net/http"
	"os"
	"testing"

	httpcli "example.com/emptycli/gen/http/cli/emptycli"
	jsonrpccli "example.com/emptycli/gen/jsonrpc/cli/emptycli"
	"example.com/emptycli/gen/rpc"
	"example.com/emptycli/gen/web"
	loomhttp "github.com/CaliLuke/loom/http"
)

type parser func(string, string, loomhttp.Doer, func(*http.Request) loomhttp.Encoder, func(*http.Response) loomhttp.Decoder, bool) (any, error)

func parse(t *testing.T, p parser, args ...string) any {
	t.Helper()
	os.Args = append([]string{"cli"}, args...)
	data, err := p("http", "localhost", http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	if err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	return data
}

func httpParser(scheme, host string, doer loomhttp.Doer, enc func(*http.Request) loomhttp.Encoder, dec func(*http.Response) loomhttp.Decoder, restore bool) (any, error) {
	_, data, err := httpcli.ParseEndpoint(scheme, host, doer, enc, dec, restore)
	return data, err
}

func jsonrpcParser(scheme, host string, doer loomhttp.Doer, enc func(*http.Request) loomhttp.Encoder, dec func(*http.Response) loomhttp.Decoder, restore bool) (any, error) {
	_, data, err := jsonrpccli.ParseEndpoint(scheme, host, doer, enc, dec, restore)
	return data, err
}

func TestHTTPEmptyPayload(t *testing.T) {
	if p, ok := parse(t, httpParser, "web", "create").(*web.E); !ok || p == nil {
		t.Errorf("web create: got %T, want a non-nil *web.E", p)
	}
}

func TestHTTPEmptyPayloadStream(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "body")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	data := parse(t, httpParser, "web", "upload", "--stream", f.Name())
	req, ok := data.(*web.UploadRequestData)
	if !ok {
		t.Fatalf("web upload: got %T, want *web.UploadRequestData", data)
	}
	if err := req.Body.Close(); err != nil {
		t.Error(err)
	}
	if req.Payload == nil {
		t.Errorf("web upload: nil payload")
	}
}

func TestJSONRPCEmptyPayload(t *testing.T) {
	if p, ok := parse(t, jsonrpcParser, "rpc", "create").(*rpc.E); !ok || p == nil {
		t.Errorf("rpc create: got %T, want a non-nil *rpc.E", p)
	}
}
`
