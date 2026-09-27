package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

const pathLocalNamesHarness = `package client

import (
	"context"
	"io"
	"strings"
	"testing"

	svc "example.com/pathlocals/gen/pathlocals"
)

func TestRequestPaths(t *testing.T) {
	c := &Client{scheme: "https", host: "example.com"}
	ctx := context.Background()
	raw := &svc.RawRequestData{Payload: &svc.RawPayload{Rd: "first", Rd2: "second"}, Body: io.NopCloser(strings.NewReader("raw body"))}
	req, err := c.BuildRawRequest(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := req.URL.String(); got != "https://example.com/raw/first/second" {
		t.Errorf("raw URL = %q", got)
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := req.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if string(body) != "raw body" {
		t.Errorf("raw body = %q", body)
	}
	req, err = c.BuildStreamRequest(ctx, &svc.StreamPayload{Scheme: "first", Scheme2: "second"})
	if err != nil {
		t.Fatal(err)
	}
	if got := req.URL.String(); got != "wss://example.com/stream/first/second" {
		t.Errorf("stream URL = %q", got)
	}
	req, err = c.BuildArrayRequest(ctx, &svc.ArrayPayload{I: []int{7, 9}, I2: "second"})
	if err != nil {
		t.Fatal(err)
	}
	if got := req.URL.String(); got != "https://example.com/array/7,9/second" {
		t.Errorf("array URL = %q", got)
	}
}
`

func TestPathLocalNames(t *testing.T) {
	root := RunHTTPDSL(t, testdata.PathLocalNamesDSL)
	services := CreateHTTPServices(root)
	for _, endpoint := range services.Get("pathlocals").Endpoints {
		t.Run(endpoint.Method.Name, func(t *testing.T) {
			for _, arg := range endpoint.Routes[0].PathInit.ClientArgs {
				require.NotContains(t, []string{"rd", "scheme", "i"}, arg.VarName)
			}
		})
	}
}

func TestPathLocalNamesGeneratedModule(t *testing.T) {
	root := RunHTTPDSL(t, testdata.PathLocalNamesDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/pathlocals", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "gen", "http", "pathlocals", "client", "paths_test.go"), []byte(pathLocalNamesHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-count=1", "./...")
}
