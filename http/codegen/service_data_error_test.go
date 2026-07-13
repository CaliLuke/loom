package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

// The tests in this file drive the analysis panic paths that fire when the
// transport expression tree is inconsistent with the service expression tree.
// Valid DSL evaluation always derives compatible body and payload/result
// types, so each test first runs a valid design and then corrupts the derived
// transport attribute the way a buggy plugin or finalizer would.

// recoverAnalysisError runs fn, requires that it panics and returns the
// recovered panic value as an error.
func recoverAnalysisError(t *testing.T, fn func()) (err error) {
	t.Helper()
	defer func() {
		r := recover()
		require.NotNil(t, r, "expected analysis to panic")
		var ok bool
		err, ok = r.(error)
		require.True(t, ok, "panic value is not an error: %v", r)
	}()
	fn()
	return nil
}

func TestAnalyzePanicsOnIncompatibleRequestBody(t *testing.T) {
	cases := []struct {
		Name     string
		BodyType expr.DataType
	}{
		{"primitive-body-object-payload", expr.Int},
		{"array-body-object-payload", &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, testdata.PayloadBodyUserInnerDSL)
			// Corrupt the derived request body so it no longer matches the
			// method payload.
			root.API.HTTP.Services[0].HTTPEndpoints[0].Body.Type = c.BodyType
			services := CreateHTTPServices(root)

			err := recoverAnalysisError(t, func() {
				services.Get("ServiceBodyUserInner")
			})

			require.ErrorContains(t, err, "build HTTP server payload transform for MethodBodyUserInner")
		})
	}
}

func TestAnalyzePanicsOnIncompatibleResponseBody(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ResultBodyObjectHeaderDSL)
	// Corrupt the derived response body so it no longer matches the method
	// result.
	root.API.HTTP.Services[0].HTTPEndpoints[0].Responses[0].Body.Type = expr.Int
	services := CreateHTTPServices(root)

	err := recoverAnalysisError(t, func() {
		services.Get("ServiceBodyObjectHeader")
	})

	require.ErrorContains(t, err, "build HTTP response result transform")
}

func TestAnalyzePanicsOnUnknownResponseBodyView(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ResultBodyMultipleViewsDSL)
	// Fix the response to a view that does not exist on the result type. The
	// DSL validates view names so this is only reachable from a mutated tree.
	// The first projection to fail is the response body type build
	// (projectResponseBodyView).
	endpoint := root.API.HTTP.Services[0].HTTPEndpoints[0]
	endpoint.MethodExpr.Result.Meta = expr.MetaExpr{expr.ViewMetaKey: []string{"bogus"}}
	services := CreateHTTPServices(root)

	err := recoverAnalysisError(t, func() {
		services.Get("ServiceBodyMultipleView")
	})

	require.ErrorContains(t, err, `project generated response body view "bogus"`)
}

func TestEffectiveClientResponseBodyPanicsOnUnknownView(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ResultBodyMultipleViewsDSL)
	services := CreateHTTPServices(root)
	sd := services.Get("ServiceBodyMultipleView")
	require.NotNil(t, sd)
	md := sd.Service.Method("MethodBodyMultipleView")
	require.NotNil(t, md)
	require.NotNil(t, md.ViewedResult)
	endpoint := root.API.HTTP.Services[0].HTTPEndpoints[0]
	rt, ok := endpoint.MethodExpr.Result.Type.(*expr.ResultTypeExpr)
	require.True(t, ok)
	body := &expr.AttributeExpr{Type: rt}
	methodResult := &expr.AttributeExpr{
		Type: rt,
		Meta: expr.MetaExpr{expr.ViewMetaKey: []string{"bogus"}},
	}

	err := recoverAnalysisError(t, func() {
		effectiveClientResponseBody(body, methodResult, md)
	})

	require.ErrorContains(t, err, `project effective client response body view "bogus"`)
}

func TestAnalyzePanicsOnIncompatibleWebSocketStreamingBody(t *testing.T) {
	root := RunHTTPDSL(t, testdata.StreamingPayloadDSL)
	// Corrupt the derived websocket streaming body so it no longer matches the
	// streaming payload.
	root.API.HTTP.Services[0].HTTPEndpoints[0].StreamingBody.Type = expr.Int
	services := CreateHTTPServices(root)

	err := recoverAnalysisError(t, func() {
		services.Get("StreamingPayloadService")
	})

	require.ErrorContains(t, err, "build WebSocket payload transform for StreamingPayloadMethod")
}
