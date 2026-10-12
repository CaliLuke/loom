package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

func TestGeneratedErrorDisclosure(t *testing.T) {
	root := RunHTTPDSL(t, httpErrorDisclosureDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/disclosure", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "disclosure_test.go"), []byte(httpErrorDisclosureHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-count=1", "./...")
}

func httpErrorDisclosureDSL() {
	dsl.Service("disclosure", func() {
		dsl.Method("show", func() {
			dsl.Result(dsl.String)
			dsl.HTTP(func() {
				dsl.GET("/show")
			})
		})
	})
}

const httpErrorDisclosureHarness = `package disclosure_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	svc "example.com/disclosure/gen/disclosure"
	server "example.com/disclosure/gen/http/disclosure/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

func TestPublicMessages(t *testing.T) {
	secret := errors.New("private database password")
	public := loom.NewServiceError(secret, "public", false, false, true)
	public.Message = "public detail"
	safe := loom.WithErrorRemedy(loom.NewServiceError(secret, "public", false, false, true), &loom.ErrorRemedy{SafeMessage: "safe detail"})
	for _, tc := range []struct {
		name    string
		err     error
		message string
	}{
		{"plain", secret, "internal server error"},
		{"wrapped", fmt.Errorf("private query: %w", secret), "internal server error"},
		{"public wrapper", fmt.Errorf("private query: %w", public), "public detail"},
		{"safe", safe, "safe detail"},
		{"join", errors.Join(public, secret), "internal server error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := tc.err.Error()
			calls := 0
			endpoints := &svc.Endpoints{Show: func(context.Context, any) (any, error) {
				calls++
				return nil, tc.err
			}}
			mux := loomhttp.NewMuxer()
			server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, func(_ context.Context, _ http.ResponseWriter, err error) {
				t.Errorf("encode: %v", err)
			}, nil))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/show", nil))
			require.Equal(t, 1, calls)
			require.Equal(t, http.StatusInternalServerError, rec.Code)
			require.NotContains(t, rec.Body.String(), "private")
			var problem loomhttp.ProblemResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &problem))
			require.Equal(t, tc.message, problem.Detail)
			require.NotEmpty(t, problem.Instance)
			require.Equal(t, before, tc.err.Error())
			require.ErrorIs(t, tc.err, secret)
		})
	}
}
`
