package generator

import (
	"testing"

	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestJSONFieldStorageGenerated(t *testing.T) {
	runDesignHarness(t, "example.com/storage", jsonFieldStorageDSL, jsonFieldStorageHarness)
}

func jsonFieldStorageDSL() {
	dsl.API("storage", func() {
		dsl.JSONRPC(func() {
		})
	})
	names := dsl.Type("Names", dsl.ArrayOf(dsl.String))
	labels := dsl.Type("Labels", dsl.MapOf(dsl.String, dsl.String))
	var node expr.UserType
	node = dsl.Type("Node", func() {
		dsl.Attribute("names", names)
		dsl.Attribute("labels", labels)
		dsl.Attribute("next", node)
	})
	state := dsl.Type("State", func() {
		dsl.Attribute("flag", dsl.Boolean, func() {
			dsl.Default(true)
		})
		dsl.Attribute("count", dsl.Int, func() {
			dsl.Default(7)
		})
		dsl.Attribute("text", dsl.String, func() {
			dsl.Default("fallback")
		})
		dsl.Attribute("node", node)
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

const jsonFieldStorageHarness = `package storage_test

import (
 "context"
 "encoding/json/v2"
 "net/http/httptest"
 "strings"
 "testing"
 plain "example.com/storage/gen/plain"
 rpc "example.com/storage/gen/rpc"
 pc "example.com/storage/gen/http/plain/client"
 ps "example.com/storage/gen/http/plain/server"
 rc "example.com/storage/gen/jsonrpc/rpc/client"
 rs "example.com/storage/gen/jsonrpc/rpc/server"
 loomhttp "github.com/CaliLuke/loom/http"
 "github.com/stretchr/testify/require"
)

func TestStorageRoundTrip(t *testing.T) {
 for _, transport := range []string{"HTTP", "RPC"} {
  t.Run(transport, func(t *testing.T) {
   mux := loomhttp.NewMuxer()
   calls := 0
   var input any
   var invoke func(context.Context, any) (any, error)
   host := httptest.NewServer(mux)
   defer host.Close()
   if transport == "HTTP" {
    value := &plain.State{Node: &plain.Node{Names: plain.Names{}, Labels: plain.Labels{}, Next: &plain.Node{Names: plain.Names{}, Labels: plain.Labels{}}}}
    input = value
    endpoints := &plain.Endpoints{Echo: func(_ context.Context, payload any) (any, error) {
     calls++
     require.Equal(t, value, payload)
     return value, nil
    }}
    ps.Mount(mux, ps.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
    invoke = pc.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false).Echo()
   } else {
    value := &rpc.State{Node: &rpc.Node{Names: rpc.Names{}, Labels: rpc.Labels{}, Next: &rpc.Node{Names: rpc.Names{}, Labels: rpc.Labels{}}}}
    input = value
    endpoints := &rpc.Endpoints{Echo: func(_ context.Context, payload any) (any, error) {
     calls++
     require.Equal(t, value, payload)
     return value, nil
    }}
    rs.Mount(mux, rs.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
    invoke = rc.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false).Echo()
   }
   wire, err := json.Marshal(input)
   require.NoError(t, err)
   require.JSONEq(t, "{\"flag\":false,\"count\":0,\"text\":\"\",\"node\":{\"names\":[],\"labels\":{},\"next\":{\"names\":[],\"labels\":{}}}}", string(wire))
   output, err := invoke(context.Background(), input)
   require.NoError(t, err)
   require.Equal(t, 1, calls)
   require.Equal(t, input, output)
  })
 }
}
`
