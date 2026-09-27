package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/stretchr/testify/require"
)

const inheritedAPIErrorHarness = `package apierror_test

import (
 "context"
 "errors"
 "io"
 "net/http"
 "net/http/httptest"
 "net/url"
 "testing"

 svc "example.com/apierror/gen/service_no_body_error_response"
 client "example.com/apierror/gen/http/service_no_body_error_response/client"
 server "example.com/apierror/gen/http/service_no_body_error_response/server"
 loomhttp "github.com/CaliLuke/loom/http"
 "github.com/stretchr/testify/require"
)

type implementation struct{}

func (*implementation) MethodServiceErrorResponse(context.Context) error {
 message := "specific failure"
 return &svc.StringError{Header: &message}
}

func TestInheritedErrorRoundTrip(t *testing.T) {
 mux := loomhttp.NewMuxer()
 server.Mount(mux, server.New(svc.NewEndpoints(&implementation{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
 host := httptest.NewServer(mux)
 defer host.Close()
 response, err := http.Get(host.URL+"/one/two")
 require.NoError(t, err)
 body, err := io.ReadAll(response.Body)
 require.NoError(t, err)
 require.NoError(t, response.Body.Close())
 require.Equal(t, 400, response.StatusCode)
 require.Empty(t, body)
 require.Equal(t, "specific failure", response.Header.Get("header"))
 address, err := url.Parse(host.URL)
 require.NoError(t, err)
 c := client.NewClient(address.Scheme, address.Host, http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
 _, err = c.MethodServiceErrorResponse()(context.Background(), nil)
 var typed *svc.StringError
 require.True(t, errors.As(err, &typed), "%v", err)
 require.NotNil(t, typed.Header)
 require.Equal(t, "specific failure", *typed.Header)
}
`

// TestInheritedAPIErrorGeneratedModule round-trips custom inherited error headers.
func TestInheritedAPIErrorGeneratedModule(t *testing.T) {
	for _, tc := range []struct {
		name   string
		design func()
	}{
		{"header", testdata.APINoBodyErrorResponseDSL},
		{"content-type", testdata.APINoBodyErrorResponseWithContentTypeDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, tc.design)
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/apierror", root)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "error_test.go"), []byte(inheritedAPIErrorHarness), 0600))
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "vet", "./...")
			runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
		})
	}
}

// TestInheritedAPIErrorSharedPackage compiles inherited errors in shared type packages.
func TestInheritedAPIErrorSharedPackage(t *testing.T) {
	for _, tc := range []struct {
		path    string
		payload bool
	}{{"", false}, {"", true}, {"types", false}, {"types/shared", false}, {"types/shared", true}} {
		t.Run(fmt.Sprintf("%s/payload=%t", tc.path, tc.payload), func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				failure := Type("Failure", func() {
					Attribute("message", String)
					if tc.path != "" {
						Meta("struct:pkg:path", tc.path)
					}
				})
				API("test", func() {
					Error("bad_request", failure)
					HTTP(func() {
						Response("bad_request", StatusBadRequest)
					})
				})
				Service("shared", func() {
					Error("bad_request")
					Method("show", func() {
						if tc.payload {
							Payload(failure)
						}
						HTTP(func() {
							POST("/")
						})
					})
				})
			})
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/sharedapierror", root)
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "vet", "./...")
			runGoCommand(t, dir, "test", "./...")
		})
	}
}
