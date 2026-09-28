package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	httpcodegen "github.com/CaliLuke/loom/http/codegen"
	"github.com/CaliLuke/loom/jsonrpc/codegen/testdata"
)

func TestViewedStreamConstantBodySelection(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.ViewedStreamResultDSL)
	services := CreateJSONRPCServices(root)
	for _, tc := range []struct {
		name, service, method, view, selected string
		dynamic                               bool
	}{
		{"websocket default", "files", "talk", defaultViewExpr, "default", false},
		{"websocket collection default", "files", "talkList", defaultViewExpr, "default", false},
		{"websocket streaming payload result", "files", "upload", defaultViewExpr, "default", false},
		{"sse connection default", "feed", "follow", defaultViewExpr, "default", false},
		{"dynamic websocket", "files", "talk", "w.view", "", true},
		{"dynamic sse", "feed", "follow", "s.currentView()", "", true},
		{"fixed websocket", "files", "talkTiny", defaultViewExpr, "tiny", false},
		{"fixed sse", "feed", "followTiny", defaultViewExpr, "tiny", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var endpoint *httpcodegen.EndpointData
			for _, ed := range services.Get(tc.service).Endpoints {
				if ed.Method.Name == tc.method {
					endpoint = ed
					break
				}
			}
			require.NotNil(t, endpoint)
			bodies := endpoint.Result.Responses[0].ServerBody
			require.NotEmpty(t, bodies)
			code, ok := viewedStreamResultBodyInit("result", tc.view, bodies[0], endpoint)
			require.True(t, ok)
			if tc.dynamic {
				require.Contains(t, code, "switch vres.View")
				for _, body := range bodies {
					require.Contains(t, code, body.Init.Name+"(vres.Projected)")
				}
				return
			}
			require.NotContains(t, code, "switch vres.View")
			found := false
			for _, body := range bodies {
				call := body.Init.Name + "(vres.Projected)"
				if body.View == tc.selected {
					found = true
					require.Contains(t, code, "body := "+call)
				} else {
					require.NotContains(t, code, call)
				}
			}
			require.True(t, found, "selected view has a body constructor")
		})
	}
}
