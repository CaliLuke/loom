package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestLocalizedResultFieldsGenerated(t *testing.T) {
	root := RunHTTPDSL(t, testdata.LocalizedResultFieldsDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/localized", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "localized_test.go"), []byte(localizedResultHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
}

const localizedResultHarness = `package localized_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	client "example.com/localized/gen/http/localized/client"
	server "example.com/localized/gen/http/localized/server"
	svc "example.com/localized/gen/localized"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

type implementation struct {
	calls int
}

func (s *implementation) Echo(_ context.Context, value *svc.LocalizedHolder) (*svc.LocalizedHolder, error) {
	s.calls++
	return value, nil
}
func TestFieldsSurviveTransport(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"complete", "{\"custom\":{\"id\":\"c\",\"name\":\"custom\"},\"plain\":{\"id\":\"p\",\"name\":\"plain\"},\"items\":[{\"id\":\"a\",\"name\":\"array\"}],\"by_key\":{\"k\":{\"id\":\"m\",\"name\":\"map\"}}}", 200},
		{"missing required member", "{\"custom\":{\"name\":\"invalid\"}}", 400},
		{"absent optional fields", "{}", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			implementation := &implementation{}
			mux := loomhttp.NewMuxer()
			server.Mount(mux, server.New(svc.NewEndpoints(implementation), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
			request := httptest.NewRequest("POST", "/localized", strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			require.Equal(t, tc.status, response.Code, response.Body.String())
			if tc.status != 200 {
				require.Zero(t, implementation.calls)
				return
			}
			require.Equal(t, 1, implementation.calls)
			decoded, err := client.DecodeEchoResponse(loomhttp.ResponseDecoder, false)(response.Result())
			require.NoError(t, err)
			value := decoded.(*svc.LocalizedHolder)
			if tc.name == "absent optional fields" {
				require.Nil(t, value.Custom)
				return
			}
			require.Equal(t, "c", value.Custom.ID)
			require.Equal(t, "custom", *value.Custom.Name)
			require.Equal(t, "plain", *value.Plain.Name)
			require.Equal(t, "array", *value.Items[0].Name)
			require.Equal(t, "map", *value.ByKey["k"].Name)
		})
	}
}
`
