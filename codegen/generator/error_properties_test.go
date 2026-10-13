package generator

import (
	"fmt"
	"testing"

	d "github.com/CaliLuke/loom/dsl"
)

const errorPropertiesHarness = `package errorproperties

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	web "example.com/errorproperties/gen/web"
	wc "example.com/errorproperties/gen/http/web/client"
	ws "example.com/errorproperties/gen/http/web/server"
	rpc "example.com/errorproperties/gen/rpc"
	rc "example.com/errorproperties/gen/jsonrpc/rpc/client"
	rs "example.com/errorproperties/gen/jsonrpc/rpc/server"
	protosvc "example.com/errorproperties/gen/proto"
	gc "example.com/errorproperties/gen/grpc/proto/client"
	gs "example.com/errorproperties/gen/grpc/proto/server"
	pb "example.com/errorproperties/gen/grpc/proto/pb"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type captureDoer struct {
	client *http.Client
	body []byte
	status int
	mutate func([]byte) []byte
}

func TestErrorProperties(t *testing.T) {
	for _, transport := range []string{"web", "rpc", "proto"} {
		t.Run(transport, func(t *testing.T) {
			var source *loom.ServiceError
			calls := 0
			fail := func(context.Context, any) (any, error) {
				calls++
				return nil, source
			}
			var endpoint loom.Endpoint
			var capture *captureDoer
			if transport != "proto" {
				mux := loomhttp.NewMuxer()
				if transport == "web" {
					ws.Mount(mux, ws.New(&web.Endpoints{Fail: fail}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
				} else {
					rs.Mount(mux, rs.New(&rpc.Endpoints{Fail: fail}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
				}
				host := httptest.NewServer(mux)
				t.Cleanup(host.Close)
				capture = &captureDoer{client: host.Client()}
				address := strings.TrimPrefix(host.URL, "http://")
				if transport == "web" {
					endpoint = wc.NewClient("http", address, capture, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false).Fail()
				} else {
					endpoint = rc.NewClient("http", address, capture, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false).Fail()
				}
			} else {
				listener := bufconn.Listen(1024 * 1024)
				server := grpc.NewServer()
				pb.RegisterProtoServer(server, gs.New(&protosvc.Endpoints{Fail: fail}, nil))
				done := make(chan error, 1)
				go func() {
					done <- server.Serve(listener)
				}()
				t.Cleanup(func() {
					server.Stop()
					require.NoError(t, <-done)
				})
				conn, err := grpc.NewClient("passthrough:///bufconn", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
					return listener.Dial()
				}))
				require.NoError(t, err)
				t.Cleanup(func() {
					require.NoError(t, conn.Close())
				})
				endpoint = gc.NewClient(conn).Fail()
			}
			for bits := range 8 {
				t.Run(fmt.Sprintf("properties%d", bits), func(t *testing.T) {
					constructors := []func(error) *loom.ServiceError{web.MakeError0, web.MakeError1, web.MakeError2, web.MakeError3, web.MakeError4, web.MakeError5, web.MakeError6, web.MakeError7}
					if transport == "rpc" {
						constructors = []func(error) *loom.ServiceError{rpc.MakeError0, rpc.MakeError1, rpc.MakeError2, rpc.MakeError3, rpc.MakeError4, rpc.MakeError5, rpc.MakeError6, rpc.MakeError7}
					} else if transport == "proto" {
						constructors = []func(error) *loom.ServiceError{protosvc.MakeError0, protosvc.MakeError1, protosvc.MakeError2, protosvc.MakeError3, protosvc.MakeError4, protosvc.MakeError5, protosvc.MakeError6, protosvc.MakeError7}
					}
					source = constructors[bits](errors.New("safe detail"))
					require.Equal(t, bits&1 != 0, source.Timeout)
					require.Equal(t, bits&2 != 0, source.Temporary)
					require.Equal(t, bits&4 != 0, source.Fault)
					source.ID = "error-instance"
					before := calls
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					_, err := endpoint(ctx, nil)
					require.Equal(t, before+1, calls)
					var received *loom.ServiceError
					require.ErrorAs(t, err, &received)
					require.Equal(t, source.Name, received.Name)
					require.Equal(t, source.ID, received.ID)
					require.Equal(t, source.Message, received.Message)
					for name, values := range map[string][2]bool{
						"timeout": {source.Timeout, received.Timeout},
						"temporary": {source.Temporary, received.Temporary},
						"fault": {source.Fault, received.Fault},
					} {
						t.Run("decoded/"+name, func(t *testing.T) {
							require.Equal(t, values[0], values[1])
						})
					}
					if capture == nil {
						return
					}
					var document map[string]jsontext.Value
					require.NoError(t, json.Unmarshal(capture.body, &document))
					if transport == "rpc" {
						require.Equal(t, http.StatusOK, capture.status)
						var failure map[string]jsontext.Value
						require.NoError(t, json.Unmarshal(document["error"], &failure))
						var code int
						require.NoError(t, json.Unmarshal(failure["code"], &code))
						require.Equal(t, -32001-bits, code)
						require.NoError(t, json.Unmarshal(failure["data"], &document))
					} else {
						require.Equal(t, []int{500, 409, 410, 411, 412, 501, 503, 504}[bits], capture.status)
					}
					for name, expected := range map[string]bool{"timeout": source.Timeout, "temporary": source.Temporary, "fault": source.Fault} {
						t.Run(name, func(t *testing.T) {
							value, exists := document[name]
							require.True(t, exists, "full problem body omits %s", name)
							var actual bool
							require.NoError(t, json.Unmarshal(value, &actual))
							require.Equal(t, expected, actual)
						})
					}
				})
			}
			if capture != nil {
				for _, property := range []string{"timeout", "temporary", "fault"} {
					for _, mutation := range []string{"missing", "null", "string"} {
						t.Run(property+"/"+mutation, func(t *testing.T) {
							capture.mutate = func(body []byte) []byte {
								var envelope map[string]any
								require.NoError(t, json.Unmarshal(body, &envelope))
								document := envelope
								if transport == "rpc" {
									document = envelope["error"].(map[string]any)["data"].(map[string]any)
								}
								switch mutation {
								case "missing":
									delete(document, property)
								case "null":
									document[property] = nil
								case "string":
									document[property] = "true"
								}
								encoded, err := json.Marshal(envelope)
								require.NoError(t, err)
								return encoded
							}
							before := calls
							result, err := endpoint(t.Context(), nil)
							require.Equal(t, before+1, calls)
							require.Nil(t, result)
							var failure *loomhttp.ClientError
							require.ErrorAs(t, err, &failure)
							kind := "validation_error"
							if mutation == "string" {
								kind = "decoding_error"
							}
							require.Equal(t, kind, failure.Name)
						})
					}
				}
			}
		})
	}
}
func (d *captureDoer) Do(req *http.Request) (*http.Response, error) {
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	body, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return nil, err
	}
	d.body, d.status = body, resp.StatusCode
	if d.mutate != nil {
		body = d.mutate(body)
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return resp, nil
}

`

