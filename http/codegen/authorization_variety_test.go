package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestAuthorizationVarietyShapesCompile(t *testing.T) {
	cases := []struct {
		name   string
		design func()
	}{
		{"custom scalar representation", func() {
			fields := func() {
				Attribute("timestamp", String, func() {
					Meta("struct:field:type", "time.Time", "time")
				})
				Required("timestamp")
			}
			input := Type("Input", fields)
			access := Authorization("edit", input)
			Service("variety", func() {
				Method("run", func() {
					Payload(fields)
					Authorize(access, func() {
						Bind("timestamp", "timestamp")
					})
				})
			})
		}},
		{"named union binding", func() {
			text := Type("TextValue", String)
			number := Type("NumberValue", Int)
			value := Type("Choice", OneOf(text, number))
			input := Type("Input", func() {
				Attribute("value", value)
				Required("value")
			})
			access := Authorization("edit", input)
			Service("variety", func() {
				Method("run", func() {
					Payload(func() {
						Attribute("value", value)
					})
					Authorize(access, func() {
						Bind("value", "value")
					})
				})
			})
		}},
		{"nested validated collections", func() {
			item := Type("Item", func() {
				Attribute("name", String, func() {
					Pattern("^[a-z]+$")
				})
				Required("name")
			})
			items := Type("Items", ArrayOf(item), func() {
				MinLength(1)
			})
			mapItem := Type("MapItem", String, func() {
				MinLength(1)
			})
			fields := func() {
				Attribute("items", MapOf(String, items))
				Attribute("nested", MapOf(String, MapOf(String, mapItem)))
			}
			input := Type("Input", fields)
			access := Authorization("edit", input)
			Service("variety", func() {
				Method("run", func() {
					Payload(fields)
					Authorize(access, func() {
						Bind("items", "items")
						Bind("nested", "nested")
					})
					HTTP(func() {
						POST("/run")
					})
				})
			})
		}},
		{"raw body and mixed results", func() {
			input := Type("Input", func() {
				Attribute("id", String)
				Required("id")
			})
			access := Authorization("edit", input)
			Service("variety", func() {
				for _, method := range []string{"raw", "mixed"} {
					Method(method, func() {
						Payload(input)
						Authorize(access, func() {
							Bind("id", "id")
						})
						Result(String)
						if method == "mixed" {
							StreamingResult(input)
						}
						HTTP(func() {
							POST("/" + method)
							Param("id")
							if method == "raw" {
								SkipRequestBodyEncodeDecode()
							} else {
								ServerSentEvents()
							}
						})
					})
				}
			})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, tc.design)
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/accessvariety", root)
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "test", "./gen/...")
		})
	}
}

func TestAuthorizationVarietyTaggedUnion(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		input := Type("Input", func() {
			Attribute("id", String, func() {
				MinLength(2)
			})
			Required("id")
		})
		access := Authorization("edit", input)
		operation := Type("Operation", func() {
			OneOf("value", func() {
				Attribute("write", input, func() {
					Meta("oneof:type:tag", "document/write.v1")
				})
				Attribute("read", String, func() {
					Meta("oneof:type:tag", "document/read.v1")
				})
			})
			Required("value")
		})
		Service("variety", func() {
			Method("dispatch", func() {
				Payload(func() {
					Attribute("operation", operation)
				})
				AuthorizeBy("operation.value", func() {
					AuthorizationCase("write", func() {
						Authorize(access, func() {
							Bind("id", "operation.value.id")
						})
					})
					AuthorizationCase("read", func() {
						NoAccessCheck("Read-only public projection")
					})
				})
				Error("forbidden")
				Result(String)
				HTTP(func() {
					POST("/dispatch")
					Response("forbidden", StatusForbidden)
				})
			})
		})
	})
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/accesstags", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "authorization_test.go"), []byte(authorizationVarietyTaggedHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

const authorizationVarietyTaggedHarness = `package integration

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	variety "example.com/accesstags/gen/variety"
	server "example.com/accesstags/gen/http/variety/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

type service struct {
	calls int
}

func (s *service) Dispatch(context.Context, *variety.DispatchPayload) (string, error) {
	s.calls++
	return "ok", nil
}

type access struct {
	checks int
}

func (a *access) AuthorizeEdit(_ context.Context, input *variety.Input) error {
	a.checks++
	if input.ID != "allowed" {
		return loom.PermanentError("forbidden", "Access denied")
	}
	return nil
}

func TestTaggedUnionAuthorization(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status, checks, calls int
	}{
		{"allowed write", "{\"operation\":{\"value\":{\"type\":\"document/write.v1\",\"value\":{\"id\":\"allowed\"}}}}", 200, 1, 1},
		{"denied write", "{\"operation\":{\"value\":{\"type\":\"document/write.v1\",\"value\":{\"id\":\"denied\"}}}}", 403, 1, 0},
		{"public read", "{\"operation\":{\"value\":{\"type\":\"document/read.v1\",\"value\":\"public\"}}}", 200, 0, 1},
		{"unknown wire tag", "{\"operation\":{\"value\":{\"type\":\"write\",\"value\":{\"id\":\"allowed\"}}}}", 400, 0, 0},
		{"nil ancestor", "{}", 400, 0, 0},
		{"nil branch", "{\"operation\":{\"value\":{\"type\":\"document/write.v1\",\"value\":null}}}", 400, 0, 0},
		{"invalid branch", "{\"operation\":{\"value\":{\"type\":\"document/write.v1\",\"value\":{\"id\":\"x\"}}}}", 400, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &service{}
			a := &access{}
			e := variety.NewEndpoints(s, a)
			mux := loomhttp.NewMuxer()
			server.Mount(mux, server.New(e, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
			req := httptest.NewRequest("POST", "/dispatch", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()
			mux.ServeHTTP(res, req)
			require.Equal(t, tc.status, res.Code, res.Body.String())
			require.Equal(t, tc.checks, a.checks)
			require.Equal(t, tc.calls, s.calls)
		})
	}
}

func TestTaggedUnionMutation(t *testing.T) {
	for _, tc := range []struct {
		name string
		mutate func(*variety.DispatchPayload)
	}{
		{"exempt to denied branch", func(p *variety.DispatchPayload) {
			p.Operation.Value = variety.NewValueWrite(&variety.Input{ID: "denied"})
		}},
		{"exempt to malformed branch", func(p *variety.DispatchPayload) {
			p.Operation.Value = variety.NewValueWrite(nil)
		}},
		{"remove ancestor", func(p *variety.DispatchPayload) {
			p.Operation = nil
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &service{}
			a := &access{}
			e := variety.NewEndpoints(s, a)
			e.Use(func(next loom.Endpoint) loom.Endpoint {
				return func(ctx context.Context, req any) (any, error) {
					tc.mutate(req.(*variety.DispatchPayload))
					return next(ctx, req)
				}
			})
			_, err := e.Dispatch(context.Background(), &variety.DispatchPayload{
				Operation: &variety.Operation{Value: variety.NewValueRead("public")},
			})
			require.Error(t, err)
			require.Zero(t, s.calls)
		})
	}
}
`
