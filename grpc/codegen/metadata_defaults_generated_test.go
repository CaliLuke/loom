package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestMetadataDefaultsGenerated(t *testing.T) {
	runMetadataDefaultsModule(t, RunGRPCDSL(t, metadataDefaultsDSL), map[string]string{
		"rt_test.go":   presenceMetadataHarness,
		"wire_test.go": metadataDefaultsWireHarness,
	})
}

func runMetadataDefaultsModule(t *testing.T, root *expr.RootExpr, harnesses map[string]string) {
	t.Helper()
	const modulePath = "example.com/mdprobe"
	dir := t.TempDir()
	renderGRPCModule(t, dir, modulePath, root, resolveGRPCLoomSource(t))
	modPath := filepath.Join(dir, "go.mod")
	mod, err := os.ReadFile(modPath)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(modPath, []byte(strings.ReplaceAll(string(mod), "require github.com/CaliLuke/loom v1.0.0", "require github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261013020843-063198a2f427")), 0o600))
	testDir := filepath.Join(dir, "internal", "rt")
	require.NoError(t, os.MkdirAll(testDir, 0o750))
	for name, source := range harnesses {
		require.NoError(t, os.WriteFile(filepath.Join(testDir, name), []byte(source), 0o600))
	}
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "build", "./...")
	runGRPCGoCommand(t, dir, "vet", "./...")
	runGRPCGoCommand(t, dir, "test", "./internal/rt", "-race", "-v")
}

func metadataDefaultsDSL() {
	label := Type("Label", String)
	var V = Type("V", func() {
		Field(1, "body", String)
		Attribute("limit", Int, func() {
			Default(3)
		})
		Attribute("lang", String, func() {
			Default("en")
		})
		Attribute("flag", Boolean, func() {
			Default(true)
		})
		Attribute("data", Bytes, func() {
			Default([]byte("fallback"))
		})
		Attribute("label", label, func() {
			Default("fallback")
		})
		Attribute("tags", ArrayOf(String), func() {
			Default([]string{"fallback"})
		})
		Attribute("numbers", ArrayOf(Int), func() {
			Default([]int{3})
		})
		Attribute("must", String, func() {
			Default("required default")
		})
		Attribute("optional", String)
		Required("body", "must")
	})
	Service("mdsvc", func() {
		Method("echo", func() {
			Payload(V)
			Result(V)
			GRPC(func() {
				Message(func() {
					Attribute("body")
				})
				Metadata(func() {
					Attribute("limit")
					Attribute("flag")
					Attribute("data")
					Attribute("label")
					Attribute("tags")
					Attribute("numbers")
					Attribute("must")
					Attribute("optional")
					Attribute("lang")
				})
				Response(CodeOK, func() {
					Message(func() {
						Attribute("body")
					})
					Headers(func() {
						Attribute("limit")
						Attribute("flag")
						Attribute("data")
						Attribute("label")
						Attribute("tags")
						Attribute("numbers")
						Attribute("must")
						Attribute("optional")
					})
					Trailers(func() {
						Attribute("lang")
					})
				})
			})
		})
	})
}

func metadataLocalDefaultsDSL() {
	Service("local", func() {
		Method("read", func() {
			Payload(func() {
				Attribute("value", Int32)
			})
			GRPC(func() {
				Metadata(func() {
					Attribute("value", Int32, func() {
						Default(7)
					})
				})
			})
		})
	})
}

const presenceMetadataHarness = `package rt

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	client "example.com/mdprobe/gen/grpc/mdsvc/client"
	mdsvcpb "example.com/mdprobe/gen/grpc/mdsvc/pb"
	server "example.com/mdprobe/gen/grpc/mdsvc/server"
	mdsvc "example.com/mdprobe/gen/mdsvc"
)

func TestMetadataPresence(t *testing.T) {
	for _, tc := range []struct {
		name    string
		values  metadata.MD
		limit   int
		lang    string
		flag    bool
		data, label string
		tags []string
		numbers []int
		optional *string
		invalid bool
	}{
		{name: "absent defaults", values: metadata.Pairs("must", "present"), limit: 3, lang: "en", flag: true, data: "fallback", label: "fallback", tags: []string{"fallback"}, numbers: []int{3}},
		{name: "explicit zero empty", values: metadata.Pairs("must", "present", "limit", "0", "lang", "", "flag", "false", "data-bin", "", "label", "", "tags", "", "numbers", "0", "optional", ""), tags: []string{""}, numbers: []int{0}, optional: strptr("")},
		{name: "ordinary", values: metadata.Pairs("must", "present", "limit", "9", "lang", "it", "flag", "true", "data-bin", "value", "label", "value", "tags", "a", "tags", "b", "numbers", "2", "numbers", "4"), limit: 9, lang: "it", flag: true, data: "value", label: "value", tags: []string{"a", "b"}, numbers: []int{2,4}},
		{name: "malformed integer", values: metadata.Pairs("must", "present", "limit", "bad"), invalid: true},
		{name: "malformed bool", values: metadata.Pairs("must", "present", "flag", "bad"), invalid: true},
		{name: "malformed element", values: metadata.Pairs("must", "present", "numbers", "bad"), invalid: true},
		{name: "required absent", values: metadata.MD{}, invalid: true},
	} {
		for _, response := range []bool{false, true} {
			t.Run(tc.name+"/response="+strconv.FormatBool(response), func(t *testing.T) {
				body := "b"
				var decoded any
				var err error
				if response {
					decoded, err = client.DecodeEchoResponse(context.Background(), &mdsvcpb.EchoResponse{Body: &body}, tc.values, tc.values)
				} else {
					decoded, err = server.DecodeEchoRequest(context.Background(), &mdsvcpb.EchoRequest{Body: &body}, tc.values)
				}
				if tc.invalid {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				v := decoded.(*mdsvc.V)
				checks := map[string]struct{want, got any}{
					"integer": {tc.limit, v.Limit},
					"string": {tc.lang, v.Lang},
					"bool": {tc.flag, v.Flag},
					"bytes": {[]byte(tc.data), v.Data},
					"alias": {mdsvc.Label(tc.label), v.Label},
					"string slice": {tc.tags, v.Tags},
					"integer slice": {tc.numbers, v.Numbers},
					"optional": {tc.optional, v.Optional},
					"required": {"present", v.Must},
				}
				for name, check := range checks {
					t.Run(name, func(t *testing.T) {
						require.Equal(t, check.want, check.got)
					})
				}
			})
		}
	}
}

func strptr(value string) *string {
	return &value
}
`
