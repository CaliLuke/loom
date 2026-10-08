package transportir_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	testcodegen "github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestResponseSelectionOrder(t *testing.T) {
	root := testcodegen.RunDSL(t, testdata.ResponseSelectionOrderDSL)
	for _, endpoint := range root.API.HTTP.Services[0].HTTPEndpoints {
		t.Run(endpoint.Name(), func(t *testing.T) {
			authored := make([]int, len(endpoint.Responses))
			for i, response := range endpoint.Responses {
				authored[i] = response.StatusCode
			}
			plan := transportir.BuildEndpoint(endpoint).Response
			codes := make([]int, 0, len(plan.Responses))
			for _, response := range plan.Responses {
				codes = append(codes, response.StatusCode)
				require.Equal(t, "message", response.BodyOrigin)
				require.Len(t, response.Headers, 1)
			}
			require.Equal(t, []int{201, 202, 200}, codes)
			require.Equal(t, "first", plan.Responses[0].TagName)
			require.Equal(t, "second", plan.Responses[1].TagName)
			require.Empty(t, plan.Responses[2].TagName)
			for i, response := range endpoint.Responses {
				require.Equal(t, authored[i], response.StatusCode, "analysis must not mutate the authored expression")
			}
		})
	}
}
