package expr_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestHTTPSecuritySchemeLocation checks the transport element name and the
// location of the credential of the security scheme of an HTTP endpoint: a
// Param that a route wildcard names is a path parameter, and a Param that no
// route wildcard names is a query string parameter.
func TestHTTPSecuritySchemeLocation(t *testing.T) {
	jwt := func(route string, mapping func()) func() {
		return func() {
			scheme := JWTSecurity("jwt")
			Service("svc", func() {
				Method("m", func() {
					Security(scheme)
					Payload(func() {
						Token("t", String)
						Required("t")
					})
					HTTP(func() {
						GET(route)
						mapping()
					})
				})
			})
		}
	}
	apiKey := func(route string, mapping func()) func() {
		return func() {
			scheme := APIKeySecurity("key")
			Service("svc", func() {
				Method("m", func() {
					Security(scheme)
					Payload(func() {
						APIKey("key", "k", String)
						Required("k")
					})
					HTTP(func() {
						GET(route)
						mapping()
					})
				})
			})
		}
	}
	cases := []struct {
		name     string
		dsl      func()
		wantName string
		wantIn   string
	}{
		{name: "token path param", dsl: jwt("/items/{t}", func() {}), wantName: "t", wantIn: "path"},
		{name: "mapped token path param", dsl: jwt("/items/{tok}", func() { Param("t:tok") }), wantName: "tok", wantIn: "path"},
		{name: "token query param", dsl: jwt("/items", func() { Param("t") }), wantName: "t", wantIn: "query"},
		{name: "mapped token query param", dsl: jwt("/items", func() { Param("t:tok") }), wantName: "tok", wantIn: "query"},
		{name: "token header", dsl: jwt("/items", func() { Header("t:Authorization") }), wantName: "Authorization", wantIn: "header"},
		{name: "api key query param", dsl: apiKey("/items", func() { Param("k") }), wantName: "k", wantIn: "query"},
		{name: "mapped api key query param", dsl: apiKey("/items", func() { Param("k:api_key") }), wantName: "api_key", wantIn: "query"},
		{name: "api key cookie", dsl: apiKey("/items", func() { Cookie("k:session") }), wantName: "session", wantIn: "cookie"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := expr.RunDSL(t, c.dsl)
			endpoint := root.API.HTTP.Services[0].HTTPEndpoints[0]
			require.Len(t, endpoint.Requirements, 1)
			require.Len(t, endpoint.Requirements[0].Schemes, 1)
			scheme := endpoint.Requirements[0].Schemes[0]
			assert.Equal(t, c.wantName, scheme.Name)
			assert.Equal(t, c.wantIn, scheme.In)
		})
	}
}

// TestHTTPAPIKeyPathParamRejected checks that an API key read from a path
// parameter is rejected: an OpenAPI API key security scheme can only name a
// header, query string parameter or cookie.
func TestHTTPAPIKeyPathParamRejected(t *testing.T) {
	cases := []struct {
		name  string
		route string
		param func()
		want  string
	}{
		{
			name:  "path param",
			route: "/items/{k}",
			param: func() {},
			want:  `API key of security scheme "key" is read from the path parameter "k", but an OpenAPI API key can only be sent in a header, a query string parameter or a cookie; map the "k" attribute with Header, Cookie, or a Param that no route wildcard names`,
		},
		{
			name:  "mapped path param",
			route: "/items/{kk}",
			param: func() { Param("k:kk") },
			want:  `API key of security scheme "key" is read from the path parameter "kk"`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				scheme := APIKeySecurity("key")
				Service("svc", func() {
					Method("m", func() {
						Security(scheme)
						Payload(func() {
							APIKey("key", "k", String)
							Required("k")
						})
						HTTP(func() {
							GET(c.route)
							c.param()
						})
					})
				})
			})
			got := stripValidationLocations(err.Error())
			if !strings.Contains(got, c.want) {
				t.Errorf("got error %q\nwant it to contain %q", got, c.want)
			}
		})
	}
}
