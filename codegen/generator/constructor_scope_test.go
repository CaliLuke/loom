package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/testingx"
)

const constructorNominalHarness = `package constructorscope_test

import (
	"context"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	client "example.com/constructorscope/gen/http/labels/client"
	server "example.com/constructorscope/gen/http/labels/server"
	svc "example.com/constructorscope/gen/labels"
	located "example.com/constructorscope/gen/types/labels"
	loomhttp "github.com/CaliLuke/loom/http"
)

type echoService struct {
}

func TestConstructorRoundTrip(t *testing.T) {
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(svc.NewEndpoints(echoService{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	host := httptest.NewServer(mux)
	t.Cleanup(host.Close)
	u, err := url.Parse(host.URL)
	require.NoError(t, err)
	c := client.NewClient(u.Scheme, u.Host, host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	label, place := svc.ALIAS("value"), located.Located("located")
	for _, items := range [][]svc.ALIAS{nil, {}, {"a", "b"}} {
		payload := &svc.Envelope{Body: "body", Value: &label, Items: items, Located: &place}
		result, err := c.Echo()(context.Background(), payload)
		require.NoError(t, err)
		require.Equal(t, payload, result)
	}
}

func (echoService) Echo(_ context.Context, payload *svc.Envelope) (*svc.Envelope, error) {
	return payload, nil
}
`

// TestConstructorNominalScopeGenerated checks the shared constructor's nominal
// casts in HTTP headers/query, gRPC metadata and shared CLI builders.
func TestConstructorNominalScopeGenerated(t *testing.T) {
	for _, reserved := range []bool{false, true} {
		name := map[bool]string{false: "ordinary", true: "reserved"}[reserved]
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if base := os.Getenv("LOOM_CONSTRUCTOR_SCOPE_OUTPUT"); base != "" {
				dir = filepath.Join(base, name)
				require.NoError(t, os.MkdirAll(dir, 0o750))
			}
			renderDesignModule(t, dir, "example.com/constructorscope", func() {
				constructorNominalScopeDSL(reserved)
			})
			if os.Getenv("LOOM_CONSTRUCTOR_SCOPE_GENERATE_ONLY") == "1" {
				return
			}
			alias := "Label"
			if reserved {
				alias = "Client2"
			}
			harness := strings.ReplaceAll(constructorNominalHarness, "ALIAS", alias)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "constructor_test.go"), []byte(harness), 0o600))
			for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}, {"test", "-race", "-count=1", "."}} {
				output, err := testingx.RunCmd(dir, "go", args...)
				require.NoError(t, err, output)
			}
		})
	}
}

func constructorNominalScopeDSL(reserved bool) {
	dsl.API("constructorscope", func() {
		dsl.JSONRPC(func() {
		})
	})
	name := "Label"
	if reserved {
		name = "Client"
	}
	label := dsl.Type(name, dsl.String)
	located := dsl.Type("Located", dsl.String, func() {
		dsl.Meta("struct:pkg:path", "types/labels")
	})
	value := dsl.Type("Envelope", func() {
		dsl.Field(1, "body", dsl.String)
		dsl.Field(2, "value", label)
		dsl.Field(3, "items", dsl.ArrayOf(label))
		dsl.Field(4, "located", located)
		dsl.Required("body")
	})
	dsl.Service("labels", func() {
		dsl.Method("echo", func() {
			dsl.Payload(value)
			dsl.Result(value)
			dsl.HTTP(func() {
				dsl.POST("/labels")
				dsl.Param("value")
				dsl.Header("items:X-Items")
				dsl.Header("located:X-Located")
				dsl.Body("body")
				dsl.Response(dsl.StatusOK, func() {
					dsl.Header("value:X-Value")
					dsl.Header("located:X-Located")
				})
			})
			dsl.GRPC(func() {
				dsl.Metadata(constructorNominalMetadata)
				dsl.Response(dsl.CodeOK, func() {
					dsl.Headers(constructorNominalMetadata)
				})
			})
		})
	})
	dsl.Service("rpc-labels", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("echo", func() {
			dsl.Payload(value)
			dsl.Result(value)
			dsl.JSONRPC(func() {
			})
		})
	})
}

func constructorNominalMetadata() {
	dsl.Attribute("value:x-value")
	dsl.Attribute("items:x-items")
	dsl.Attribute("located:x-located")
}
