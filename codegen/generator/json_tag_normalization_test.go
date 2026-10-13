package generator

import (
	"testing"

	dsl "github.com/CaliLuke/loom/dsl"
)

const jsonTagNormalizationHarness = `package omission_test

import (
    "context"
    "encoding/json/jsontext"
    "encoding/json/v2"
    "io"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
    "time"

    plain "example.com/omission/gen/plain"
    rpc "example.com/omission/gen/rpc"
    pc "example.com/omission/gen/http/plain/client"
    ps "example.com/omission/gen/http/plain/server"
    rc "example.com/omission/gen/jsonrpc/rpc/client"
    rs "example.com/omission/gen/jsonrpc/rpc/server"
    loomhttp "github.com/CaliLuke/loom/http"
    "github.com/stretchr/testify/require"
)

func TestClientRequest(t *testing.T) {
    for _, mode := range []string{"absent", "empty", "populated", "custom zero"} {
        t.Run("HTTP/"+mode, func(t *testing.T) {
            input := &plain.State{}
            if mode == "empty" {
                input.Text = pointer("")
                input.Bytes = []byte{}
                input.List = []string{}
                input.Mapping = map[string]string{}
                input.Child = &plain.Child{}
            } else if mode == "custom zero" {
                input.When = &time.Time{}
            } else if mode == "populated" {
                input.Text = pointer("value")
                input.Bytes = []byte("value")
                input.List = []string{"value"}
                input.Mapping = map[string]string{"key": "value"}
                input.Child = &plain.Child{Note: pointer("value")}
            }
            calls := 0
            var received *plain.State
            endpoints := &plain.Endpoints{Echo: func(_ context.Context, value any) (any, error) {
                calls++
                received = value.(*plain.State)
                return value, nil
            }}
            mux := loomhttp.NewMuxer()
            ps.Mount(mux, ps.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
            host := httptest.NewServer(mux)
            defer host.Close()
            client := pc.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
            _, err := client.Echo()(context.Background(), input)
            require.NoError(t, err)
            require.Equal(t, 1, calls)
            require.Equal(t, input, received)
        })
        t.Run("JSONRPC/"+mode, func(t *testing.T) {
            input := &rpc.State{}
            if mode == "empty" {
                input.Text = pointer("")
                input.Bytes = []byte{}
                input.List = []string{}
                input.Mapping = map[string]string{}
                input.Child = &rpc.Child{}
            } else if mode == "custom zero" {
                input.When = &time.Time{}
            } else if mode == "populated" {
                input.Text = pointer("value")
                input.Bytes = []byte("value")
                input.List = []string{"value"}
                input.Mapping = map[string]string{"key": "value"}
                input.Child = &rpc.Child{Note: pointer("value")}
            }
            calls := 0
            var received *rpc.State
            endpoints := &rpc.Endpoints{Echo: func(_ context.Context, value any) (any, error) {
                calls++
                received = value.(*rpc.State)
                return value, nil
            }}
            mux := loomhttp.NewMuxer()
            rs.Mount(mux, rs.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
            host := httptest.NewServer(mux)
            defer host.Close()
            client := rc.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
            _, err := client.Echo()(context.Background(), input)
            require.NoError(t, err)
            require.Equal(t, 1, calls)
            require.Equal(t, input, received)
        })
    }
}

func TestServerResponse(t *testing.T) {
    for _, transport := range []string{"HTTP", "JSONRPC"} {
        for _, mode := range []string{"absent", "empty", "populated", "custom zero"} {
            t.Run(transport+"/"+mode, func(t *testing.T) {
                calls := 0
                mux := loomhttp.NewMuxer()
                path, body := "/echo", "{}"
                if transport == "HTTP" {
                    endpoints := &plain.Endpoints{Echo: func(_ context.Context, _ any) (any, error) {
                        calls++
                        output := &plain.State{}
                        if mode == "empty" {
                            output = &plain.State{Text: pointer(""), Bytes: []byte{}, List: []string{}, Mapping: map[string]string{}, Child: &plain.Child{}}
                        } else if mode == "custom zero" {
                            output.When = &time.Time{}
                        } else if mode == "populated" {
                            output = &plain.State{Text: pointer("value"), Bytes: []byte("value"), List: []string{"value"}, Mapping: map[string]string{"key": "value"}, Child: &plain.Child{Note: pointer("value")}}
                        }
                        return output, nil
                    }}
                    ps.Mount(mux, ps.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
                } else {
                    endpoints := &rpc.Endpoints{Echo: func(_ context.Context, _ any) (any, error) {
                        calls++
                        output := &rpc.State{}
                        if mode == "empty" {
                            output = &rpc.State{Text: pointer(""), Bytes: []byte{}, List: []string{}, Mapping: map[string]string{}, Child: &rpc.Child{}}
                        } else if mode == "custom zero" {
                            output.When = &time.Time{}
                        } else if mode == "populated" {
                            output = &rpc.State{Text: pointer("value"), Bytes: []byte("value"), List: []string{"value"}, Mapping: map[string]string{"key": "value"}, Child: &rpc.Child{Note: pointer("value")}}
                        }
                        return output, nil
                    }}
                    rs.Mount(mux, rs.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
                    path, body = "/rpc", "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"echo\",\"params\":{}}"
                }
                req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
                req.Header.Set("Content-Type", "application/json")
                res := httptest.NewRecorder()
                mux.ServeHTTP(res, req)
                require.Equal(t, http.StatusOK, res.Code, res.Body.String())
                require.Equal(t, 1, calls)
                wire, err := io.ReadAll(res.Body)
                require.NoError(t, err)
                var fields map[string]jsontext.Value
                require.NoError(t, json.Unmarshal(wire, &fields))
                if transport == "JSONRPC" {
                    require.Contains(t, fields, "result")
                    result := fields["result"]
                    fields = nil
                    require.NoError(t, json.Unmarshal(result, &fields))
                }
                if mode == "custom zero" {
                    require.Contains(t, fields, "when")
                    require.JSONEq(t, "\"0001-01-01T00:00:00Z\"", string(fields["when"]))
                    return
                }
                if mode == "absent" {
                    require.Empty(t, fields)
                    return
                }
                for _, name := range []string{"text", "bytes", "list", "mapping", "child"} {
                    t.Run(name, func(t *testing.T) {
                        require.Contains(t, fields, name, string(wire))
                    })
                }
                if mode == "empty" {
                    for name, value := range map[string]string{"text": "\"\"", "bytes": "\"\"", "list": "[]", "mapping": "{}", "child": "{}"} {
                        t.Run(name+" value", func(t *testing.T) {
                            require.Contains(t, fields, name, string(wire))
                            require.JSONEq(t, value, string(fields[name]))
                        })
                    }
                }
            })
        }
    }
}

func TestNativeServiceCustomCodec(t *testing.T) {
    for _, tc := range []struct {
        name string
        value any
        want string
    }{
        {"HTTP absent", &plain.State{}, "{}"},
        {"HTTP supplied custom zero", &plain.State{When: &time.Time{}}, "{\"when\":\"0001-01-01T00:00:00Z\"}"},
        {"JSONRPC absent", &rpc.State{}, "{}"},
        {"JSONRPC supplied custom zero", &rpc.State{When: &time.Time{}}, "{\"when\":\"0001-01-01T00:00:00Z\"}"},
    } {
        t.Run(tc.name, func(t *testing.T) {
            wire, err := json.Marshal(tc.value)
            require.NoError(t, err)
            require.JSONEq(t, tc.want, string(wire))
        })
    }
}

func pointer[T any](value T) *T {
    return &value
}
`

