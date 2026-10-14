package codegen

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/stretchr/testify/require"
)

func TestSelectedResponseProjectionReadiness(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		API("selectedresponse", func() {
			JSONRPC(func() {
			})
		})

		record := Type("SelectedRecord", func() {
			Attribute("text", String)
			Required("text")
		})

		Service("fields", func() {
			JSONRPC(func() {
				POST("/")
			})
			Method("call", func() {
				Result(func() {
					Attribute("value", String)
					Attribute("internal", String)
					Required("value")
				})
				JSONRPC(func() {
					Response(StatusOK, func() {
						Body("value")
					})
				})
			})
			for _, tc := range []struct {
				name     string
				datatype any
			}{
				{"meta", String}, {"list", ArrayOf(String)}, {"record", record},
			} {
				Method(tc.name, func() {
					Result(func() {
						Attribute("value", tc.datatype)
						Attribute("internal", String)
						Required("value")
						if tc.name == "meta" {
							Attribute("header", String)
							Attribute("cookie", String)
							Required("header", "cookie")
						}
					})
					JSONRPC(func() {
						Response(StatusOK, func() {
							if tc.name == "meta" {
								Body(func() {
									Attribute("value")
									Attribute("header")
									Attribute("cookie")
								})
							} else {
								Body("value")
							}
						})
					})
				})
			}

			Method("whole", func() {
				Result(func() {
					Attribute("value", String)
					Attribute("internal", String)
					Required("value", "internal")
				})
				JSONRPC(func() {
				})
			})
		})
	})
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/selectedresponse", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "response_test.go"), []byte(selectedResponseProjectionReadinessHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "build", "./...")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}

