package codegen

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
)

func TestGeneratedExtensionMethodClientServerIntegration(t *testing.T) {
	const modulePath = "example.com/extensionmethodsit"

	root := RunHTTPDSL(t, extensionMethodsDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)

	if err := os.WriteFile(filepath.Join(dir, "extension_methods_test.go"), []byte(extensionMethodsHarness), 0o600); err != nil {
		t.Fatalf("write extension methods harness: %v", err)
	}

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

func extensionMethodsDSL() {
	Service("methods", func() {
		Method("Query", func() {
			Result(String)
			HTTP(func() {
				QUERY("/query")
			})
		})
		Method("Purge", func() {
			Result(String)
			HTTP(func() {
				Route("purge", "/purge")
			})
		})
	})
}

const extensionMethodsHarness = `package extensionmethodsit_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"

	methods "example.com/extensionmethodsit/gen/methods"
	methodsclient "example.com/extensionmethodsit/gen/http/methods/client"
	methodsserver "example.com/extensionmethodsit/gen/http/methods/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

func TestGeneratedExtensionMethods(t *testing.T) {
	endpoints := &methods.Endpoints{
		Query: func(context.Context, any) (any, error) {
			return "query", nil
		},
		Purge: func(context.Context, any) (any, error) {
			return "purge", nil
		},
	}
	var observed []string
	mux := loomhttp.NewMuxer()
	mux.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			observed = append(observed, r.Method)
			next.ServeHTTP(w, r)
		})
	})
	server := methodsserver.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	methodsserver.Mount(mux, server)
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)

	u, err := url.Parse(httpServer.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	client := methodsclient.NewClient(u.Scheme, u.Host, httpServer.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	tests := []struct {
		name   string
		method string
		want   string
		call   loom.Endpoint
	}{
		{name: "QUERY", method: "QUERY", want: "query", call: client.Query()},
		{name: "custom", method: "PURGE", want: "purge", call: client.Purge()},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.call(t.Context(), nil)
			if err != nil {
				t.Fatalf("call generated client: %v", err)
			}
			if got, ok := result.(string); !ok || got != test.want {
				t.Errorf("result = %#v, want %q", result, test.want)
			}
			if !slices.Contains(observed, test.method) {
				t.Errorf("observed methods = %v, want %q", observed, test.method)
			}
		})
	}
}
`
