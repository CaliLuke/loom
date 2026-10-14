package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

const jsonRPCSelectedObjectErrorHarness = `package selectederrors_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	client "example.com/selectederrors/gen/jsonrpc/tools/client"
	tools "example.com/selectederrors/gen/tools"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/jsonrpc"
)

func TestSelectedObjectError(t *testing.T) {
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request jsonrpc.RawRequest
		if err := json.UnmarshalRead(r.Body, &request); err != nil {
			t.Error(err)
			return
		}
		var name string
		if err := json.Unmarshal(request.Params, &name); err != nil {
			t.Error(err)
			return
		}
		code := 4403
		if name == "denied" {
			code = 4404
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Error-Kind", name)
		response := jsonrpc.MakeErrorResponse(request.ID, jsonrpc.Code(code), "safe", map[string]string{"kind": "ordinary"})
		if err := loomhttp.ResponseEncoder(r.Context(), w).Encode(response); err != nil {
			t.Error(err)
		}
	}))
	defer host.Close()
	c := client.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	for _, name := range []string{"forbidden", "denied"} {
		_, err := c.Call()(context.Background(), name)
		var typed *tools.Failure
		if !errors.As(err, &typed) || typed.Kind != name || typed.Details == nil || typed.Details.Kind != "ordinary" {
			t.Errorf("selected object must not borrow outer ErrorName provenance: %T %v", err, err)
		}
	}
}
`

const jsonRPCSelectedErrorHarness = `package selectederrors_test

import (
	"context"
	"errors"
	client "example.com/selectederrors/gen/jsonrpc/tools/client"
	server "example.com/selectederrors/gen/jsonrpc/tools/server"
	tools "example.com/selectederrors/gen/tools"
	loomhttp "github.com/CaliLuke/loom/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type service struct{}

func TestSelectedErrors(t *testing.T) {
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(tools.NewEndpoints(service{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
	host := httptest.NewServer(mux)
	defer host.Close()
	c := client.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	for _, name := range []string{"forbidden", "denied"} {
		_, err := c.Call()(context.Background(), name)
		var typed *tools.Failure
		if !errors.As(err, &typed) {
			t.Errorf("%s typed error: %T %v", name, err, err)
		} else if typed.Kind != name {
			t.Errorf("got discriminator %q, want %q", typed.Kind, name)
		}
	}
}

func (service) Call(_ context.Context, name string) (string, error) {
	return "", &tools.Failure{Kind: name}
}
`

// TestJSONRPCSelectedErrorDiscriminator retains ErrorName in the selected body.
func TestJSONRPCSelectedErrorDiscriminator(t *testing.T) {
	for _, selected := range []string{"root", "member", "other object"} {
		t.Run(selected, func(t *testing.T) {
			root := RunJSONRPCDSL(t, func() {
				details := dsl.Type("Details", func() {
					dsl.Attribute("kind", dsl.String)
					dsl.Required("kind")
				})
				failure := dsl.Type("Failure", func() {
					dsl.ErrorName("kind:error_kind", dsl.String)
					dsl.Required("kind")
					if selected == "other object" {
						dsl.Attribute("details", details)
						dsl.Required("details")
					}
				})
				dsl.Service("tools", func() {
					dsl.JSONRPC(func() {
						dsl.POST("/rpc")
					})
					dsl.Method("call", func() {
						dsl.Payload(dsl.String)
						dsl.Result(dsl.String)
						dsl.Error("forbidden", failure)
						dsl.Error("denied", failure)
						dsl.JSONRPC(func() {
							for _, name := range []string{"forbidden", "denied"} {
								code := 4403
								if selected == "other object" && name == "denied" {
									code = 4404
								}
								dsl.Response(name, code, func() {
									switch selected {
									case "other object":
										dsl.Header("kind:X-Error-Kind")
										dsl.Body("details")
									case "root":
										dsl.Body("kind")
									default:
										dsl.Body(func() {
											dsl.Attribute("kind")
										})
									}
								})
							}
						})
					})
				})
			})
			dir := t.TempDir()
			renderJSONRPCModule(t, dir, "example.com/selectederrors", root)
			harness := jsonRPCSelectedErrorHarness
			if selected == "other object" {
				harness = jsonRPCSelectedObjectErrorHarness
			}
			require.NoError(t, os.WriteFile(filepath.Join(dir, "error_test.go"), []byte(harness), 0o600))
			runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
			runGoJSONRPCTestCommand(t, dir, "test", "-count=1", "./...")
		})
	}
}