const selectedResponseProjectionReadinessHarness = `package selectedresponse_test

import (
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	fields "example.com/selectedresponse/gen/fields"
	"example.com/selectedresponse/gen/jsonrpc/fields/client"
	"example.com/selectedresponse/gen/jsonrpc/fields/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loomrpc "github.com/CaliLuke/loom/jsonrpc"
)

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestSelectedResponseWire(t *testing.T) {
	for _, tc := range []struct {
		method, value, want string
	}{
		{"call", "hello", "\"hello\""},
		{"call", "", "\"\""},
		{"whole", "hello", "{\"value\":\"hello\",\"internal\":\"private-sentinel\"}"},
	} {
		t.Run(tc.method+"/"+tc.value, func(t *testing.T) {
			called := false
			endpoints := &fields.Endpoints{
				Call: func(context.Context, any) (any, error) {
					called = true
					return &fields.CallResult{Value: tc.value, Internal: new("private-sentinel")}, nil
				},
				Whole: func(context.Context, any) (any, error) {
					called = true
					return &fields.WholeResult{Value: tc.value, Internal: "private-sentinel"}, nil
				},
			}
			mux := loomhttp.NewMuxer()
			server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil).Mount(mux)
			r := httptest.NewRequest(http.MethodPost, "http://example.test/", strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":\"request\",\"method\":\""+tc.method+"\"}"))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Code != http.StatusOK || !called {
				t.Fatalf("status=%d called=%v body=%s", w.Code, called, w.Body.String())
			}
			var envelope struct {
				ID     string         "json:\"id\""
				Result jsontext.Value "json:\"result\""
			}
			if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if envelope.ID != "request" || string(envelope.Result) != tc.want {
				t.Errorf("response=%s; want id=request result=%s", w.Body.String(), tc.want)
			}
		})
	}
}

func TestSelectedResponseEncoderControl(t *testing.T) {
	w := httptest.NewRecorder()
	result := &fields.CallResult{Value: "hello", Internal: new("private-sentinel")}
	if err := server.EncodeCallResponse(loomhttp.ResponseEncoder)(context.Background(), w, result); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(w.Body.String()) != "\"hello\"" {
		t.Errorf("selected response encoder=%s", w.Body.String())
	}
}

func TestSelectedResponseClient(t *testing.T) {
	mux := loomhttp.NewMuxer()
	endpoints := &fields.Endpoints{Call: func(context.Context, any) (any, error) {
		return &fields.CallResult{Value: "hello", Internal: new("private-sentinel")}, nil
	}}
	server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil).Mount(mux)
	doer := doerFunc(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w.Result(), nil
	})
	result, err := client.NewClient("http", "example.test", doer, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false).Call()(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.(*fields.CallResult).Value != "hello" {
		t.Errorf("result=%#v", result)
	}
}

func TestProjectionFramingAndCodecBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, request, method, want string
		calls, invocations          int
		fail                        bool
	}{
		{"ordinary", "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"call\"}", "call", "\"hello\"", 1, 1, false},
		{"metadata", "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"meta\"}", "meta", "{\"value\":\"hello\",\"header\":\"public-tag\",\"cookie\":\"public-cookie\"}", 1, 1, false},
		{"empty list", "{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"list\"}", "list", "[]", 1, 1, false},
		{"object", "{\"jsonrpc\":\"2.0\",\"id\":4,\"method\":\"record\"}", "record", "{\"text\":\"visible\"}", 1, 1, false},
		{"notification", "{\"jsonrpc\":\"2.0\",\"method\":\"call\"}", "call", "", 0, 1, false},
		{"metadata notification", "{\"jsonrpc\":\"2.0\",\"method\":\"meta\"}", "meta", "", 0, 1, false},
		{"batch", "[{\"jsonrpc\":\"2.0\",\"id\":\"one\",\"method\":\"call\"},{\"jsonrpc\":\"2.0\",\"method\":\"call\"},{\"jsonrpc\":\"2.0\",\"id\":\"two\",\"method\":\"list\"}]", "batch", "", 2, 3, false},
		{"encode failure", "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"call\"}", "call", "", 1, 1, true},
		{"object encode failure", "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"meta\"}", "meta", "", 1, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls, invoked, failures := 0, 0, 0
			endpoints := &fields.Endpoints{
				Call: func(context.Context, any) (any, error) {
					invoked++
					return &fields.CallResult{Value: "hello", Internal: new("private-sentinel")}, nil
				},
				Meta: func(context.Context, any) (any, error) {
					invoked++
					return &fields.MetaResult{Value: "hello", Internal: new("private-sentinel"), Header: "public-tag", Cookie: "public-cookie"}, nil
				},
				List: func(context.Context, any) (any, error) {
					invoked++
					return &fields.ListResult{Value: []string{}, Internal: new("private-sentinel")}, nil
				},
				Record: func(context.Context, any) (any, error) {
					invoked++
					return &fields.RecordResult{Value: &fields.SelectedRecord{Text: "visible"}, Internal: new("private-sentinel")}, nil
				},
			}
			encoder := func(ctx context.Context, w http.ResponseWriter) loomhttp.Encoder {
				base := loomhttp.ResponseEncoder(ctx, w)
				return loomhttp.EncodingFunc(func(v any) error {
					calls++
					if _, ok := v.(*loomrpc.Response); !ok {
						t.Errorf("envelope input=%T", v)
					}
					if tc.fail {
						return errors.New("encoder-probe")
					}
					return base.Encode(v)
				})
			}
			mux := loomhttp.NewMuxer()
			server.New(endpoints, mux, loomhttp.RequestDecoder, encoder, func(_ context.Context, _ http.ResponseWriter, err error) {
				failures++
				if !strings.Contains(err.Error(), "encoder-probe") {
					t.Errorf("unexpected encoding failure=%v", err)
				}
			}).Mount(mux)
			req := httptest.NewRequest(http.MethodPost, "http://example.test/", strings.NewReader(tc.request))
			req.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, req)
			if calls != tc.calls || invoked != tc.invocations || failures != boolInt(tc.fail) {
				t.Errorf("calls=%d invoked=%d failures=%d body=%s", calls, invoked, failures, response.Body.String())
			}
			if strings.Contains(response.Body.String(), "private-sentinel") {
				t.Errorf("excluded sibling leaked: %s", response.Body.String())
			}
			if tc.fail || !strings.Contains(tc.request, "\"id\"") {
				if response.Body.Len() != 0 {
					t.Errorf("unexpected response: %s", response.Body.String())
				}
				return
			}
			if response.Code != http.StatusOK {
				t.Errorf("status=%d", response.Code)
			}
			if tc.method == "batch" {
				var got []struct {
					ID     string         "json:\"id\""
					Result jsontext.Value "json:\"result\""
				}
				if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				if len(got) != 2 || got[0].ID != "one" || string(got[0].Result) != "\"hello\"" || got[1].ID != "two" || string(got[1].Result) != "[]" {
					t.Errorf("batch=%s", response.Body.String())
				}
				return
			}
			var got struct {
				Result jsontext.Value "json:\"result\""
			}
			if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if string(got.Result) != tc.want {
				t.Errorf("result=%s want=%s", got.Result, tc.want)
			}
			if response.Header().Get("X-Selected") != "" || len(response.Header().Values("Set-Cookie")) != 0 {
				t.Errorf("metadata=%v", response.Header())
			}
		})
	}
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
`

