package codegen

import (
	"strings"
	"testing"

	"github.com/CaliLuke/loom/dsl"
	"github.com/stretchr/testify/require"
)

func TestResponseCodecSelectionPrecedesMappedHeader(t *testing.T) {
	for _, static := range []string{"", "application/json", "text/plain"} {
		t.Run(static, func(t *testing.T) {
			generated := serverEncodeSectionCode(t, func() {
				dsl.Service("headers", func() {
					dsl.Method("show", func() {
						dsl.Result(func() {
							dsl.Attribute("data", dsl.Bytes)
							dsl.Attribute("media", dsl.String, func() {
								dsl.Enum("application/json", "text/plain")
							})
							dsl.Required("data", "media")
						})
						dsl.HTTP(func() {
							dsl.GET("/")
							dsl.Response(dsl.StatusOK, func() {
								dsl.Body("data")
								dsl.Header("media:Content-Type")
								if static != "" {
									dsl.ContentType(static)
								}
							})
						})
					})
				})
			})
			selected := strings.Index(generated, "enc := encoder(ctx, w)")
			mapped := strings.Index(generated, `w.Header().Set("Content-Type",`)
			require.NotEqual(t, -1, selected)
			require.NotEqual(t, -1, mapped)
			require.Less(t, selected, mapped, "mapped Content-Type cannot select the already-created encoder")
			if static == "" {
				require.NotContains(t, generated, "loomhttp.ContentTypeKey")
			} else {
				override := strings.Index(generated, `ctx = context.WithValue(ctx, loomhttp.ContentTypeKey, "`+static+`")`)
				require.NotEqual(t, -1, override)
				require.Less(t, override, selected)
			}
		})
	}
}
