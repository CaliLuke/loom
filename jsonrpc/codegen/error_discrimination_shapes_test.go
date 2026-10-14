package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

const jsonRPCErrorShapesHarness = `package errorshapes_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	client "example.com/errorshapes/gen/jsonrpc/shapes/client"
	server "example.com/errorshapes/gen/jsonrpc/shapes/server"
	shapes "example.com/errorshapes/gen/shapes"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

type (
	service          struct{}
	candidateDecoder struct {
		base       loomhttp.Decoder
		failString bool
		sentinel   error
	}
)

func TestErrorShapes(t *testing.T) {
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(shapes.NewEndpoints(service{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
	host := httptest.NewServer(mux)
	defer host.Close()
	c := client.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	cases := []struct {
		name     string
		endpoint loom.Endpoint
		input    string
		expected string
	}{
		{"string", c.Disjoint(), "first", "disjoint_first"},
		{"number", c.Disjoint(), "second", "disjoint_second"},
		{"enum alpha", c.Constrained(), "first", "constrained_first"},
		{"enum beta", c.Constrained(), "second", "constrained_second"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.endpoint(context.Background(), tc.input)
			var named interface {
				LoomErrorName() string
			}
			if !errors.As(err, &named) || named.LoomErrorName() != tc.expected {
				t.Errorf("got %T %v, want declared %s", err, err, tc.expected)
			}
		})
	}
	for _, input := range []string{"first", "second"} {
		t.Run("ambiguous/"+input, func(t *testing.T) {
			_, err := c.Ambiguous()(context.Background(), input)
			var invalid *loomhttp.ClientError
			if !errors.As(err, &invalid) || !strings.Contains(invalid.Message, "ambiguous") {
				t.Errorf("want explicit ambiguous response error, got %T %v", err, err)
			}
		})
	}
}

func TestErrorDecoderUnexpectedFailurePreservesCause(t *testing.T) {
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(shapes.NewEndpoints(service{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
	host := httptest.NewServer(mux)
	defer host.Close()
	sentinel := errors.New("application decoder failed")
	for _, input := range []string{"first", "second"} {
		t.Run(input, func(t *testing.T) {
			decoder := func(response *http.Response) loomhttp.Decoder {
				return candidateDecoder{base: loomhttp.ResponseDecoder(response), failString: input == "second", sentinel: sentinel}
			}
			c := client.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, decoder, false)
			_, err := c.Disjoint()(context.Background(), input)
			if !errors.Is(err, sentinel) {
				t.Errorf("unexpected eligible failure must win before or after a successful trial: %T %v", err, err)
			}
		})
	}
}

func (d candidateDecoder) Decode(value any) error {
	_, isString := value.(*string)
	_, isInt := value.(*int)
	if (d.failString && isString) || (!d.failString && isInt) {
		return d.sentinel
	}
	return d.base.Decode(value)
}

func (service) Disjoint(_ context.Context, input string) (string, error) {
	if input == "first" {
		return "", shapes.DisjointFirst("same")
	}
	return "", shapes.DisjointSecond(7)
}

func (service) Constrained(_ context.Context, input string) (string, error) {
	if input == "first" {
		return "", shapes.ConstrainedFirst("alpha")
	}
	return "", shapes.ConstrainedSecond("beta")
}

func (service) Ambiguous(_ context.Context, input string) (string, error) {
	if input == "first" {
		return "", shapes.AmbiguousFirst("same")
	}
	return "", shapes.AmbiguousSecond("same")
}
`

// TestJSONRPCErrorDiscriminatorShapes checks identity without imposing a common key.
func TestJSONRPCErrorDiscriminatorShapes(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		dsl.Service("shapes", func() {
			dsl.JSONRPC(func() {
				dsl.POST("/rpc")
			})
			for _, method := range []string{"disjoint", "constrained", "ambiguous"} {
				dsl.Method(method, func() {
					dsl.Payload(dsl.String)
					dsl.Result(dsl.String)
					dsl.Error(method+"_first", dsl.String, func() {
						if method == "constrained" {
							dsl.Enum("alpha")
						}
					})
					if method == "disjoint" {
						dsl.Error(method+"_second", dsl.Int)
					} else {
						dsl.Error(method+"_second", dsl.String, func() {
							if method == "constrained" {
								dsl.Enum("beta")
							}
						})
					}
					dsl.JSONRPC(func() {
						dsl.Response(method+"_first", 4403)
						dsl.Response(method+"_second", 4403)
					})
				})
			}
		})
	})
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/errorshapes", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "errors_test.go"), []byte(jsonRPCErrorShapesHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "-count=1", "./...")
}
