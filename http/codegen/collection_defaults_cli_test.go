package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestScalarMapDefaultsGeneratedCLI(t *testing.T) {
	root := RunHTTPDSL(t, testdata.OpenAPIScalarMapKeysDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/scalardefaults", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "defaults_test.go"), []byte(scalarMapDefaultsCLIHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "./...")
}

func TestCollectionBodyDefaultsGeneratedCLI(t *testing.T) {
	root := RunHTTPDSL(t, testdata.CollectionBodyDefaultsDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/collectiondefaults", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "defaults_test.go"), []byte(strings.ReplaceAll(testdata.CollectionBodyDefaultsCLIHarness, "TRANSPORT", "http")), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "./...")
}

func TestBooleanMapCLIImportDoesNotShadowService(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		dsl.Service("loomhttpcli", func() {
			dsl.Method("list", func() {
				dsl.Payload(func() {
					dsl.Attribute("flags", dsl.MapOf(dsl.Boolean, dsl.String))
					dsl.Attribute("loomhttpcli", dsl.String)
					dsl.Attribute("loomhttpcli2", dsl.String)
				})
				dsl.HTTP(func() {
					dsl.GET("/flags")
					dsl.Param("flags")
					dsl.Param("loomhttpcli")
					dsl.Param("loomhttpcli2")
				})
			})
		})
	})
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/clicollision", root)
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
}

const scalarMapDefaultsCLIHarness = `package scalardefaults_test

import (
	"flag"
	"net/http"
	"testing"

	svc "example.com/scalardefaults/gen/flags"
	cli "example.com/scalardefaults/gen/http/cli/scalar_map_keys"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

func TestCollectionFlags(t *testing.T) {
	oldFlags := flag.CommandLine
	t.Cleanup(func() {
		flag.CommandLine = oldFlags
	})
	for _, tc := range []struct {
		name string
		args []string
		labels map[bool]string
		valid bool
	}{
		{"default", nil, map[bool]string{false:"off", true:"on"}, true},
		{"override", []string{"--labels={\"false\":\"custom\"}"}, map[bool]string{false:"custom"}, true},
		{"empty", []string{"--labels={}"}, map[bool]string{}, true},
		{"empty-flag", []string{"--labels="}, nil, false},
		{"malformed", []string{"--labels={"}, nil, false},
		{"noncanonical-key", []string{"--labels={\"TRUE\":\"custom\"}"}, nil, false},
		{"wrong-value", []string{"--labels={\"true\":1}"}, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"flags", "list", "--enabled={\"true\":[false,true]}", "--groups={\"team\":{\"false\":9}}"}, tc.args...)
			flag.CommandLine = flag.NewFlagSet("collection-defaults", flag.ContinueOnError)
			require.NoError(t, flag.CommandLine.Parse(args))
			_, payload, err := cli.ParseEndpoint("http", "localhost", http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
			if !tc.valid {
				require.Error(t, err)
				require.Contains(t, err.Error(), "labels")
				return
			}
			require.NoError(t, err)
			value := payload.(*svc.ListPayload)
			require.Equal(t, tc.labels, value.Labels)
			require.Equal(t, map[bool][]bool{true:{false,true}}, value.Enabled)
			require.Equal(t, map[string]map[bool]int{"team":{false:9}}, value.Groups)
		})
	}
}
`
