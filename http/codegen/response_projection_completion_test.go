package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestResponseProjectionCompletionPlan(t *testing.T) {
	root := RunHTTPDSL(t, testdata.NullableProjectionCompletionDSL)
	data := CreateHTTPServices(root).Get("completion")
	for _, endpoint := range data.Endpoints {
		t.Run(endpoint.Method.Name, func(t *testing.T) {
			response := endpoint.Result.Responses[0]
			switch endpoint.Method.Name {
			case "metadata", "required_metadata":
				require.False(t, response.EncodePlan.HasBody)
				require.Contains(t, response.EncodePlan.SourcePreparation(nil, ""), "if !responseHasValue")
				require.True(t, response.ResultInit.ReturnIsStruct)
				require.NotContains(t, response.ResultInit.ReturnTypeName, "Nullable[")
				for _, h := range response.Headers {
					require.NotEqual(t, "payloadError", h.VarName)
					require.NotEqual(t, "errorOwner", h.VarName)
				}
			case "optional", "nullable_optional":
				require.Contains(t, response.EncodePlan.SourcePreparation(nil, ""), ".Value == nil")
			case "whole", "selected", "dynamic_selected":
				require.Contains(t, response.EncodePlan.SourcePreparation(nil, ""), "res.Projected.Value()")
			case "tagged":
				for _, r := range endpoint.Result.Responses {
					if r.Tag != nil {
						require.NotContains(t, r.Tag.Match, "res.Projected.Kind")
					}
				}
			}
		})
	}
}

func TestViewedProjectionUsesNativeSource(t *testing.T) {
	root := RunHTTPDSL(t, testdata.NullableProjectionCompletionDSL)
	data := CreateHTTPServices(root).Get("completion")
	endpoint := data.Endpoint("whole")
	require.Empty(t, endpoint.Method.ViewedResult.ViewName)
	require.Contains(t, endpoint.Result.Responses[0].EncodePlan.SourcePreparation(nil, ""), "res.Projected.Value()")
}

func TestResponseFormatterPreparationPlan(t *testing.T) {
	root := RunHTTPDSL(t, testdata.NullableResponseProjectionDSL)
	data := CreateHTTPServices(root).Get("projection")
	for _, name := range []string{"formatted_child", "selected"} {
		t.Run(name, func(t *testing.T) {
			response := data.Endpoint(name).Errors[0].Errors[0].Response
			prepare := response.EncodePlan.SourcePreparation(nil, "formatter == nil")
			require.Contains(t, prepare, "if !res.Present()")
			if name == "formatted_child" {
				require.Contains(t, prepare, "if (formatter == nil) && (!responseHasValue)")
				require.Contains(t, prepare, "if (formatter == nil) && (!responseValue.Detail.Present())")
			} else {
				require.Contains(t, prepare, "if !responseHasValue")
				require.NotContains(t, prepare, "(formatter == nil) && (!responseHasValue)")
			}
		})
	}
}
