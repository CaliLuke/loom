package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/loom/internal/testingx"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

const cliJSONBodyHarness = `package presence_test

import (
    "reflect"
    "testing"
    "context"
    "net/http"
    "net/http/httptest"
    "os"
    "strings"

    hcli "example.com/presence/gen/http/cli/presence"
    rcli "example.com/presence/gen/jsonrpc/cli/presence"
    ps "example.com/presence/gen/http/plain/server"
    rs "example.com/presence/gen/jsonrpc/rpc/server"
    plain "example.com/presence/gen/plain"
    rpc "example.com/presence/gen/rpc"
    loom "github.com/CaliLuke/loom/pkg"
    loomhttp "github.com/CaliLuke/loom/http"

    pc "example.com/presence/gen/http/plain/client"
    nc "example.com/presence/gen/http/nestedplain/client"
    rc "example.com/presence/gen/jsonrpc/rpc/client"
    nr "example.com/presence/gen/jsonrpc/nestedrpc/client"
    "github.com/stretchr/testify/require"
    "github.com/google/uuid"
)

type recordingCLIDoer struct {
    client *http.Client
    status int
}

func TestCLIBodyFields(t *testing.T) {
    for _, transport := range []struct {
        name string
        echo func(string) (any, error)
        required func(string) (any, error)
        nested func(string) (any, error)
    }{
        {"http", erase(pc.BuildEchoPayload), erase(pc.BuildRequiredPayload), erase(nc.BuildEchoPayload)},
        {"jsonrpc", erase(rc.BuildEchoPayload), erase(rc.BuildRequiredPayload), erase(nr.BuildEchoPayload)},
    } {
        t.Run(transport.name, func(t *testing.T) {
            for _, tc := range []struct {
                name, input string
                flag bool
                count int
                label string
            }{
                {"absent defaults", "{}", true, 7, "fallback"},
                {"supplied zeros", "{\"flag\":false,\"count\":0,\"label\":\"\"}", false, 0, ""},
                {"populated", "{\"flag\":true,\"count\":9,\"label\":\"value\"}", true, 9, "value"},
            } {
                t.Run(tc.name, func(t *testing.T) {
                    value, err := transport.echo(tc.input)
                    require.NoError(t, err)
                    actual := reflect.ValueOf(value).Elem()
                    require.Equal(t, tc.flag, actual.FieldByName("Flag").Bool())
                    require.Equal(t, int64(tc.count), actual.FieldByName("Count").Int())
                    require.Equal(t, tc.label, actual.FieldByName("Label").String())
                })
            }
            for _, input := range []string{"", " \t\r\n", "{", "{\"flag\":", "null", "{\"flag\":null}"} {
                t.Run("reject "+input, func(t *testing.T) {
                    _, err := transport.echo(input)
                    require.Error(t, err)
                })
            }
            t.Run("empty collections retained", func(t *testing.T) {
                value, err := transport.echo("{\"bytes\":\"\",\"list\":[],\"mapping\":{}}")
                require.NoError(t, err)
                actual := reflect.ValueOf(value).Elem()
                for _, name := range []string{"Bytes", "List", "Mapping"} {
                    require.False(t, actual.FieldByName(name).IsNil(), name)
                    require.Zero(t, actual.FieldByName(name).Len(), name)
                }
            })
            t.Run("custom JSON codec", func(t *testing.T) {
                value, err := transport.nested("{\"child\":{\"token\":\"550e8400-e29b-41d4-a716-446655440000\"}}")
                require.NoError(t, err)
                child := reflect.ValueOf(value).Elem().FieldByName("Child").Elem()
                token := child.FieldByName("Token").Interface().(*uuid.UUID)
                require.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), *token)
            })
            t.Run("recursive nested defaults", func(t *testing.T) {
                value, err := transport.nested("{\"child\":{\"next\":{}}}")
                require.NoError(t, err)
                child := reflect.ValueOf(value).Elem().FieldByName("Child").Elem().FieldByName("Next").Elem()
                require.True(t, child.FieldByName("Flag").Bool())
            })
            t.Run("missing required", func(t *testing.T) {
                _, err := transport.required("{}")
                require.Error(t, err)
                require.Contains(t, err.Error(), "value")
            })
            t.Run("supplied required zero", func(t *testing.T) {
                value, err := transport.required("{\"value\":false}")
                require.NoError(t, err)
                require.False(t, reflect.ValueOf(value).Elem().FieldByName("Value").Bool())
            })
            for _, tc := range []struct {
                name, input string
                missing bool
                flag bool
            }{
                {"absent child remains absent", "{}", true, false},
                {"nested absent member defaults", "{\"child\":{}}", false, true},
                {"nested supplied zero", "{\"child\":{\"flag\":false}}", false, false},
            } {
                t.Run(tc.name, func(t *testing.T) {
                    value, err := transport.nested(tc.input)
                    require.NoError(t, err)
                    child := reflect.ValueOf(value).Elem().FieldByName("Child")
                    require.Equal(t, tc.missing, child.IsNil())
                    if !tc.missing {
                        require.Equal(t, tc.flag, child.Elem().FieldByName("Flag").Bool())
                    }
                })
            }
        })
    }
}

func TestCLIBodyNullable(t *testing.T) {
    for _, input := range []string{"{}", "{\"value\":null}", "{\"value\":\"\"}"} {
        t.Run(input, func(t *testing.T) {
            h, err := pc.BuildNullablePayload(input)
            require.NoError(t, err)
            r, err := rc.BuildNullablePayload(input)
            require.NoError(t, err)
            if input == "{\"value\":null}" {
                require.True(t, h.Value.IsNull())
                require.True(t, r.Value.IsNull())
            } else {
                expected := ""
                if input == "{}" {
                    expected = "fallback"
                }
                hv, hok := h.Value.Value()
                rv, rok := r.Value.Value()
                require.True(t, hok)
                require.True(t, rok)
                require.Equal(t, expected, hv)
                require.Equal(t, expected, rv)
            }
        })
    }
}

func TestCLIInputDispatch(t *testing.T) {
    original := os.Args
    t.Cleanup(func() {
        os.Args = original
    })
    for _, transport := range []string{"http", "jsonrpc"} {
        t.Run(transport, func(t *testing.T) {
            calls := 0
            var received bool
            endpoint := loom.Endpoint(func(_ context.Context, value any) (any, error) {
                calls++
                received = reflect.ValueOf(value).Elem().FieldByName("Value").Bool()
                return nil, nil
            })
            mux := loomhttp.NewMuxer()
            name, status := "plain", http.StatusNoContent
            parse := hcli.ParseEndpoint
            if transport == "http" {
                ps.Mount(mux, ps.New(&plain.Endpoints{Required: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
            } else {
                name, status, parse = "rpc", http.StatusOK, rcli.ParseEndpoint
                rs.Mount(mux, rs.New(&rpc.Endpoints{Required: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
            }
            server := httptest.NewServer(mux)
            defer server.Close()
            doer := &recordingCLIDoer{client: server.Client()}
            for _, tc := range []struct {
                body string
                valid, value bool
            }{
                {"{}", false, false},
                {"null", false, false},
                {"{", false, false},
                {"{\"value\":false}", true, false},
                {"{\"value\":true}", true, true},
            } {
                t.Run(tc.body, func(t *testing.T) {
                    calls, doer.status = 0, 0
                    os.Args = []string{"cli", name, "required", "--body", tc.body}
                    call, payload, err := parse("http", strings.TrimPrefix(server.URL, "http://"), doer, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
                    if !tc.valid {
                        require.Error(t, err)
                        require.Nil(t, call)
                        require.Nil(t, payload)
                        require.Zero(t, calls)
                        require.Zero(t, doer.status)
                        if tc.body == "{}" {
                            require.Contains(t, err.Error(), "value")
                        }
                        return
                    }
                    require.NoError(t, err)
                    require.Equal(t, tc.value, reflect.ValueOf(payload).Elem().FieldByName("Value").Bool())
                    _, err = call(context.Background(), payload)
                    require.NoError(t, err)
                    require.Equal(t, 1, calls)
                    require.Equal(t, tc.value, received)
                    require.Equal(t, status, doer.status)
                })
            }
        })
    }
}


func TestCLIBareCollection(t *testing.T) {
    original := os.Args
    t.Cleanup(func() {
        os.Args = original
    })
    for _, transport := range []string{"plain", "rpc"} {
        for _, tc := range []struct {
            body string
            valid bool
            want []string
        }{
            {"[]", true, []string{}},
            {"[\"\"]", true, []string{""}},
            {"null", false, nil},
            {"[null]", false, nil},
        } {
            t.Run(transport+tc.body, func(t *testing.T) {
                os.Args = []string{"cli", transport, "list", "--p", tc.body}
                parse := hcli.ParseEndpoint
                if transport == "rpc" {
                    parse = rcli.ParseEndpoint
                }
                endpoint, payload, err := parse("http", "localhost", nil, nil, nil, false)
                if !tc.valid {
                    require.Error(t, err)
                    require.Nil(t, endpoint)
                    require.Nil(t, payload)
                    return
                }
                require.NoError(t, err)
                require.Equal(t, tc.want, payload)
            })
        }
    }
}

func (d *recordingCLIDoer) Do(request *http.Request) (*http.Response, error) {
    response, err := d.client.Do(request)
    if response != nil {
        d.status = response.StatusCode
    }
    return response, err
}

func erase[T any](build func(string) (T, error)) func(string) (any, error) {
    return func(input string) (any, error) {
        return build(input)
    }
}

`

