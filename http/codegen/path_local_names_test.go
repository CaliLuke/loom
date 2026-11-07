package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
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

// pathIntegerTextDSL places authored path params whose names equal the text
// locals that faithful integer path formatting derives for opaque storage,
// before and after the integer, for scalar and array params.
func pathIntegerTextDSL() {
	duration := func() {
		Meta("struct:field:type", "time.Duration", "time")
	}
	Service("pathtext", func() {
		Method("scalar", func() {
			Payload(func() {
				Attribute("d", Int, duration)
				Attribute("d_text", String)
				Required("d", "d_text")
			})
			HTTP(func() {
				GET("/scalar/{d}/{d_text}")
			})
		})
		Method("first", func() {
			Payload(func() {
				Attribute("d_text", String)
				Attribute("d", Int, duration)
				Required("d_text", "d")
			})
			HTTP(func() {
				GET("/first/{d_text}/{d}")
			})
		})
		Method("array", func() {
			Payload(func() {
				Attribute("ds", ArrayOf(Int, duration))
				Attribute("v_text", String)
				Attribute("es", ArrayOf(Int, duration))
				Required("ds", "v_text", "es")
			})
			HTTP(func() {
				GET("/array/{ds}/{v_text}/{es}")
			})
		})
	})
}

const pathIntegerTextHarness = `package client

import (
	"context"
	"testing"
	"time"

	svc "example.com/pathtext/gen/pathtext"
)

func TestIntegerTextLocals(t *testing.T) {
	if got := ScalarPathtextPath(5*time.Second, "x"); got != "/scalar/5000000000/x" {
		t.Errorf("scalar path = %q", got)
	}
	if got := FirstPathtextPath("x", -5); got != "/first/x/-5" {
		t.Errorf("first path = %q", got)
	}
	if got := ArrayPathtextPath([]time.Duration{-1, 2}, "x", []time.Duration{3}); got != "/array/-1,2/x/3" {
		t.Errorf("array path = %q", got)
	}
	c := &Client{scheme: "https", host: "example.com"}
	req, err := c.BuildScalarRequest(context.Background(), &svc.ScalarPayload{D: 5 * time.Second, DText: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if got := req.URL.String(); got != "https://example.com/scalar/5000000000/x" {
		t.Errorf("scalar URL = %q", got)
	}
}
`

func TestPathIntegerTextLocalsGeneratedModule(t *testing.T) {
	root := RunHTTPDSL(t, pathIntegerTextDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/pathtext", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "gen", "http", "pathtext", "client", "paths_test.go"), []byte(pathIntegerTextHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-count=1", "./...")
}