func TestJSONTagNormalizationGenerated(t *testing.T) {
	runDesignHarness(t, "example.com/omission", func() {
		jsonTagNormalizationDSL(true)
	}, jsonTagNormalizationHarness)
}

func jsonTagNormalizationDSL(fullTag bool) {
	dsl.API("omission", func() {
		dsl.JSONRPC(func() {
		})
	})
	child := dsl.Type("Child", func() {
		dsl.Attribute("note", dsl.String)
	})
	state := dsl.Type("State", func() {
		dsl.Attribute("text", dsl.String, func() {
			if fullTag {
				dsl.Meta("struct:tag:json", "text,omitzero")
			}
		})
		dsl.Attribute("bytes", dsl.Bytes, func() {
			if fullTag {
				dsl.Meta("struct:tag:json", "bytes,omitzero")
			}
		})
		dsl.Attribute("list", dsl.ArrayOf(dsl.String), func() {
			if fullTag {
				dsl.Meta("struct:tag:json", "list,omitzero")
			}
		})
		dsl.Attribute("mapping", dsl.MapOf(dsl.String, dsl.String), func() {
			if fullTag {
				dsl.Meta("struct:tag:json", "mapping,omitzero")
			}
		})
		dsl.Attribute("child", child, func() {
			if fullTag {
				dsl.Meta("struct:tag:json", "child,omitzero")
			}
		})
		dsl.Attribute("when", dsl.String, func() {
			dsl.Meta("struct:field:type", "time.Time", "time")
		})
	})
	for _, name := range []string{"plain", "rpc"} {
		dsl.Service(name, func() {
			if name == "rpc" {
				dsl.JSONRPC(func() {
					dsl.POST("/rpc")
				})
			}
			dsl.Method("echo", func() {
				dsl.Payload(state)
				dsl.Result(state)
				if name == "rpc" {
					dsl.JSONRPC(func() {
					})
				} else {
					dsl.HTTP(func() {
						dsl.POST("/echo")
					})
				}
			})
		})
	}
}
