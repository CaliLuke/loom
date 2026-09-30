package representation

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestResponseCodecAuthority(t *testing.T) {
	for _, test := range []struct {
		name, media string
		header      bool
		codec       expr.ValueCodec
	}{
		{"default", "", false, expr.ValueCodecJSON},
		{"unknown explicit", "application/unknown", false, expr.ValueCodecJSON},
		{"explicit text", "text/plain", false, expr.ValueCodecText},
		{"xml", "application/xml", false, expr.ValueCodecCustom},
		{"gob", "application/gob", false, expr.ValueCodecCustom},
		{"dynamic", "", true, expr.ValueCodecCustom},
		{"static JSON", "application/json", true, expr.ValueCodecJSON},
		{"static text", "text/plain", true, expr.ValueCodecText},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				dsl.Service("codec", func() {
					dsl.Method("show", func() {
						dsl.Payload(dsl.Bytes)
						dsl.Result(func() {
							dsl.Attribute("data", dsl.Bytes)
							dsl.Attribute("media", dsl.String, func() {
								dsl.Enum("application/json", "text/plain", "application/gob")
							})
							dsl.Required("data")
						})
						dsl.HTTP(func() {
							dsl.POST("/")
							dsl.Response(dsl.StatusOK, func() {
								dsl.Body("data")
								if test.header {
									dsl.Header("media:Content-Type")
								}
								if test.media != "" {
									dsl.ContentType(test.media)
								}
							})
						})
					})
				})
			})
			prepared, err := PrepareService(root.API.HTTP.Services[0], nil)
			require.NoError(t, err)
			endpoint := prepared.Endpoints[0]
			require.Equal(t, expr.ValueCodecJSON, endpoint.Request.BodyValue.Codec, "client request encoding remains JSON")
			response := endpoint.Response.Responses[0]
			require.Equal(t, test.codec, response.BodyValue.Codec)
			require.Len(t, response.BodyValues, len(response.ContentTypes))
			require.Len(t, response.DocumentValues, len(response.ContentTypes))
			for _, media := range response.ContentTypes {
				body, document := response.BodyValues[media], response.DocumentValues[media]
				require.Same(t, response.BodyValue.Source, body.Source)
				require.NoError(t, body.Error)
				require.NoError(t, document.Error)
				require.Equal(t, test.codec, body.Plan.Root().Codec())
				require.Equal(t, test.codec, document.Plan.Root().Codec())
			}
		})
	}
}
