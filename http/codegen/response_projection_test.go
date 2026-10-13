package codegen

import (
	"testing"

	"github.com/CaliLuke/loom/codegen/service"
	"github.com/stretchr/testify/require"
)

func TestResponseProjectionValues(t *testing.T) {
	for _, tc := range []struct {
		name     string
		response ResponseData
		want     []string
	}{
		{
			name:     "whole",
			response: ResponseData{ServerBody: []*TypeData{{}}},
			want:     []string{"res"},
		},
		{
			name:     "selected native",
			response: ResponseData{ResultAttr: "Value", ServerBody: []*TypeData{{}}},
			want:     []string{"res.Value"},
		},
		{
			name: "selected viewed",
			response: ResponseData{
				ResultAttr:   "Value",
				ViewedResult: &service.ViewedResultTypeData{},
				ServerBody:   []*TypeData{{}},
			},
			want: []string{"res.Projected.Value"},
		},
		{
			name: "problem",
			response: ResponseData{
				SourceVar: "problem", ResultAttr: "Detail", ServerBody: []*TypeData{{}},
			},
			want: []string{"problem.Detail"},
		},
		{
			name: "constructor arguments",
			response: ResponseData{ServerBody: []*TypeData{{Init: &InitData{
				Name:       "NewBody",
				ServerArgs: []*InitArgData{{Ref: "res.Projected"}, {Ref: "res.View"}},
			}}}},
			want: []string{"NewBody(res.Projected, res.View)"},
		},
		{
			name: "view variants",
			response: ResponseData{
				ViewedResult: &service.ViewedResultTypeData{},
				ServerBody: []*TypeData{
					{View: "default", Init: &InitData{
						Name: "NewDefault", ServerArgs: []*InitArgData{{Ref: "res.Projected"}},
					}},
					{View: "tiny", Init: &InitData{
						Name: "NewTiny", ServerArgs: []*InitArgData{{Ref: "res.Projected"}},
					}},
				},
			},
			want: []string{"NewDefault(res.Projected)", "NewTiny(res.Projected)"},
		},
		{name: "bodyless"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan := newResponseEncodePlan(&tc.response)
			require.Len(t, plan.BodyValues, len(tc.want))
			for i, body := range tc.response.ServerBody {
				require.Equal(t, tc.want[i], plan.BodyValues[body])
			}
		})
	}
}
