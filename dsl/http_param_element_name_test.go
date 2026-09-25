package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestParamElementNameMatchesRouteParam checks that a route wildcard names
// the transport element of a Param declared with an element name suffix, so
// that Param("key:k") with the route "/{k}" is the path parameter of the key
// attribute, and that a suffixed Param absent from the route stays a query
// parameter.
func TestParamElementNameMatchesRouteParam(t *testing.T) {
	root := expr.RunDSL(t, func() {
		Service("svc", func() {
			Method("show", func() {
				Payload(func() {
					Attribute("key", String)
					Attribute("id", Int)
					Attribute("q", String)
					Required("key", "id")
				})
				HTTP(func() {
					GET("/items/{k}/{id}")
					Param("key:k")
					Param("q:query")
				})
			})
		})
	})
	endpoint := root.API.HTTP.Service("svc").Endpoint("show")

	path := endpoint.PathParams()
	assert.Equal(t, []string{"key", "id"}, mappedNames(path))
	assert.Equal(t, "k", path.ElemName("key"))
	assert.Equal(t, "id", path.ElemName("id"))
	assert.True(t, path.IsRequired("key"))
	assert.True(t, path.IsRequired("id"))
	query := endpoint.QueryParams()
	assert.Equal(t, []string{"q"}, mappedNames(query))
	assert.Equal(t, "query", query.ElemName("q"))
	assert.ElementsMatch(t, []string{"key", "id", "q"}, mappedNames(endpoint.Params), "no implicit route param k")
}

// TestParamElementNameRouteRejections checks that a route wildcard naming
// the attribute of a Param whose element name differs is rejected with a
// message naming the element, and that a wildcard matching neither is still
// reported as missing.
func TestParamElementNameRouteRejections(t *testing.T) {
	cases := map[string]struct {
		route string
		param string
		want  string
	}{
		"attribute name in route": {
			route: "/items/{key}",
			param: "key:k",
			want:  `route param "key" names the attribute of Param "key:k"; use "{k}" in the route`,
		},
		"unknown element": {
			route: "/items/{other}",
			param: "key:k",
			want:  `Route param "other" not found in method payload`,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				Service("svc", func() {
					Method("show", func() {
						Payload(func() {
							Attribute("key", String)
							Required("key")
						})
						HTTP(func() {
							GET(tc.route)
							Param(tc.param)
						})
					})
				})
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func mappedNames(ma *expr.MappedAttributeExpr) []string {
	object := *expr.AsObject(ma.Type)
	names := make([]string, 0, len(object))
	for _, nat := range object {
		names = append(names, nat.Name)
	}
	return names
}
