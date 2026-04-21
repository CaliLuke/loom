package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/service"
)

func TestWriteServerWebSocketBodyInitSwitchesViews(t *testing.T) {
	ws := &WebSocketData{
		Response: &ResponseData{
			ServerBody: []*TypeData{
				{
					View: "default",
					Init: &InitData{
						Name: "newDefaultBody",
						ServerArgs: []*InitArgData{
							{Ref: "res"},
						},
					},
				},
				{
					View: "tiny",
					Init: &InitData{
						Name: "newTinyBody",
						ServerArgs: []*InitArgData{
							{Ref: "res"},
						},
					},
				},
			},
		},
		Endpoint: &EndpointData{
			Method: &service.MethodData{
				MethodResultData: service.MethodResultData{
					ViewedResult: &service.ViewedResultTypeData{
						Views: []*service.ViewData{
							{Name: "default"},
							{Name: "tiny"},
						},
					},
				},
			},
		},
	}

	var b sourceBuilder
	writeServerWebSocketBodyInit(&b, ws, ws.Response.ServerBody[0])

	require.Equal(t, "\tvar body any\n\tswitch s.view {\n\tcase \"default\", \"\":\n\t\tbody = newDefaultBody(res, )\n\tcase \"tiny\":\n\t\tbody = newTinyBody(res, )\n\t}\n", b.String())
}