func TestDeclaredErrorPropertiesGenerated(t *testing.T) {
	runDesignHarness(t, "example.com/errorproperties", errorPropertiesDSL, errorPropertiesHarness)
}

func errorPropertiesDSL() {
	d.API("errorproperties", func() {
		d.JSONRPC(func() {
		})
	})
	for _, transport := range []string{"web", "rpc", "proto"} {
		d.Service(transport, func() {
			if transport == "rpc" {
				d.JSONRPC(func() {
					d.POST("/rpc")
				})
			}
			d.Method("fail", func() {
				for bits := range 8 {
					d.Error(fmt.Sprintf("error%d", bits), d.ErrorResult, func() {
						if bits&1 != 0 {
							d.Timeout()
						}
						if bits&2 != 0 {
							d.Temporary()
						}
						if bits&4 != 0 {
							d.Fault()
						}
					})
				}
				switch transport {
				case "web":
					d.HTTP(func() {
						d.POST("/fail")
						for bits, status := range []int{500, 409, 410, 411, 412, 501, 503, 504} {
							d.Response(fmt.Sprintf("error%d", bits), status)
						}
					})
				case "rpc":
					d.JSONRPC(func() {
						for bits := range 8 {
							d.Response(fmt.Sprintf("error%d", bits), -32001-bits)
						}
					})
				case "proto":
					d.GRPC(func() {
						for bits := range 8 {
							d.Response(fmt.Sprintf("error%d", bits), d.CodeUnknown)
						}
					})
				}
			})
		})
	}
}
