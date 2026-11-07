package expr_test

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestNullableDefaultErrorMedia(t *testing.T) {
	for _, test := range []struct {
		name     string
		nullable bool
		media    string
		want     string
	}{
		{"nullable default", true, "", "application/json"},
		{"nullable JSON", true, "application/json", "application/json"},
		{"ordinary problem", false, "", "application/problem+json"},
		{"reject nullable problem", true, "application/problem+json", ""},
		{"reject parameterized problem", true, "Application/Problem+JSON; charset=utf-8", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			design := func() {
				Service("problems", func() {
					Method("read", func() {
						Error("missing", func() {
							if test.nullable {
								Nullable()
							}
						})
						HTTP(func() {
							GET("/read")
							Response("missing", StatusNotFound, func() {
								if test.media != "" {
									ContentType(test.media)
								}
							})
						})
					})
				})
			}
			if test.want == "" {
				require.Contains(t, expr.RunInvalidDSL(t, design).Error(), "nullable whole problem body requires")
				return
			}
			root := expr.RunDSL(t, design)
			require.Equal(t, test.want, root.API.HTTP.Services[0].HTTPEndpoints[0].HTTPErrors[0].Response.ContentType)
		})
	}
}
