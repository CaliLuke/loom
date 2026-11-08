package codegen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestInformationalHeaderDestination(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		choice := Type("Choice", OneOf(String, Int), Untagged)
		carrier := Type("Hints", func() {
			Attribute("labels", MapOf(String, String))
			Attribute("choice", choice)
			Required("labels", "choice")
		})
		Service("headers", func() {
			Method("read", func() {
				HTTP(func() {
					GET("/headers")
					Informational("hints", StatusEarlyHints, carrier, func() {
						Header("labels:X-Labels")
						Header("choice:X-Choice")
					})
					Response(StatusNoContent)
				})
			})
		})
	})
	response := CreateHTTPServices(root).Get("headers").Endpoint("read").Informational[0].Response
	for _, header := range response.Headers {
		t.Run(header.Name, func(t *testing.T) {
			require.True(t, header.Styled)
			ordinary := renderStyledEncode(header.Element, "res", "response-header", "w.Header()")
			interim := renderStyledEncode(header.Element, "res", "response-header", "informationalHeaders")
			require.Contains(t, interim, "informationalHeaders.Set(")
			require.NotContains(t, interim, "w.Header()")
			require.Equal(t, ordinary, strings.ReplaceAll(interim, "informationalHeaders", "w.Header()"))
		})
	}
}

func TestInformationalHeaderFormatting(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		testdata.InformationalServer(false)
	})
	response := CreateHTTPServices(root).Get("selection").Endpoint("read").Informational[0].Response
	ordinary, err := renderResponseHeaders(response, "res", "w.Header()")
	require.NoError(t, err)
	interim, err := renderResponseHeaders(response, "res", "informationalHeaders")
	require.NoError(t, err)
	require.Contains(t, interim, `informationalHeaders.Set("X-Trace", res.Trace)`)
	require.Contains(t, interim, `loomhttp.SetHeaderValues(informationalHeaders, "X-Labels",`)
	require.Contains(t, interim, `informationalHeaders.Set("X-Metadata", parameterEncoded)`)
	require.NotContains(t, interim, "w.Header()")
	require.Equal(t, ordinary, strings.ReplaceAll(interim, "informationalHeaders", "w.Header()"))
}

func TestInformationalHeaderMappingValidation(t *testing.T) {
	for _, tc := range []struct {
		name           string
		native, mapped int
	}{
		{"stronger mapping", 2, 4}, {"inherited stronger native", 4, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, informationalHeaderMappingDSL(tc.native, tc.mapped))
			response := CreateHTTPServices(root).Get("mapping").Endpoint("read").Informational[0].Response
			facts, err := expr.EffectiveConstraintFactsFor(response.Headers[0].attribute)
			require.NoError(t, err)
			require.NotNil(t, facts.Validation().Lowered().MinLength)
			require.Equal(t, tc.mapped, *facts.Validation().Lowered().MinLength)
			_, native := response.Carrier.FindAttribute("trace")
			require.Equal(t, tc.native, *native.Validation.MinLength)
			require.Contains(t, response.Headers[0].Validate, "< 4")
		})
	}
}

func informationalHeaderMappingDSL(native, mapped int) func() {
	return func() {
		carrier := Type("Hints", func() {
			Attribute("trace", String, func() { MinLength(native) })
			Required("trace")
		})
		Service("mapping", func() {
			Method("read", func() {
				HTTP(func() {
					GET("/mapping")
					Informational("hints", StatusEarlyHints, carrier, func() {
						Header("trace:X-Trace", func() { MinLength(mapped) })
					})
					Response(StatusNoContent)
				})
			})
		})
	}
}
