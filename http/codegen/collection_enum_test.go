package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

const collectionEnumHarness = `package enums_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	svc "example.com/enums/gen/enums"
	client "example.com/enums/gen/http/enums/client"
	server "example.com/enums/gen/http/enums/server"
	pathclient "example.com/enums/gen/http/service_path_array_string_validate/client"
	pathserver "example.com/enums/gen/http/service_path_array_string_validate/server"
	pathsvc "example.com/enums/gen/service_path_array_string_validate"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

func TestCollectionEnums(t *testing.T) {
	var calls atomic.Int32
	echo := func(_ context.Context, value any) (any, error) {
		calls.Add(1)
		return value, nil
	}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(&svc.Endpoints{Records: echo, Precise: echo, Labels: echo, Bytes: echo, Nested: echo, Optional: echo}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	pathserver.Mount(mux, pathserver.New(&pathsvc.Endpoints{MethodPathArrayStringValidate: echo}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	host := httptest.NewServer(mux)
	defer host.Close()
	for _, tc := range []struct {
		path, body string
		valid      bool
	}{
		{"/records", "[{\"rating\":1.2345679,\"data\":\"YWI=\"}]", true},
		{"/records", "[{\"rating\":1.2345677,\"data\":\"YWI=\"}]", false},
		{"/precise", "[1.23456789]", true},
		{"/precise", "[1.2345677]", false},
		{"/labels", "[\"val\"]", true},
		{"/labels", "[\"a\",\"b\"]", true},
		{"/labels", "[]", false},
		{"/labels", "[\"b\",\"a\"]", false},
		{"/labels", "[\"val\",\"val\"]", false},
		{"/bytes", "\"AH8=\"", true},
		{"/bytes", "\"fwA=\"", false},
		{"/nested", "[[\"val\"]]", true},
		{"/nested", "[[\"bad\"]]", false},
		{"/optional", "{}", true},
		{"/optional", "{\"labels\":[\"val\"]}", true},
		{"/optional", "{\"labels\":[]}", false},
	} {
		t.Run(tc.path+tc.body, func(t *testing.T) {
			before := calls.Load()
			response, err := http.Post(host.URL+tc.path, "application/json", strings.NewReader(tc.body))
			require.NoError(t, err)
			body, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			if tc.valid {
				require.Less(t, response.StatusCode, 300, string(body))
				require.Equal(t, before+1, calls.Load())
				if tc.path != "/optional" {
					expected := tc.body
					if tc.path == "/precise" {
						expected = "[1.2345679]"
					}
					require.JSONEq(t, expected, string(body))
				}
			} else {
				require.Equal(t, 400, response.StatusCode, string(body))
				require.Contains(t, string(body), "\"code\":\"invalid_enum_value\"")
				require.Equal(t, before, calls.Load())
			}
		})
	}
	address, err := url.Parse(host.URL)
	require.NoError(t, err)
	c := client.NewClient(address.Scheme, address.Host, http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	result, err := c.Labels()(context.Background(), svc.LabelList{"val"})
	require.NoError(t, err)
	require.Equal(t, svc.LabelList{"val"}, result)
	result, err = c.Precise()(context.Background(), svc.PreciseValues{1.23456789})
	require.NoError(t, err)
	require.Equal(t, svc.PreciseValues{1.23456789}, result)
	for _, tc := range []struct {
		raw   string
		valid bool
	}{
		{"[\"val\"]", true}, {"[]", false}, {"[\"val\",\"val\"]", false}, {"[\"bad\"]", false},
	} {
		payload, err := pathclient.BuildMethodPathArrayStringValidatePayload(tc.raw)
		if tc.valid {
			require.NoError(t, err)
			require.Equal(t, []string{"val"}, payload.P)
		} else {
			require.Error(t, err)
		}
	}
	for _, tc := range []struct {
		path  string
		valid bool
	}{{"/val", true}, {"/bad", false}, {"/val,val", false}} {
		before := calls.Load()
		response, err := http.Get(host.URL + tc.path)
		require.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())
		if tc.valid {
			require.Less(t, response.StatusCode, 300, string(body))
			require.Equal(t, before+1, calls.Load())
		} else {
			require.Equal(t, 400, response.StatusCode)
			require.Contains(t, string(body), "invalid_enum_value")
			require.Equal(t, before, calls.Load())
		}
	}
	// Client response validation must reject a well-formed disallowed array.
	badHost := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, "[\"bad\"]"); err != nil {
			t.Error(err)
		}
	}))
	defer badHost.Close()
	badAddress, err := url.Parse(badHost.URL)
	require.NoError(t, err)
	badClient := client.NewClient(badAddress.Scheme, badAddress.Host, http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	_, err = badClient.Labels()(context.Background(), svc.LabelList{"val"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid value for \"body\"")
}
`

func TestCollectionEnumGeneratedHTTP(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		testdata.PayloadPathArrayStringValidateDSL()
		testdata.CollectionEnumDSL()
	})
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/enums", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "enum_test.go"), []byte(collectionEnumHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "./...")
}
