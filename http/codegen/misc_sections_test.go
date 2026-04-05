package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
)

func TestRequestBuilderSectionStructuredDeclaration(t *testing.T) {
	section := requestBuilderSection(&EndpointData{
		ClientStruct: "WidgetClient",
		RequestInit: &InitData{
			Name:        "BuildWidgetRequest",
			Description: "BuildWidgetRequest builds the widget request.",
			ClientArgs: []*InitArgData{
				{
					AttributeData: &AttributeData{VarName: "id", TypeRef: "string"},
				},
			},
			ClientCode: `return http.NewRequest("GET", "/widgets/"+id, nil)`,
		},
	})

	code := codegen.SectionCode(t, section)
	require.Contains(t, code, "func (c *WidgetClient) BuildWidgetRequest(ctx context.Context, id string) (*http.Request, error) {")
	require.Contains(t, code, `return http.NewRequest("GET", "/widgets/"+id, nil)`)
}

func TestTransformHelperSectionStructuredDeclaration(t *testing.T) {
	section := transformHelperSection("client-transform-helper", &codegen.TransformFunctionData{
		Name:          "buildWidgetResponse",
		ParamTypeRef:  "*source.Widget",
		ResultTypeRef: "*target.Widget",
		Code: `res := &target.Widget{
	Name: v.Name,
}`,
	})

	code := codegen.SectionCode(t, section)
	require.Contains(t, code, "func buildWidgetResponse(v *source.Widget) *target.Widget {")
	require.Contains(t, code, "return res")
}
