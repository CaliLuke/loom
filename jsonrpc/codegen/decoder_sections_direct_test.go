package codegen

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/codegen/testutil"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

// TestWriteJSONRPCDecodedResponseReturnBranches covers every return shape of
// the decoded response writer: constructor results (with and without tags),
// raw client bodies and empty responses.
func TestWriteJSONRPCDecodedResponseReturnBranches(t *testing.T) {
	endpoint := &httpcodegen.EndpointData{
		ServiceName: "calc",
		Method:      &service.MethodData{Name: "show"},
	}
	cases := []struct {
		name string
		resp *httpcodegen.ResponseData
	}{
		{
			name: "result-init",
			resp: &httpcodegen.ResponseData{
				ResultInit: &httpcodegen.InitData{
					Name: "NewShowResult",
					ClientArgs: []*httpcodegen.InitArgData{
						{Ref: "&body"},
					},
				},
			},
		},
		{
			name: "result-init-tag-value",
			resp: &httpcodegen.ResponseData{
				ResultInit: &httpcodegen.InitData{
					Name: "NewShowResult",
					ClientArgs: []*httpcodegen.InitArgData{
						{Ref: "&body"},
					},
				},
				Tag: &httpcodegen.ResponseTagData{Assignment: `res.Status = "ok"`},
			},
		},
		{
			name: "result-init-tag-pointer",
			resp: &httpcodegen.ResponseData{
				ResultInit: &httpcodegen.InitData{
					Name: "NewShowResult",
					ClientArgs: []*httpcodegen.InitArgData{
						{Ref: "&body"},
					},
				},
				Tag: &httpcodegen.ResponseTagData{Assignment: `tmp := "ok"
res.Status = &tmp`},
			},
		},
		{
			name: "client-body",
			resp: &httpcodegen.ResponseData{
				ClientBody: &httpcodegen.TypeData{VarName: "string"},
			},
		},
		{
			name: "empty",
			resp: &httpcodegen.ResponseData{},
		},
	}

	var rendered strings.Builder
	for _, c := range cases {
		code := renderGroupWriter(t, func(g *jen.Group) {
			writeJSONRPCDecodedResponseReturn(g, endpoint, c.resp)
		})
		rendered.WriteString("// case: " + c.name + "\n")
		rendered.WriteString(code)
		rendered.WriteString("\n")
	}

	golden := filepath.Join("testdata", "golden", "jsonrpc-decoded-response-returns.golden")
	testutil.AssertString(t, golden, rendered.String())
}

// TestWriteJSONRPCResponseSuccessHandlingEmptyResult covers the early return
// emitted when the method defines no result responses to decode.
func TestWriteJSONRPCResponseSuccessHandlingEmptyResult(t *testing.T) {
	cases := []struct {
		name     string
		endpoint *httpcodegen.EndpointData
	}{
		{name: "nil-result", endpoint: &httpcodegen.EndpointData{}},
		{
			name: "no-responses",
			endpoint: &httpcodegen.EndpointData{
				Result: &httpcodegen.ResultData{},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code := renderGroupWriter(t, func(g *jen.Group) {
				writeJSONRPCResponseSuccessHandling(g, c.endpoint)
			})
			require.Contains(t, code, "return nil, nil")
			require.NotContains(t, code, "jresp.Result")
		})
	}
}

