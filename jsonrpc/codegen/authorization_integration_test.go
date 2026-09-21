package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestAuthorizationJSONRPCIntegration(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		ref := Type("ResourceRef", func() {
			Attribute("id", String)
			Required("id")
		})
		access := Authorization("resource.edit", ref)
		Service("resources", func() {
			StrictAuthorization()
			JSONRPC(func() {
				POST("/rpc")
			})
			Method("edit", func() {
				Payload(func() {
					Attribute("id", String)
					Required("id")
				})
				Authorize(access, func() {
					Bind("id", "id")
				})
				Result(String)
				Error("forbidden")
				JSONRPC(func() {
					Response("forbidden", 1001)
				})
			})
		})
	})
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/accessjsonrpc", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "authorization_test.go"), []byte(authorizationJSONRPCHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", ".")
}

const authorizationJSONRPCHarness = `package integration

import (
	"context"
	resourcessvr "example.com/accessjsonrpc/gen/jsonrpc/resources/server"
	resources "example.com/accessjsonrpc/gen/resources"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type resourceService struct {
	calls int
}

func (s *resourceService) Edit(_ context.Context, p *resources.EditPayload) (string, error) {
	s.calls++
	return p.ID, nil
}

type accessEvaluator struct {
}

func (*accessEvaluator) AuthorizeResourceEdit(_ context.Context, r *resources.ResourceRef) error {
	if r.ID != "allowed" {
		return loom.PermanentError("forbidden", "Access denied")
	}
	return nil
}
func TestGeneratedJSONRPCAccess(t *testing.T) {
	s := &resourceService{}
	mux := loomhttp.NewMuxer()
	server := resourcessvr.New(resources.NewEndpoints(s, &accessEvaluator{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, func(_ context.Context, _ http.ResponseWriter, err error) {
		t.Log(err)
	})
	resourcessvr.Mount(mux, server)
	for _, tc := range []struct {
		id, callID string
		calls      int
		want       string
	}{
		{"denied", ",\"id\":1", 0, "1001"},
		{"denied", "", 0, ""},
		{"allowed", ",\"id\":2", 1, "allowed"},
	} {
		request := httptest.NewRequest("POST", "/rpc", strings.NewReader("{\"jsonrpc\":\"2.0\",\"method\":\"edit\",\"params\":{\"id\":\""+tc.id+"\"}"+tc.callID+"}"))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		require.Equal(t, tc.calls, s.calls, response.Body.String())
		if tc.want == "" {
			require.Empty(t, response.Body.String())
		} else {
			require.Contains(t, response.Body.String(), tc.want)
		}
	}
}
`
