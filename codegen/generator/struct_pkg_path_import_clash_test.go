package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestStructPkgPathFrameworkImportNamesCompile generates the service,
// transport and example output of designs whose HTTP, gRPC and JSON-RPC
// services use types generated in a struct:pkg:path package named like an
// import of the generated files, then builds and vets the module. A file that
// imports such a package next to an import of the same name, such as
// "types/security" next to the Loom security package, imports the type
// package under an alias; the other files keep the package name.
func TestStructPkgPathFrameworkImportNamesCompile(t *testing.T) {
	cases := []struct {
		Name string
		// Interceptors adds interceptors to the design and skips the example
		// generation: the example interceptors do not compile (#429).
		Interceptors bool
		// Files maps generated files to content they must contain.
		Files map[string][]string
		// Absent maps generated files to content they must not contain.
		Absent map[string][]string
	}{
		{Name: "atomic"},
		{Name: "bufio"},
		{Name: "bytes"},
		{Name: "codes"},
		{
			Name: "context",
			Files: map[string][]string{
				"gen/shop/service.go": {`"context"`, `context2 "example.com/probe/gen/types/context"`, "Direct(context.Context, *context2.Item) (res *context2.Item, err error)"},
			},
		},
		{Name: "grpc"},
		{Name: "io"},
		{Name: "jsonrpc"},
		{Name: "jsontext"},
		{
			Name: "loom",
			Files: map[string][]string{
				"gen/shop/endpoints.go": {`loom "github.com/CaliLuke/loom/pkg"`, `loom2 "example.com/probe/gen/types/loom"`},
			},
		},
		{Name: "loomgrpc"},
		{Name: "loomhttp"},
		{Name: "loompb"},
		{Name: "loomtransport"},
		{Name: "metadata"},
		{
			Name: "multipart",
			Files: map[string][]string{
				"gen/http/shop/client/client.go":        {`"mime/multipart"`, `multipart2 "example.com/probe/gen/types/multipart"`, "type ShopUploadEncoderFunc func(*multipart.Writer, *multipart2.Item) error"},
				"gen/http/shop/client/encode_decode.go": {`multipart2 "example.com/probe/gen/types/multipart"`, "mw := multipart.NewWriter(body)", "p, ok := v.(*multipart2.Item)"},
				"gen/grpc/shop/server/types.go":         {`multipart "example.com/probe/gen/types/multipart"`},
			},
			Absent: map[string][]string{
				"gen/grpc/shop/server/types.go": {"multipart2"},
			},
		},
		{Name: "path"},
		{
			Name: "regexp",
			Files: map[string][]string{
				"gen/jsonrpc/live/server/types.go": {`regexp2 "regexp"`, `regexp "example.com/probe/gen/types/regexp"`, "regexp2.MustCompile"},
			},
		},
		{Name: "rpcviews"},
		{
			Name: "security",
			Files: map[string][]string{
				"gen/shop/service.go":   {`"github.com/CaliLuke/loom/security"`, `security2 "example.com/probe/gen/types/security"`, "Put(context.Context, *PutPayload) (res *security2.Item, err error)"},
				"gen/shop/endpoints.go": {`"github.com/CaliLuke/loom/security"`, `security2 "example.com/probe/gen/types/security"`},
				"gen/rpc/service.go":    {`security2 "example.com/probe/gen/types/security"`},
			},
			Absent: map[string][]string{
				"gen/http/shop/server/encode_decode.go": {"security2"},
			},
		},
		{Name: "security-interceptors", Interceptors: true},
		{Name: "shoppb"},
		{Name: "shopviews"},
		{Name: "strconv"},
		{Name: "strings"},
		{Name: "sync"},
		{Name: "utf8"},
		{Name: "websocket"},
		{
			Name: "log",
			Absent: map[string][]string{
				"gen/shop/service.go":                   {"log2"},
				"gen/http/shop/server/encode_decode.go": {"log2"},
			},
		},
		{Name: "common"},
	}
	source := structPkgPathLoomSource(t)
	dirs := make(map[string]string, len(cases))
	for _, c := range cases {
		name, _, _ := strings.Cut(c.Name, "-")
		codegen.RunDSL(t, structPkgPathClashDSL(name, c.Interceptors))
		dir := writeStructPkgPathModule(t, source)
		_, err := Generate(dir, "gen", false)
		require.NoError(t, err, c.Name)
		if !c.Interceptors {
			_, err = Generate(dir, "example", false)
			require.NoError(t, err, c.Name)
		}
		for path, wants := range c.Files {
			content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(path)))
			if !assert.NoError(t, err, c.Name) {
				continue
			}
			for _, want := range wants {
				assert.Contains(t, string(content), want, "%s: %s", c.Name, path)
			}
		}
		for path, absents := range c.Absent {
			content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(path)))
			if !assert.NoError(t, err, c.Name) {
				continue
			}
			for _, absent := range absents {
				assert.NotContains(t, string(content), absent, "%s: %s", c.Name, path)
			}
		}
		dirs[c.Name] = dir
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()
			dir := dirs[c.Name]
			for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}} {
				output, err := testingx.RunCmd(dir, "go", args...)
				require.NoError(t, err, output)
			}
		})
	}
}