func TestSelectedStreamProjectionInventory(t *testing.T) {
	for _, transport := range []string{"sse", "websocket"} {
		t.Run(transport, func(t *testing.T) {
			root := RunJSONRPCDSL(t, func() {
				API("selectedstream", func() {
					JSONRPC(func() {
					})
				})
				Service("events", func() {
					JSONRPC(func() {
						if transport == "websocket" {
							GET("/rpc")
						} else {
							POST("/rpc")
						}
					})
					Method("watch", func() {
						if transport == "websocket" {
							StreamingPayload(String)
						} else {
							StreamingResult(String)
						}
						Result(func() {
							Attribute("value", String)
							Attribute("internal", String)
							Required("value")
						})
						JSONRPC(func() {
							if transport == "sse" {
								ServerSentEvents()
							}
							Response(StatusOK, func() {
								Body("value")
							})
						})
					})
				})
			})
			data := CreateJSONRPCServices(root)
			for _, name := range []string{"events"} {
				endpoint := data.Get(name).Endpoints[0]
				response := endpoint.Result.Responses[0]
				t.Logf("selected=%s body=%d init=%v", response.ResultAttr, len(response.ServerBody), response.ServerBody[0].Init != nil)
			}
			dir := t.TempDir()
			if base := os.Getenv("LOOM_PROJECTION_ARTIFACTS"); base != "" {
				dir = filepath.Join(base, transport)
				require.NoError(t, os.MkdirAll(dir, 0o750))
			}
			renderJSONRPCModule(t, dir, "example.com/selectedstream", root)
			runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
			runGoJSONRPCTestCommand(t, dir, "build", "./...")
			runGoJSONRPCTestCommand(t, dir, "vet", "./...")
			if transport == "websocket" {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "response_test.go"), []byte(selectedWebSocketProjectionHarness), 0o600))
				runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
			}
		})
	}
}

const selectedWebSocketProjectionHarness = `package selectedstream_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	events "example.com/selectedstream/gen/events"
	server "example.com/selectedstream/gen/jsonrpc/events/server"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/gorilla/websocket"
)

type implementation struct{}

func (implementation) Watch(_ context.Context, value string) (*events.WatchResult, error) {
	return &events.WatchResult{Value: value, Internal: new("private-sentinel")}, nil
}
func (implementation) HandleStream(ctx context.Context, stream events.Stream) error {
	if err := stream.SendWatchNotification(ctx, &events.WatchResult{Value: "", Internal: new("private-sentinel")}); err != nil {
		return err
	}
	for {
		if err := stream.Recv(ctx); err != nil {
			return err
		}
	}
}
func TestSelectedWebSocketResponse(t *testing.T) {
	mux := loomhttp.NewMuxer()
	svc := implementation{}
	server.Mount(mux, server.New(svc.HandleStream, events.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, &websocket.Upgrader{}, loomhttp.WebSocketConfig{}))
	host := httptest.NewServer(mux)
	defer host.Close()
	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(host.URL, "http")+"/rpc", nil)
	if response != nil && response.Body != nil {
		if err := response.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	_, notification, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(notification), "\"params\":\"\"") || !strings.Contains(string(notification), "\"method\":\"watch\"") || strings.Contains(string(notification), "\"id\"") || strings.Contains(string(notification), "private-sentinel") {
		t.Errorf("selected notification=%s", notification)
	}
	for _, value := range []string{"hello", ""} {
		request := "{\"jsonrpc\":\"2.0\",\"id\":\"request\",\"method\":\"watch\",\"params\":\"" + value + "\"}"
		if err := conn.WriteMessage(websocket.TextMessage, []byte(request)); err != nil {
			t.Fatal(err)
		}
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "\"result\":\""+value+"\"") || !strings.Contains(string(data), "\"id\":\"request\"") || strings.Contains(string(data), "private-sentinel") {
			t.Errorf("selected response=%s", data)
		}
	}
}
`
