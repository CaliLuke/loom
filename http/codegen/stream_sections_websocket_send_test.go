package codegen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/service"
)

func TestWriteServerWebSocketResponseBodySwitchesViews(t *testing.T) {
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

	ws.Response.EncodePlan = newResponseEncodePlan(ws.Response)

	var b strings.Builder
	require.True(t, writeServerWebSocketResponseBody(&b, ws))

	require.Equal(t, "\tvar body any\n\tswitch res.View {\n\tcase \"default\", \"\":\n\t\tbody = newDefaultBody(res)\n\tcase \"tiny\":\n\t\tbody = newTinyBody(res)\n\t}\n\treturn s.conn.WriteJSON(ctx, body)\n", b.String())
}
