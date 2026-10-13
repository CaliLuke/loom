package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

const headerCollectionPhysicalHarness = `package headerrepresentation_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	loomhttp "github.com/CaliLuke/loom/http"
	svc "example.com/headerrepresentation/gen/headers"
	client "example.com/headerrepresentation/gen/http/headers/client"
	server "example.com/headerrepresentation/gen/http/headers/server"
	located "example.com/headerrepresentation/gen/types/located"
)

func TestNamedCollectionRoundTrip(t *testing.T) {
	for _, value := range []*svc.Values{
		{},
		{
			Label:    []svc.LabelValue{},
			Flag:     []svc.FlagValue{},
			Signed:   []svc.SignedValue{},
			Unsigned: []svc.UnsignedValue{},
			Real:     []svc.RealValue{},
			Bytes:    []svc.BytesValue{},
			Outer:    svc.Labels{},
			Located:  []located.Located{},
		},
		{
			Label:    []svc.LabelValue{"alpha", "beta"},
			Flag:     []svc.FlagValue{false, true},
			Signed:   []svc.SignedValue{-3, 8},
			Unsigned: []svc.UnsignedValue{0, 9},
			Real:     []svc.RealValue{1.5, -2},
			Bytes:    []svc.BytesValue{[]byte("a"), []byte("b")},
			Outer:    svc.Labels{"outer"},
			Located:  []located.Located{"located"},
		},
	} {
		calls := make(chan any, 1)
		mux := loomhttp.NewMuxer()
		endpoints := &svc.Endpoints{Echo: func(ctx context.Context, request any) (any, error) {
			calls <- request
			return request, nil
		}}
		server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
		host := httptest.NewServer(mux)
		t.Cleanup(host.Close)
		u, err := url.Parse(host.URL)
		require.NoError(t, err)
		c := client.NewClient(u.Scheme, u.Host, host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
		got, err := c.Echo()(context.Background(), value)
		require.NoError(t, err)
		require.Equal(t, value, got)
		select {
		case request := <-calls:
			require.Equal(t, value, request)
		default:
			t.Error("service was not invoked")
		}
	}
}

func TestNamedCollectionRejectsBeforeWrite(t *testing.T) {
	for _, item := range []svc.LabelValue{"", "a,b", " left", "right\t"} {
		value := &svc.Values{Label: []svc.LabelValue{item}}
		request, err := http.NewRequest(http.MethodGet, "http://example.com/echo", nil)
		require.NoError(t, err)
		err = client.EncodeEchoRequest(loomhttp.RequestEncoder)(request, value)
		require.Error(t, err)
		require.Empty(t, request.Header.Values("X-Label"))
		recorder := httptest.NewRecorder()
		err = server.EncodeEchoResponse(loomhttp.ResponseEncoder)(context.Background(), recorder, value)
		require.Error(t, err)
		require.Empty(t, recorder.Header().Values("X-Label"))
		require.Empty(t, recorder.Body.String())
	}
}
`

// TestHeaderCollectionPhysicalTypes compiles both directions with native and
// named element carriers, including named outer collections and located types.
func TestHeaderCollectionPhysicalTypes(t *testing.T) {
	for _, named := range []bool{false, true} {
		name := "native"
		if named {
			name = "named"
		}
		t.Run(name, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				headerCollectionPhysicalDSL(named)
			})
			dir := t.TempDir()
			if output := os.Getenv("LOOM_HEADER_REPRESENTATION_OUTPUT"); output != "" {
				dir = filepath.Join(output, t.Name())
				require.NoError(t, os.MkdirAll(dir, 0750))
			}
			renderHTTPModule(t, dir, "example.com/headerrepresentation", root)
			if os.Getenv("LOOM_HEADER_REPRESENTATION_GENERATE_ONLY") != "" {
				return
			}
			if named {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "header_test.go"), []byte(headerCollectionPhysicalHarness), 0600))
			}
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "test", "-race", "./...")
			runGoCommand(t, dir, "vet", "./...")
		})
	}
}

// TestHeaderCollectionPhysicalMetadata checks that the shared analysis already
// preserves both wire and declared carriers for the encoding owner to consume.
func TestHeaderCollectionPhysicalMetadata(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		headerCollectionPhysicalDSL(true)
	})
	endpoint := CreateHTTPServices(root).Get("headers").Endpoints[0]
	for _, headers := range [][]*HeaderData{
		endpoint.Payload.Request.Headers,
		endpoint.Result.Responses[0].Headers,
	} {
		require.Len(t, headers, 8)
		for _, header := range headers {
			wire := expr.AsArray(header.Type)
			physical := expr.AsArray(header.FieldType)
			require.NotNil(t, wire)
			require.NotNil(t, physical)
			_, wireNamed := wire.ElemType.Type.(expr.UserType)
			_, physicalNamed := physical.ElemType.Type.(expr.UserType)
			require.False(t, wireNamed, header.Name)
			require.True(t, physicalNamed, header.Name)
			require.Equal(t, physical.ElemType.Type.(expr.UserType).Attribute().Type, wire.ElemType.Type)
		}
	}
}

func headerCollectionPhysicalDSL(named bool) {
	types := []expr.DataType{String, Boolean, Int64, UInt64, Float64, Bytes}
	names := []string{"label", "flag", "signed", "unsigned", "real", "bytes"}
	if named {
		for i, primitive := range types {
			types[i] = Type(names[i]+"Value", primitive)
		}
	}
	var located expr.DataType = String
	if named {
		located = Type("Located", String, func() {
			Meta("struct:pkg:path", "types/located")
		})
	}
	outer := Type("Labels", ArrayOf(types[0]))
	values := Type("Values", func() {
		for i, typ := range types {
			Attribute(names[i], ArrayOf(typ))
		}
		Attribute("outer", outer)
		Attribute("located", ArrayOf(located))
	})
	Service("headers", func() {
		Method("echo", func() {
			Payload(values)
			Result(values)
			HTTP(func() {
				GET("/echo")
				for _, name := range names {
					Header(name + ":X-" + name)
				}
				Header("outer:X-Outer")
				Header("located:X-Located")
				Response(StatusOK, func() {
					for _, name := range names {
						Header(name + ":X-" + name)
					}
					Header("outer:X-Outer")
					Header("located:X-Located")
				})
			})
		})
	})
}
