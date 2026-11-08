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

func TestWebSocketUpgradeClosesInformationalAdmission(t *testing.T) {
	endpoint := &EndpointData{Method: &service.MethodData{MethodResultData: service.MethodResultData{
		ViewedResult: &service.ViewedResultTypeData{},
	}}}
	for _, operation := range []string{"Send", "Recv"} {
		t.Run(operation, func(t *testing.T) {
			code := renderWebsocketUpgrade(endpoint, operation, operation == "Recv", true)
			admission := strings.Index(code, "exchange.BeginFinal(loomhttp.FinalUpgrade)")
			require.GreaterOrEqual(t, admission, 0)
			require.Less(t, strings.Index(code, "s.once.Do(func() {"), admission)
			require.Less(t, admission, strings.Index(code, "s.upgrader.Upgrade("))
			if operation == "Send" {
				require.Less(t, admission, strings.Index(code, "respHdr := make(http.Header)"))
			}
		})
	}
}
