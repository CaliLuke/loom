package transportir_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	testcodegen "github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

func TestResolvedParameterCodecs(t *testing.T) {
	root := testcodegen.RunDSL(t, testdata.ParameterStylesDSL)
	service := transportir.BuildService(root.API.HTTP.Services[0])
	endpoints := make(map[string]*transportir.Endpoint)
	for _, endpoint := range service.Endpoints {
		endpoints[endpoint.Name] = endpoint
	}
	for _, tc := range []struct {
		method  string
		style   httpencoding.Style
		shape   httpencoding.Shape
		explode bool
	}{
		{"PathMatrix", httpencoding.Matrix, httpencoding.Scalar, false},
		{"PathLabelExploded", httpencoding.Label, httpencoding.Sequence, true},
		{"QueryObject", httpencoding.DeepObject, httpencoding.Object, false},
		{"QueryResidual", httpencoding.Form, httpencoding.ScalarMap, true},
		{"QueryJSON", httpencoding.JSON, httpencoding.ScalarMap, false},
		{"HeaderMap", httpencoding.Simple, httpencoding.ScalarMap, false},
		{"CookieList", httpencoding.CookieStyle, httpencoding.Sequence, false},
	} {
		t.Run(tc.method, func(t *testing.T) {
			request := endpoints[tc.method].Request
			parameters := append(append(append(request.PathParams, request.QueryParams...), request.Headers...), request.Cookies...)
			require.NotEmpty(t, parameters)
			require.Equal(t, httpencoding.Codec{Style: tc.style, Shape: tc.shape, Explode: tc.explode}, parameters[0].Codec)
		})
	}
	form := endpoints["FormFields"].Request
	require.Equal(t, httpencoding.DeepObject, form.FormFieldCodecs["filter"].Style)
	require.Equal(t, httpencoding.Form, form.FormFieldCodecs["attributes"].Style)
	require.False(t, form.FormFieldCodecs["attributes"].Explode)
	require.Equal(t, httpencoding.Form, form.FormFieldCodecs["flags"].Style)
	require.True(t, form.FormFieldCodecs["flags"].Explode)
	require.NotEqual(t, httpencoding.JSON, form.FormFieldCodecs["attributes"].Style)
	require.True(t, endpoints["QueryResidual"].Request.QueryParams[0].Keys.Residual)
	require.Equal(t, []string{"value[R]", "value[G]"}, endpoints["QueryObject"].Request.QueryParams[0].Keys.Literals)
}
