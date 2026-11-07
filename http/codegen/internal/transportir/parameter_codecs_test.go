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
	require.Equal(t, httpencoding.DeepObject, form.Form.Fields["filter"].Style)
	require.Equal(t, httpencoding.Form, form.Form.Fields["attributes"].Style)
	require.False(t, form.Form.Fields["attributes"].Explode)
	require.Equal(t, httpencoding.Form, form.Form.Fields["flags"].Style)
	require.True(t, form.Form.Fields["flags"].Explode)
	require.NotEqual(t, httpencoding.JSON, form.Form.Fields["attributes"].Style)
	require.True(t, endpoints["QueryResidual"].Request.QueryParams[0].Keys.Residual)
	require.Equal(t, []string{"value[R]", "value[G]"}, endpoints["QueryObject"].Request.QueryParams[0].Keys.Literals)
}

func TestSingleUnionFormFieldKeepsNamedCodec(t *testing.T) {
	root := testcodegen.RunDSL(t, testdata.UntaggedFormDSL)
	service := transportir.BuildService(root.API.HTTP.Services[0])
	checked := 0
	for _, endpoint := range service.Endpoints {
		switch endpoint.Name {
		case "fields", "exploded", "Tags", "JSONField", "NullableJSONField", "Map", "MapJSON", "Object", "ObjectJSON":
		default:
			continue
		}
		checked++
		t.Run(endpoint.Name, func(t *testing.T) {
			codec, exists := endpoint.Request.Form.Fields["value"]
			require.True(t, exists, "authored object property must retain its field codec")
			if endpoint.Name == "Tags" || endpoint.Name == "JSONField" || endpoint.Name == "NullableJSONField" || endpoint.Name == "MapJSON" || endpoint.Name == "ObjectJSON" {
				require.Equal(t, httpencoding.JSON, codec.Style)
			} else {
				require.Equal(t, httpencoding.Form, codec.Style)
			}
		})
	}
	require.Equal(t, 9, checked)
}