// structPkgPathClashDSL returns a design whose "shop" service, exposed over
// HTTP and gRPC, "live" and "rpc" services, exposed over JSON-RPC, use types
// generated in the struct:pkg:path package "types/<name>" in payloads,
// results, unions, errors, views, security, multipart requests, streams,
// server-sent events and parameters.
// interceptors adds a server and a client interceptor to a method.
func structPkgPathClashDSL(name string, interceptors bool) func() {
	pkgPath := "types/" + name
	return func() {
		dsl.API("probe", func() {
			dsl.JSONRPC(func() {})
		})
		item := dsl.Type("Item", func() {
			dsl.Meta("struct:pkg:path", pkgPath)
			dsl.Field(1, "id", dsl.String)
			dsl.Field(2, "name", dsl.String)
			dsl.Field(3, "code", dsl.String, func() {
				dsl.Pattern("^[a-z]+$")
			})
			dsl.Required("id")
		})
		code := dsl.Type("Code", dsl.String, func() {
			dsl.Meta("struct:pkg:path", pkgPath)
			dsl.Pattern("^[A-Z]+$")
		})
		card := dsl.ResultType("application/vnd.probe.card", "Card", func() {
			dsl.Meta("struct:pkg:path", pkgPath)
			dsl.Attributes(func() {
				dsl.Field(1, "id", dsl.String)
				dsl.Field(2, "item", item)
			})
			dsl.View("default", func() {
				dsl.Attribute("id")
				dsl.Attribute("item")
			})
			dsl.View("tiny", func() {
				dsl.Attribute("id")
			})
		})
		choice := dsl.Type("Choice", func() {
			dsl.Meta("struct:pkg:path", pkgPath)
			dsl.OneOf("pick", func() {
				dsl.Field(1, "text", dsl.String)
				dsl.Field(2, "item", item)
			})
		})
		audit := dsl.Interceptor("Audit", func() {
			dsl.ReadPayload(func() {
				dsl.Attribute("id")
			})
			dsl.WriteResult(func() {
				dsl.Attribute("name")
			})
		})
		failure := dsl.Type("Failure", func() {
			dsl.Meta("struct:pkg:path", pkgPath)
			dsl.Field(1, "message", dsl.String)
			dsl.Required("message")
		})
		conflict := dsl.Type("Conflict", func() {
			dsl.Meta("struct:pkg:path", pkgPath)
			dsl.Field(1, "reason", dsl.String)
		})
		jwt := dsl.JWTSecurity("jwt", func() {
			dsl.Scope("api:read")
		})
		key := dsl.APIKeySecurity("key")
		basic := dsl.BasicAuthSecurity("basic")
		oauth := dsl.OAuth2Security("oauth", func() {
			dsl.ClientCredentialsFlow("/token", "/refresh")
			dsl.Scope("api:read")
		})
		both := func(route func()) {
			dsl.HTTP(route)
			dsl.GRPC(func() {})
		}
		dsl.Service("shop", func() {
			dsl.Error("failed", failure)
			dsl.Method("put", func() {
				dsl.Security(jwt, func() {
					dsl.Scope("api:read")
				})
				dsl.Payload(func() {
					dsl.Token("token", dsl.String)
					dsl.Field(1, "item", item)
					dsl.Required("item", "token")
				})
				dsl.Result(item)
				both(func() {
					dsl.POST("/v1/items")
					dsl.Body("item")
					dsl.Response("failed", dsl.StatusBadRequest)
				})
			})
			dsl.Method("keyed", func() {
				dsl.Security(key)
				dsl.Payload(func() {
					dsl.APIKey("key", "key", dsl.String)
					dsl.Field(1, "item", item)
					dsl.Field(2, "code", code)
				})
				dsl.Result(dsl.ArrayOf(item))
				both(func() {
					dsl.POST("/v1/keyed")
					dsl.Param("key")
					dsl.Param("code")
				})
			})
			dsl.Method("basic", func() {
				dsl.Security(basic)
				dsl.Payload(func() {
					dsl.Username("user", dsl.String)
					dsl.Password("pass", dsl.String)
					dsl.Field(1, "item", item)
				})
				dsl.Result(dsl.MapOf(dsl.String, item))
				both(func() {
					dsl.POST("/v1/basic")
				})
			})
			dsl.Method("oauth", func() {
				dsl.Security(oauth)
				dsl.Payload(func() {
					dsl.AccessToken("tok", dsl.String)
					dsl.Field(1, "item", item)
				})
				dsl.Result(item)
				both(func() {
					dsl.POST("/v1/oauth")
				})
			})
			dsl.Method("direct", func() {
				if interceptors {
					dsl.ServerInterceptor(audit)
					dsl.ClientInterceptor(audit)
				}
				dsl.Payload(item)
				dsl.Result(item)
				dsl.Error("broken", conflict)
				both(func() {
					dsl.PUT("/v1/direct")
					dsl.Response("broken", dsl.StatusConflict)
				})
			})
			dsl.Method("show", func() {
				dsl.Payload(func() {
					dsl.Field(1, "view", dsl.String)
				})
				dsl.Result(card)
				both(func() {
					dsl.GET("/v1/show")
					dsl.Param("view")
				})
			})
			dsl.Method("choose", func() {
				dsl.Payload(choice)
				dsl.Result(choice)
				both(func() {
					dsl.POST("/v1/choose")
				})
			})
			dsl.Method("upload", func() {
				dsl.Payload(item)
				dsl.Result(item)
				dsl.HTTP(func() {
					dsl.POST("/v1/upload")
					dsl.MultipartRequest()
				})
			})
			dsl.Method("relay", func() {
				dsl.StreamingPayload(item)
				dsl.StreamingResult(item)
				both(func() {
					dsl.GET("/v1/relay")
				})
			})
			dsl.Method("events", func() {
				dsl.StreamingResult(item)
				dsl.HTTP(func() {
					dsl.GET("/v1/events")
					dsl.ServerSentEvents()
				})
			})
			dsl.Method("watch", func() {
				dsl.StreamingResult(item)
				dsl.GRPC(func() {})
			})
		})
		dsl.Service("live", func() {
			dsl.JSONRPC(func() {
				dsl.GET("/live")
			})
			dsl.Method("exchange", func() {
				dsl.StreamingPayload(item)
				dsl.StreamingResult(item)
				dsl.JSONRPC(func() {})
			})
		})
		dsl.Service("rpc", func() {
			dsl.Security(jwt)
			dsl.JSONRPC(func() {
				dsl.POST("/rpc")
			})
			dsl.Method("put", func() {
				dsl.Payload(func() {
					dsl.Token("token", dsl.String)
					dsl.Field(1, "item", item)
					dsl.Required("token")
				})
				dsl.Result(item)
				dsl.Error("failed", conflict)
				dsl.JSONRPC(func() {
					dsl.Response("failed", dsl.RPCInvalidParams)
				})
			})
			dsl.Method("events", func() {
				dsl.Payload(func() {
					dsl.Token("token", dsl.String)
					dsl.ID("id", dsl.String)
				})
				dsl.StreamingResult(item)
				dsl.JSONRPC(func() {
					dsl.ServerSentEvents()
				})
			})
		})
	}
}