func TestCLIJSONBodyContracts(t *testing.T) {
	dir := t.TempDir()
	renderDesignModule(t, dir, "example.com/presence", cliJSONBodyDSL)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cli_test.go"), []byte(cliJSONBodyHarness), 0o600))
	for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}, {"test", "-race", "-count=1", "."}} {
		output, err := testingx.RunCmd(dir, "go", args...)
		require.NoError(t, err, output)
	}
}

func cliJSONBodyDSL() {
	cliJSONBodyBaseDSL()
	var child expr.UserType
	child = dsl.Type("CLIChild", func() {
		dsl.Attribute("flag", dsl.Boolean, func() {
			dsl.Default(true)
		})
		dsl.Attribute("token", dsl.String, func() {
			dsl.Meta("struct:field:type", "uuid.UUID", "github.com/google/uuid")
		})
		dsl.Attribute("next", child)
	})
	for _, name := range []string{"nestedplain", "nestedrpc"} {
		dsl.Service(name, func() {
			if name == "nestedrpc" {
				dsl.JSONRPC(func() {
					dsl.POST("/nestedrpc")
				})
			}
			dsl.Method("echo", func() {
				dsl.Payload(func() {
					dsl.Attribute("child", child)
				})
				if name == "nestedrpc" {
					dsl.JSONRPC(func() {
					})
				} else {
					dsl.HTTP(func() {
						dsl.POST("/nested")
					})
				}
			})
		})
	}
}

