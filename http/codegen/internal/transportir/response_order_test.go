package transportir_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	testcodegen "github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestResponseSelectionOrder(t *testing.T) {
	root := testcodegen.RunDSL(t, testdata.ResponseSelectionOrderDSL)
	for _, endpoint := range root.API.HTTP.Services[0].HTTPEndpoints {
		t.Run(endpoint.Name(), func(t *testing.T) {
			authored := make([]int, len(endpoint.Responses))
			for i, response := range endpoint.Responses {
				authored[i] = response.Status.Code
			}
			plan := transportir.BuildEndpoint(endpoint).Response
			codes := make([]int, 0, len(plan.Responses))
			for _, response := range plan.Responses {
				require.True(t, response.Selector.IsExact())
				codes = append(codes, response.Selector.Code)
				if endpoint.Name() == "bodyless" {
					require.Empty(t, response.BodyOrigin)
					require.Empty(t, response.Headers)
					require.NotNil(t, response.Body)
					require.Equal(t, expr.Empty, response.Body.Type)
				} else {
					require.Equal(t, "message", response.BodyOrigin)
					require.Len(t, response.Headers, 3)
				}
			}
			if endpoint.Name() == "bodyless" {
				require.Equal(t, []int{202, 200}, codes)
				require.Equal(t, "kind", plan.Responses[0].Tag.Key)
				require.Equal(t, "accepted", plan.Responses[0].Tag.Value)
				require.Nil(t, plan.Responses[1].Tag)
			} else {
				require.Equal(t, []int{201, 202, 200}, codes)
				require.Equal(t, "first", plan.Responses[0].Tag.Key)
				require.Equal(t, "second", plan.Responses[1].Tag.Key)
				require.Nil(t, plan.Responses[2].Tag)
			}
			for i, response := range endpoint.Responses {
				require.Equal(t, authored[i], response.Status.Code, "analysis must not mutate the authored expression")
			}
		})
	}
}
