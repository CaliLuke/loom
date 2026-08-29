package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

func TestJSONRPCGeneratedTypedAndRawErrorsRoundTrip(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonRPCTypedErrorRoundTripDSL)
	dir := t.TempDir()
	const modulePath = "example.com/jsonrpcerrorroundtrip"
	renderJSONRPCModule(t, dir, modulePath, root)

	testFile := filepath.Join(dir, "error_roundtrip_test.go")
	require.NoError(t, os.WriteFile(testFile, []byte(jsonRPCTypedErrorRoundTripTestSource), 0o644))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "./...")
}

func jsonRPCTypedErrorRoundTripDSL() {
	toolError := dsl.Type("ToolError", func() {
		dsl.ErrorName("name", dsl.String)
		dsl.Attribute("code", dsl.String)
		dsl.Attribute("message", dsl.String)
		dsl.Attribute("remedy_code", dsl.String)
		dsl.Attribute("retry_hint", dsl.String)
		dsl.Required("name", "code", "message", "remedy_code", "retry_hint")
	})
	dsl.API("error-round-trip", func() {
		dsl.Error("forbidden")
		dsl.HTTP(func() {
			dsl.Response("forbidden", dsl.StatusForbidden)
		})
	})
	dsl.Service("tools", func() {
		dsl.Error("forbidden", toolError, func() {
			dsl.Remedy(func() {
				dsl.RemedyCode("tools.request.change")
				dsl.SafeMessage("Request cannot be completed.")
				dsl.RetryHint("Change the request and retry.")
			})
		})
		dsl.JSONRPC(func() {
			dsl.POST("/tools")
		})
		dsl.Method("typed", func() {
			dsl.Payload(func() {
				dsl.ID("id", dsl.String)
				dsl.Attribute("kind", dsl.String)
				dsl.Required("id", "kind")
			})
			dsl.Result(dsl.String)
			dsl.Error("forbidden", toolError)
			dsl.Error("denied", toolError)
			dsl.JSONRPC(func() {
				dsl.Response("forbidden", 4403)
				dsl.Response("denied", 4403)
			})
		})
		dsl.Method("raw", func() {
			dsl.Payload(func() {
				dsl.ID("id", dsl.String)
				dsl.Required("id")
			})
			dsl.Result(dsl.String)
			dsl.JSONRPC(func() {})
		})
	})
}

const jsonRPCTypedErrorRoundTripTestSource = `package integration_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tools "example.com/jsonrpcerrorroundtrip/gen/tools"
	toolsclient "example.com/jsonrpcerrorroundtrip/gen/jsonrpc/tools/client"
	toolsserver "example.com/jsonrpcerrorroundtrip/gen/jsonrpc/tools/server"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/jsonrpc"
)

type toolsService struct{}

func (*toolsService) Typed(_ context.Context, payload *tools.TypedPayload) (string, error) {
	return "", &tools.ToolError{
		Name:       payload.Kind,
		Code:       "TOOLS_" + strings.ToUpper(payload.Kind),
		Message:    "instance message for " + payload.Kind,
		RemedyCode: "tools." + payload.Kind + ".change",
		RetryHint:  "retry " + payload.Kind + " after changing the request",
	}
}

func (*toolsService) Raw(context.Context, *tools.RawPayload) (string, error) {
	return "", &tools.ToolError{
		Name:       "forbidden",
		Code:       "TOOLS_FORBIDDEN",
		Message:    "private implementation detail",
		RemedyCode: "tools.request.change",
		RetryHint:  "Change the request and retry.",
	}
}

func TestGeneratedErrorsRoundTrip(t *testing.T) {
	mux := loomhttp.NewMuxer()
	server := toolsserver.New(
		tools.NewEndpoints(&toolsService{}),
		mux,
		loomhttp.RequestDecoder,
		loomhttp.ResponseEncoder,
		func(_ context.Context, _ http.ResponseWriter, err error) {
			t.Errorf("server error: %v", err)
		},
	)
	toolsserver.Mount(mux, server)
	httpServer := httptest.NewServer(mux)
	defer httpServer.Close()

	client := toolsclient.NewClient(
		"http",
		strings.TrimPrefix(httpServer.URL, "http://"),
		httpServer.Client(),
		loomhttp.RequestEncoder,
		loomhttp.ResponseDecoder,
		false,
	)

	for _, name := range []string{"forbidden", "denied"} {
		_, err := client.Typed()(context.Background(), &tools.TypedPayload{ID: name, Kind: name})
		var typed *tools.ToolError
		if !errors.As(err, &typed) {
			t.Fatalf("typed %s error: got %T %v", name, err, err)
		}
		if typed.Name != name || typed.Code != "TOOLS_"+strings.ToUpper(name) {
			t.Errorf("typed %s discriminator: %#v", name, typed)
		}
		if typed.Message != "instance message for "+name {
			t.Errorf("typed %s message: %q", name, typed.Message)
		}
		if typed.RemedyCode != "tools."+name+".change" {
			t.Errorf("typed %s remedy code: %q", name, typed.RemedyCode)
		}
		if typed.RetryHint != "retry "+name+" after changing the request" {
			t.Errorf("typed %s retry hint: %q", name, typed.RetryHint)
		}
	}

	_, err := client.Raw()(context.Background(), &tools.RawPayload{ID: "raw"})
	var raw *jsonrpc.RawErrorResponse
	if !errors.As(err, &raw) {
		t.Fatalf("raw error: got %T %v", err, err)
	}
	if raw.Code != int(jsonrpc.InternalError) || raw.Message != "Request cannot be completed." {
		t.Errorf("raw envelope: %#v", raw)
	}
	var data jsonrpc.ErrorData
	if err := json.Unmarshal(raw.Data, &data); err != nil {
		t.Fatalf("decode raw data: %v", err)
	}
	if data.Name != "forbidden" {
		t.Errorf("raw data identity: %#v", data)
	}
	if data.Remedy == nil || data.Remedy.Code != "tools.request.change" || data.Remedy.RetryHint != "Change the request and retry." {
		t.Errorf("raw data remedy: %#v", data.Remedy)
	}
}
`
