package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

const defaultBodyHarness = `package defaults_test

import (
	"context"
	"encoding/json/v2"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	svc "example.com/defaults/gen/defaults"
	cli "example.com/defaults/gen/http/cli/test_api"
	server "example.com/defaults/gen/http/defaults/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

func TestDefaultBodies(t *testing.T) {
	var calls atomic.Int32
	var received any
	echo := func(_ context.Context, payload any) (any, error) {
		received = reflect.ValueOf(payload).Elem().FieldByName("Value").Interface()
		calls.Add(1)
		return nil, nil
	}
	endpoints := new(svc.Endpoints)
	for index := range reflect.ValueOf(endpoints).Elem().NumField() {
		reflect.ValueOf(endpoints).Elem().Field(index).Set(reflect.ValueOf(loom.Endpoint(echo)))
	}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	host := httptest.NewServer(mux)
	defer host.Close()
	oldFlags := flag.CommandLine
	t.Cleanup(func() {
		flag.CommandLine = oldFlags
	})
	flag.CommandLine = flag.NewFlagSet("default-body", flag.ContinueOnError)
	require.NoError(t, flag.CommandLine.Parse([]string{"defaults", "path"}))
	_, _, err := cli.ParseEndpoint("http", "localhost", http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	require.Error(t, err, "a path parameter flag remains required even when its server-side field has a default")
	for _, tc := range []struct {
		name, expected, zero, flagZero string
	}{
		{"text", "\"fallback\"", "\"\"", ""},
		{"empty", "\"\"", "\"\"", ""},
		{"number", "7", "0", "0"},
		{"zero", "0", "0", "0"},
		{"enabled", "true", "false", "false"},
		{"disabled", "false", "false", "false"},
		{"precise", "1.25", "0", "0"},
		{"bytes", "\"AP8=\"", "\"\"", ""},
		{"alias", "\"inherited\"", "\"\"", "\"\""},
		{"override", "\"override\"", "\"\"", "\"\""},
	} {
		for _, required := range []bool{false, true} {
			name := tc.name
			if required {
				name += "_required"
			}
			t.Run(name, func(t *testing.T) {
				for _, input := range []struct {
					name, body, expected string
					valid                bool
				}{
					{"absent", "", tc.expected, !required},
					{"whitespace", " \n\t", tc.expected, !required},
					{"explicit", tc.expected, tc.expected, true},
					{"zero", tc.zero, tc.zero, true},
					{"null", "null", "", false},
					{"malformed", "{x}", "", false},
					{"truncated", "[", "", false},
				} {
					t.Run(input.name, func(t *testing.T) {
						before := calls.Load()
						response, err := http.Post(host.URL+"/"+name, "application/json", strings.NewReader(input.body))
						require.NoError(t, err)
						body, err := io.ReadAll(response.Body)
						require.NoError(t, err)
						require.NoError(t, response.Body.Close())
						if input.valid {
							require.Equal(t, http.StatusNoContent, response.StatusCode, string(body))
							require.Equal(t, before+1, calls.Load())
							value, err := json.Marshal(received)
							require.NoError(t, err)
							require.JSONEq(t, input.expected, string(value))
						} else {
							require.Equal(t, http.StatusBadRequest, response.StatusCode, string(body))
							require.Equal(t, before, calls.Load())
							if input.body == "" || input.name == "whitespace" {
								require.Contains(t, string(body), "missing_payload")
							} else {
								require.Contains(t, string(body), "decode_payload")
							}
						}
					})
				}
				for _, explicit := range []bool{false, true} {
					args := []string{"defaults", strings.ReplaceAll(name, "_", "-")}
					expected := tc.expected
					if explicit {
						args = append(args, "--body="+tc.flagZero)
						expected = tc.zero
					}
					flag.CommandLine = flag.NewFlagSet("default-body", flag.ContinueOnError)
					require.NoError(t, flag.CommandLine.Parse(args))
					_, payload, err := cli.ParseEndpoint("http", "localhost", http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
					if required && !explicit {
						require.Error(t, err)
						continue
					}
					require.NoError(t, err)
					value, err := json.Marshal(reflect.ValueOf(payload).Elem().FieldByName("Value").Interface())
					require.NoError(t, err)
					require.JSONEq(t, expected, string(value))
				}
				if tc.name != "text" && tc.name != "empty" && tc.name != "bytes" {
					flag.CommandLine = flag.NewFlagSet("default-body", flag.ContinueOnError)
					require.NoError(t, flag.CommandLine.Parse([]string{"defaults", strings.ReplaceAll(name, "_", "-"), "--body="}))
					_, _, err := cli.ParseEndpoint("http", "localhost", http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
					require.Error(t, err, "an explicitly empty numeric or JSON flag must not be treated as absent")
				}
			})
		}
	}
}
`

func TestSelectedBodyDefaultsGeneratedHTTP(t *testing.T) {
	root := RunHTTPDSL(t, testdata.SelectedBodyDefaultsDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/defaults", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "default_test.go"), []byte(defaultBodyHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "./...")
}
