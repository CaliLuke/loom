package codegen

import (
	"strings"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
	"github.com/stretchr/testify/require"
)

func TestResponseBodyInitConsumesSourcePreparation(t *testing.T) {
	// The common consumer accepts captured response plans. Use an actual HTTP
	// plan with selection to test preparation independently of protocol framing.
	for _, nullable := range []bool{false, true} {
		root := httpcodegen.RunHTTPDSL(t, func() {
			Service("events", func() {
				Method("read", func() {
					Result(func() {
						if nullable {
							Nullable()
						}
						Attribute("value", String)
						Required("value")
					})
					HTTP(func() {
						GET("/")
						Response(StatusOK, func() {
							Body("value")
						})
					})
				})
			})
		})
		response := httpcodegen.CreateHTTPServices(root).Get("events").Endpoints[0].Result.Responses[0]
		code := responseBodyInit("native", response, response.ServerBody, nil)
		failure := func(value string) string {
			return "handlePreparationFailure(" + value + ")\nreturn nil"
		}
		routed := responseBodyInit("native", response, response.ServerBody, failure)
		require.Contains(t, routed, response.EncodePlan.SourcePreparation(failure, ""))
		body := response.ServerBody[0]
		originalInit := body.Init
		body.Init = &httpcodegen.InitData{ServerErrorAware: true}
		conversion := responseBodyInit("native", response, response.ServerBody, failure)
		require.Contains(t, conversion, failure("conversionError"))
		body.Init = originalInit
		prepare := response.EncodePlan.SourcePreparation(nil, "")
		if !nullable {
			require.Contains(t, prepare, "if res == nil")
			require.Contains(t, code, prepare)
			require.Less(t, strings.Index(code, prepare), strings.Index(code, "body = res.Value"))
			continue
		}

		require.NotEmpty(t, prepare)
		require.Contains(t, code, prepare)
		require.Less(t, strings.Index(code, prepare), strings.Index(code, "body = responseValue.Value"))
	}
}