// TestWriteJSONRPCViewedInitReturnWithTag covers the tag assignment branch on
// viewed results, which stores the tag value on the projected value before
// wrapping it in the viewed result type.
func TestWriteJSONRPCViewedInitReturnWithTag(t *testing.T) {
	endpoint := &httpcodegen.EndpointData{
		ServiceName:    "viewer",
		ServicePkgName: "viewer",
		Method: &service.MethodData{
			Name: "show",
			MethodResultData: service.MethodResultData{
				Result: "Fixedviewresult",
				ViewedResult: &service.ViewedResultTypeData{
					UserTypeData: &service.UserTypeData{VarName: "Fixedviewresult"},
					ViewsPkg:     "viewerviews",
					ViewName:     "tiny",
					ResultInit:   &service.InitData{Name: "NewFixedviewresult"},
				},
			},
		},
	}
	resp := &httpcodegen.ResponseData{
		ResultInit: &httpcodegen.InitData{
			Name: "NewShowFixedviewresultOK",
			ClientArgs: []*httpcodegen.InitArgData{
				{Ref: "&body"},
			},
		},
		Tag: &httpcodegen.ResponseTagData{Assignment: `tmp := "ok"
p.Status = &tmp`},
		ViewedResult: endpoint.Method.ViewedResult,
	}

	code := renderGroupWriter(t, func(g *jen.Group) {
		writeJSONRPCViewedInitReturn(g, endpoint, resp, nil)
	})

	require.Contains(t, code, `tmp := "ok"`)
	require.Contains(t, code, "p.Status = &tmp")
	require.NotContains(t, code, "res.Status")
	require.Contains(t, code, `view := "tiny"`)
	require.Contains(t, code, "vres := &viewerviews.Fixedviewresult{Projected: p, View: view}")
	require.Contains(t, code, "res, err := viewer.NewFixedviewresult(vres)")
}

// TestWriteErrorCandidateReturnBranches covers the error result return writer used
// by the error decode switch.
func TestWriteErrorCandidateReturnBranches(t *testing.T) {
	cases := []struct {
		name string
		resp *httpcodegen.ResponseData
		want string
	}{
		{
			name: "result-init",
			resp: &httpcodegen.ResponseData{
				ResultInit: &httpcodegen.InitData{
					Name: "NewDivideBadRequest",
					ClientArgs: []*httpcodegen.InitArgData{
						{Ref: "&body"},
					},
				},
			},
			want: "return NewDivideBadRequest(&body), nil",
		},
		{
			name: "client-body",
			resp: &httpcodegen.ResponseData{
				ClientBody: &httpcodegen.TypeData{VarName: "DivideBadRequestResponseBody"},
			},
			want: "return body, nil",
		},
		{
			name: "empty",
			resp: &httpcodegen.ResponseData{},
			want: "return nil, nil",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code := renderGroupWriter(t, func(g *jen.Group) {
				writeErrorCandidateReturn(g, c.resp)
			})
			require.Contains(t, code, c.want)
		})
	}
}

// TestWriteJSONRPCResponseTagPlans checks shared tag plans at the decoded return boundary.
func TestWriteJSONRPCResponseTagPlans(t *testing.T) {
	cases := []struct {
		name        string
		resp        *httpcodegen.ResponseData
		want        []string
		wantMissing []string
	}{
		{
			name: "no-tag",
			resp: &httpcodegen.ResponseData{},
			wantMissing: []string{
				"res.",
			},
		},
		{
			name: "value-tag",
			resp: &httpcodegen.ResponseData{
				Tag: &httpcodegen.ResponseTagData{Assignment: `res.Status = "ok"`},
			},
			want: []string{`res.Status = "ok"`},
		},
		{
			name: "pointer-tag",
			resp: &httpcodegen.ResponseData{
				Tag: &httpcodegen.ResponseTagData{Assignment: `tmp := "ok"
res.Status = &tmp`},
			},
			want: []string{`tmp := "ok"`, `res.Status = &tmp`},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.resp.ResultInit = &httpcodegen.InitData{Name: "NewResult"}
			code := renderGroupWriter(t, func(g *jen.Group) {
				writeJSONRPCDecodedInitReturn(g, &httpcodegen.EndpointData{}, c.resp)
			})
			for _, want := range c.want {
				require.Contains(t, code, want)
			}
			for _, missing := range c.wantMissing {
				require.NotContains(t, code, missing)
			}
		})
	}
}

// TestDecoderScalarHelpers covers the small pure helpers used by the decoder
// section writers.
func TestDecoderScalarHelpers(t *testing.T) {
	t.Run("viewedResultPrefix", func(t *testing.T) {
		require.Equal(t, "", viewedResultPrefix(nil))
		require.Equal(t, "", viewedResultPrefix(&service.ViewedResultTypeData{IsCollection: true}))
		require.Equal(t, "&", viewedResultPrefix(&service.ViewedResultTypeData{}))
	})
}
