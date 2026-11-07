package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestHTTPEmptyMediaBody(t *testing.T) {
	for _, tc := range []struct {
		name      string
		form      bool
		multipart bool
		optional  bool
	}{
		{name: "ordinary bodyless control"},
		{name: "required form", form: true},
		{name: "optional form", form: true, optional: true},
		{name: "required multipart", multipart: true},
		{name: "optional multipart", multipart: true, optional: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				empty := Type("EmptyRecord", func() {})
				Service("Empty", func() {
					Method("Submit", func() {
						Payload(empty)
						HTTP(func() {
							POST("/")
							if tc.form {
								FormRequest()
							}
							if tc.multipart {
								MultipartRequest()
							}
							if tc.optional {
								OptionalRequestBody()
							}
						})
					})
				})
			})
			body := root.API.HTTP.Services[0].HTTPEndpoints[0].Body
			if !tc.form && !tc.multipart {
				require.Equal(t, expr.Empty, body.Type)
				return
			}
			require.NotEqual(t, expr.Empty, body.Type)
			require.NotNil(t, expr.AsObject(body.Type))
			require.Empty(t, *expr.AsObject(body.Type))
		})
	}
}