func cliJSONBodyBaseDSL() {
	dsl.API("presence", func() {
		dsl.JSONRPC(func() {
		})
	})
	label := dsl.Type("Label", dsl.String)
	state := dsl.Type("State", func() {
		dsl.Field(1, "flag", dsl.Boolean, func() {
			dsl.Default(true)
		})
		dsl.Field(2, "count", dsl.Int, func() {
			dsl.Default(7)
		})
		dsl.Field(3, "text", dsl.String)
		dsl.Field(4, "bytes", dsl.Bytes)
		dsl.Field(5, "list", dsl.ArrayOf(dsl.String))
		dsl.Field(6, "mapping", dsl.MapOf(dsl.String, dsl.String))
		dsl.Field(7, "label", label, func() {
			dsl.Default("fallback")
		})
	})
	nullable := dsl.Type("NullableState", func() {
		dsl.Attribute("value", dsl.String, func() {
			dsl.Nullable()
			dsl.Default("fallback")
		})
	})
	for _, name := range []string{"plain", "rpc", "proto"} {
		dsl.Service(name, func() {
			if name == "rpc" {
				dsl.JSONRPC(func() {
					dsl.POST("/rpc")
				})
			}
			if name != "proto" {
				dsl.Method("list", func() {
					dsl.Payload(dsl.ArrayOf(dsl.String))
					if name == "rpc" {
						dsl.JSONRPC(func() {
						})
					} else {
						dsl.HTTP(func() {
							dsl.POST("/list")
						})
					}
				})
				dsl.Method("required", func() {
					dsl.Payload(func() {
						dsl.Attribute("value", dsl.Boolean, func() {
							dsl.Default(true)
						})
						dsl.Required("value")
					})
					if name == "rpc" {
						dsl.JSONRPC(func() {
						})
					} else {
						dsl.HTTP(func() {
							dsl.POST("/required")
						})
					}
				})
				dsl.Method("nullable", func() {
					dsl.Payload(nullable)
					dsl.Result(nullable)
					if name == "rpc" {
						dsl.JSONRPC(func() {
						})
					} else {
						dsl.HTTP(func() {
							dsl.POST("/nullable")
						})
					}
				})
			}
			dsl.Method("echo", func() {
				dsl.Payload(state)
				dsl.Result(state)
				switch name {
				case "plain":
					dsl.HTTP(func() {
						dsl.POST("/echo")
					})
				case "rpc":
					dsl.JSONRPC(func() {
					})
				case "proto":
					dsl.GRPC(func() {
					})
				}
			})
		})
	}
}
