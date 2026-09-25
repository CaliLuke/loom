package expr_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestHTTPStreamingBodyRemovesPkgPath checks that the HTTP WebSocket
// streaming body of an object, array or map streaming payload refers to no
// struct:pkg:path type, that the element types of a collection keep their
// names, and that the method streaming payload keeps its struct:pkg:path
// types.
func TestHTTPStreamingBodyRemovesPkgPath(t *testing.T) {
	cases := []struct {
		Name string
		// Payload returns the streaming payload type built from the
		// relocated type.
		Payload func(expr.DataType) expr.DataType
		// Names lists the user types of the streaming body.
		Names []string
	}{
		{
			Name:    "object",
			Payload: func(spa expr.DataType) expr.DataType { return spa },
			Names:   []string{"UploadStreamingBody", "SPAStreamingBody", "InnerStreamingBody"},
		},
		{
			Name:    "array",
			Payload: func(spa expr.DataType) expr.DataType { return ArrayOf(spa) },
			Names:   []string{"SPA", "Inner"},
		},
		{
			Name:    "map",
			Payload: func(spa expr.DataType) expr.DataType { return MapOf(String, spa) },
			Names:   []string{"SPA", "Inner"},
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				inner := Type("Inner", func() {
					Meta("struct:pkg:path", "pkgb")
					Attribute("n", Int)
				})
				spa := Type("SPA", func() {
					Meta("struct:pkg:path", "pkgb")
					Attribute("id", String)
					Attribute("inner", inner)
					Required("id")
				})
				Service("svc", func() {
					Method("upload", func() {
						StreamingPayload(c.Payload(spa))
						Result(String)
						HTTP(func() {
							GET("/upload")
						})
					})
				})
			})
			e := root.API.HTTP.Services[0].HTTPEndpoints[0]
			require.NotNil(t, e.StreamingBody)
			assert.Equal(t, c.Names, userTypes(e.StreamingBody.Type, func(ut expr.UserType) string {
				_, ok := ut.Attribute().Meta["struct:pkg:path"]
				assert.False(t, ok, "streaming body type %q has struct:pkg:path", ut.Name())
				return ut.Name()
			}))
			assert.Equal(t, []string{"pkgb", "pkgb"}, userTypes(e.MethodExpr.StreamingPayload.Type, func(ut expr.UserType) string {
				return ut.Attribute().Meta["struct:pkg:path"][0]
			}))
		})
	}
}

// userTypes returns the results of f for the user types that dt refers to,
// in depth-first order.
func userTypes(dt expr.DataType, f func(expr.UserType) string) []string {
	var res []string
	seen := make(map[string]struct{})
	var visit func(expr.DataType)
	visit = func(dt expr.DataType) {
		switch t := dt.(type) {
		case expr.UserType:
			if _, ok := seen[t.ID()]; ok {
				return
			}
			seen[t.ID()] = struct{}{}
			res = append(res, f(t))
			visit(t.Attribute().Type)
		case *expr.Object:
			for _, nat := range *t {
				visit(nat.Attribute.Type)
			}
		case *expr.Array:
			visit(t.ElemType.Type)
		case *expr.Map:
			visit(t.KeyType.Type)
			visit(t.ElemType.Type)
		}
	}
	visit(dt)
	return res
}
