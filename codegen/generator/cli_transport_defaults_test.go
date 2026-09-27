package generator

import (
	"fmt"
	"testing"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

const transportDefaultHarness = `package defaults_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	loomhttp "github.com/CaliLuke/loom/http"
	cli "example.com/defaults/gen/http/cli/%s"
	server "example.com/defaults/gen/http/%s/server"
)

func TestOmittedAndExplicitDefaults(t *testing.T) {
	args := os.Args
	t.Cleanup(func() {
		os.Args = args
	})
	for _, tc := range []struct {
		name string
		provided bool
		want string
	}{
		{"omitted", false, %q},
		{"explicit", true, %q},
	} {
		t.Run(tc.name, func(t *testing.T) {
			os.Args = []string{"defaults-cli", %q, %q}
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.provided {
				os.Args = append(os.Args, %q, tc.want)
				if %t {
					r.AddCookie(&http.Cookie{Name: %q, Value: tc.want})
				} else {
					r.Header.Set(%q, tc.want)
				}
			}
			endpoint, payload, err := cli.ParseEndpoint("http", "localhost", nil, nil, nil, false)
			require.NoError(t, err)
			require.NotNil(t, endpoint)
			check := func(payload any) {
				field := reflect.ValueOf(payload).Elem().FieldByName(%q)
				require.False(t, field.IsNil())
				require.Equal(t, tc.want, fmt.Sprint(field.Elem().Interface()))
			}
			check(payload)
			decoded, err := server.Decode%sRequest(loomhttp.NewMuxer(), loomhttp.RequestDecoder)(r)
			require.NoError(t, err)
			check(decoded)
		})
	}
}
`

func TestCLITransportDefaultsCompile(t *testing.T) {
	for _, tc := range []struct {
		name          string
		design        func()
		cli           string
		service       string
		method        string
		methodName    string
		flag          string
		wireName      string
		field         string
		cookie        bool
		defaultValue  string
		explicitValue string
	}{
		{"header", testdata.OpenAPI32FeaturesDSL, "production", "catalog", "parameters", "Parameters", "--filter", "X-Filter", "Filter", false, "all", "override"},
		{"header with metadata", func() {
			mappingDefaultDSL(false, false)
		}, "test_api", "defaults", "read", "Read", "--value", "value", "CustomValue", false, "all", "override"},
		{"cookie with metadata", func() {
			mappingDefaultDSL(true, false)
		}, "test_api", "defaults", "read", "Read", "--value", "value", "CustomValue", true, "all", "override"},
		{"header custom type", func() {
			mappingDefaultDSL(false, true)
		}, "test_api", "defaults", "read", "Read", "--value", "value", "CustomValue", false, "550e8400-e29b-41d4-a716-446655440000", "6ba7b810-9dad-11d1-80b4-00c04fd430c8"},
		{"cookie custom type", func() {
			mappingDefaultDSL(true, true)
		}, "test_api", "defaults", "read", "Read", "--value", "value", "CustomValue", true, "550e8400-e29b-41d4-a716-446655440000", "6ba7b810-9dad-11d1-80b4-00c04fd430c8"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			harness := fmt.Sprintf(transportDefaultHarness, tc.cli, tc.service, tc.defaultValue, tc.explicitValue, tc.service, tc.method, tc.flag, tc.cookie, tc.wireName, tc.wireName, tc.field, tc.methodName)
			runDesignHarness(t, "example.com/defaults", tc.design, harness)
		})
	}
}

func mappingDefaultDSL(cookie, customType bool) {
	dsl.Service("defaults", func() {
		dsl.Method("read", func() {
			dsl.Payload(func() {
				dsl.Attribute("value", dsl.String, func() {
					dsl.Meta("struct:field:name", "CustomValue")
					if customType {
						dsl.Meta("struct:field:type", "uuid.UUID", "github.com/google/uuid")
						dsl.Format(dsl.FormatUUID)
					}
				})
			})
			dsl.HTTP(func() {
				dsl.GET("/")
				mapping := dsl.Header
				if cookie {
					mapping = dsl.Cookie
				}
				mapping("value", dsl.String, func() {
					if customType {
						dsl.Default("550e8400-e29b-41d4-a716-446655440000")
					} else {
						dsl.Default("all")
					}
					dsl.Meta("openapi:allowReserved", "true")
				})
			})
		})
	})
}
