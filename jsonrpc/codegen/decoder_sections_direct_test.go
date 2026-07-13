package codegen

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/expr"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

// TestWriteResponseHeaderDecodeGolden covers every branch of the response
// header decoding writer: string headers (required, optional, defaulted),
// string slices, scalar slices, byte slices, and scalar conversions.
func TestWriteResponseHeaderDecodeGolden(t *testing.T) {
	cases := []struct {
		name   string
		header *httpcodegen.HeaderData
	}{
		{
			name: "string-required",
			header: makeHeader(&httpcodegen.AttributeData{
				Name:     "token",
				VarName:  "token",
				Type:     expr.String,
				TypeRef:  "string",
				Required: true,
			}, "X-Token", false, false),
		},
		{
			name: "string-pointer-required",
			header: makeHeader(&httpcodegen.AttributeData{
				Name:     "token",
				VarName:  "token",
				Type:     expr.String,
				TypeRef:  "*string",
				Required: true,
				Pointer:  true,
			}, "X-Token", false, false),
		},
		{
			name: "string-optional-default",
			header: makeHeader(&httpcodegen.AttributeData{
				Name:         "region",
				VarName:      "region",
				Type:         expr.String,
				TypeRef:      "string",
				DefaultValue: "us-east",
			}, "X-Region", false, false),
		},
		{
			name: "string-optional-no-default",
			header: makeHeader(&httpcodegen.AttributeData{
				Name:    "trace",
				VarName: "trace",
				Type:    expr.String,
				TypeRef: "*string",
				Pointer: true,
			}, "X-Trace", false, false),
		},
		{
			name: "string-slice-required",
			header: makeHeader(&httpcodegen.AttributeData{
				Name:     "tags",
				VarName:  "tags",
				Type:     &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}},
				TypeRef:  "[]string",
				Required: true,
			}, "X-Tags", true, true),
		},
		{
			name: "string-slice-optional",
			header: makeHeader(&httpcodegen.AttributeData{
				Name:    "tags",
				VarName: "tags",
				Type:    &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}},
				TypeRef: "[]string",
			}, "X-Tags", true, true),
		},
		{
			name: "int-slice-required",
			header: makeHeader(&httpcodegen.AttributeData{
				Name:     "ids",
				VarName:  "ids",
				Type:     &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Int}},
				TypeRef:  "[]int",
				Required: true,
			}, "X-Ids", false, true),
		},
		{
			name: "bytes-slice-optional",
			header: makeHeader(&httpcodegen.AttributeData{
				Name:    "blobs",
				VarName: "blobs",
				Type:    &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Bytes}},
				TypeRef: "[][]byte",
			}, "X-Blobs", false, true),
		},
		{
			name: "any-slice-passthrough",
			header: makeHeader(&httpcodegen.AttributeData{
				Name:    "vals",
				VarName: "vals",
				Type:    &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Any}},
				TypeRef: "[]any",
			}, "X-Vals", false, true),
		},
		{
			name: "int-required",
			header: makeHeader(&httpcodegen.AttributeData{
				Name:     "count",
				VarName:  "count",
				Type:     expr.Int,
				TypeRef:  "int",
				Required: true,
			}, "X-Count", false, false),
		},
		{
			name: "bytes-optional",
			header: makeHeader(&httpcodegen.AttributeData{
				Name:    "raw",
				VarName: "raw",
				Type:    expr.Bytes,
				TypeRef: "[]byte",
			}, "X-Raw", false, false),
		},
	}

	var rendered strings.Builder
	for _, c := range cases {
		code := renderGroupWriter(t, func(g *jen.Group) {
			writeResponseHeaderDecode(g, c.header)
		})
		rendered.WriteString("// case: " + c.name + "\n")
		rendered.WriteString(code)
		rendered.WriteString("\n")
	}

	golden := filepath.Join("testdata", "golden", "jsonrpc-response-header-decode.golden")
	testutil.AssertString(t, golden, rendered.String())
}

func TestWriteResponseHeaderDecodeRejectsImpossibleShapes(t *testing.T) {
	cases := []struct {
		name   string
		header *httpcodegen.HeaderData
		want   string
	}{
		{
			name: "slice flag on non-array",
			header: makeHeader(&httpcodegen.AttributeData{
				Name:    "odd",
				VarName: "odd",
				Type:    expr.Int,
				TypeRef: "[]int",
			}, "X-Odd", false, true),
			want: `decode JSON-RPC response field "odd": slice conversion requires an array type, got int`,
		},
		{
			name: "unsupported map",
			header: makeHeader(&httpcodegen.AttributeData{
				Name:    "meta",
				VarName: "meta",
				Type: &expr.Map{
					KeyType:  &expr.AttributeExpr{Type: expr.String},
					ElemType: &expr.AttributeExpr{Type: expr.String},
				},
				TypeRef: "map[string]string",
			}, "X-Meta", false, false),
			want: `decode JSON-RPC response field "meta": unsupported type map`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.PanicsWithError(t, tc.want, func() {
				renderGroupWriter(t, func(g *jen.Group) {
					writeResponseHeaderDecode(g, tc.header)
				})
			})
		})
	}
}

// TestWriteSingleResponseDecodeElements covers the header and cookie block
// writers reached through writeSingleResponseDecode, including the trailing
// MustValidate error check and validation snippets.
func TestWriteSingleResponseDecodeElements(t *testing.T) {
	method := &service.MethodData{Name: "show"}
	cases := []struct {
		name string
		data *httpcodegen.ResponseData
	}{
		{
			name: "headers-with-validation",
			data: &httpcodegen.ResponseData{
				Headers: []*httpcodegen.HeaderData{
					makeHeaderWithValidate(&httpcodegen.AttributeData{
						Name:     "token",
						VarName:  "token",
						Type:     expr.String,
						TypeRef:  "string",
						Required: true,
					}, "X-Token", "err = loom.MergeErrors(err, nil)"),
					makeHeader(&httpcodegen.AttributeData{
						Name:    "trace",
						VarName: "trace",
						Type:    expr.String,
						TypeRef: "string",
					}, "X-Trace", false, false),
				},
				MustValidate: true,
			},
		},
		{
			name: "cookies-string",
			data: &httpcodegen.ResponseData{
				Cookies: []*httpcodegen.CookieData{
					makeCookie(&httpcodegen.AttributeData{
						Name:     "session",
						VarName:  "session",
						Type:     expr.String,
						TypeRef:  "string",
						Required: true,
					}, "session"),
					makeCookie(&httpcodegen.AttributeData{
						Name:    "theme",
						VarName: "theme",
						Type:    expr.String,
						TypeRef: "string",
					}, "theme"),
				},
				MustValidate: true,
			},
		},
		{
			name: "cookies-scalar-and-validate",
			data: &httpcodegen.ResponseData{
				Cookies: []*httpcodegen.CookieData{
					makeCookieWithValidate(&httpcodegen.AttributeData{
						Name:     "count",
						VarName:  "count",
						Type:     expr.Int,
						TypeRef:  "int",
						Required: true,
					}, "count", "err = loom.MergeErrors(err, nil)"),
					makeCookie(&httpcodegen.AttributeData{
						Name:    "ratio",
						VarName: "ratio",
						Type:    expr.Float64,
						TypeRef: "float64",
					}, "ratio"),
				},
			},
		},
		{
			name: "headers-and-cookies",
			data: &httpcodegen.ResponseData{
				Headers: []*httpcodegen.HeaderData{
					makeHeader(&httpcodegen.AttributeData{
						Name:    "trace",
						VarName: "trace",
						Type:    expr.String,
						TypeRef: "string",
					}, "X-Trace", false, false),
				},
				Cookies: []*httpcodegen.CookieData{
					makeCookie(&httpcodegen.AttributeData{
						Name:    "session",
						VarName: "session",
						Type:    expr.String,
						TypeRef: "string",
					}, "session"),
				},
				MustValidate: true,
			},
		},
	}

	var rendered strings.Builder
	for _, c := range cases {
		code := renderGroupWriter(t, func(g *jen.Group) {
			writeSingleResponseDecode(g, c.data, "calc", method)
		})
		rendered.WriteString("// case: " + c.name + "\n")
		rendered.WriteString(code)
		rendered.WriteString("\n")
	}

	golden := filepath.Join("testdata", "golden", "jsonrpc-response-elements-decode.golden")
	testutil.AssertString(t, golden, rendered.String())
}

// TestScalarConversionWriters covers the strconv-based conversion writers for
// every supported scalar type, in both the query-style and slice-item forms,
// including pointer targets and aliased type references.
func TestScalarConversionWriters(t *testing.T) {
	type conversionCase struct {
		name     string
		typeName string
		attr     *httpcodegen.AttributeData
	}
	scalarCases := []conversionCase{
		{name: "int", typeName: "int", attr: &httpcodegen.AttributeData{Name: "v", VarName: "v", TypeRef: "int"}},
		{name: "int32", typeName: "int32", attr: &httpcodegen.AttributeData{Name: "v", VarName: "v", TypeRef: "int32"}},
		{name: "int64", typeName: "int64", attr: &httpcodegen.AttributeData{Name: "v", VarName: "v", TypeRef: "int64"}},
		{name: "uint", typeName: "uint", attr: &httpcodegen.AttributeData{Name: "v", VarName: "v", TypeRef: "uint"}},
		{name: "uint32", typeName: "uint32", attr: &httpcodegen.AttributeData{Name: "v", VarName: "v", TypeRef: "uint32"}},
		{name: "uint64", typeName: "uint64", attr: &httpcodegen.AttributeData{Name: "v", VarName: "v", TypeRef: "uint64"}},
		{name: "float32", typeName: "float32", attr: &httpcodegen.AttributeData{Name: "v", VarName: "v", TypeRef: "float32"}},
		{name: "float64", typeName: "float64", attr: &httpcodegen.AttributeData{Name: "v", VarName: "v", TypeRef: "float64"}},
		{name: "boolean", typeName: "boolean", attr: &httpcodegen.AttributeData{Name: "v", VarName: "v", TypeRef: "bool"}},
		{name: "bytes", typeName: "bytes", attr: &httpcodegen.AttributeData{Name: "v", VarName: "v", TypeRef: "[]byte"}},
		{name: "int-pointer", typeName: "int", attr: &httpcodegen.AttributeData{Name: "v", VarName: "v", TypeRef: "*int", Pointer: true}},
		{name: "int32-alias", typeName: "int32", attr: &httpcodegen.AttributeData{Name: "v", VarName: "v", TypeRef: "MyInt32"}},
		{name: "int64-alias", typeName: "int64", attr: &httpcodegen.AttributeData{Name: "v", VarName: "v", TypeRef: "MyInt64"}},
		{name: "int64-alias-pointer", typeName: "int64", attr: &httpcodegen.AttributeData{Name: "v", VarName: "v", TypeRef: "*MyInt64", Pointer: true}},
		{name: "boolean-pointer", typeName: "boolean", attr: &httpcodegen.AttributeData{Name: "v", VarName: "v", TypeRef: "*bool", Pointer: true}},
	}

	var rendered strings.Builder
	for _, c := range scalarCases {
		queryCode := renderGroupWriter(t, func(g *jen.Group) {
			handled := writeScalarQueryTypeConversion(g, c.attr, c.typeName)
			require.Truef(t, handled, "query conversion for %s must be handled", c.name)
		})
		rendered.WriteString("// query case: " + c.name + "\n")
		rendered.WriteString(queryCode)
		rendered.WriteString("\n")

		itemCode := renderGroupWriter(t, func(g *jen.Group) {
			handled := writeScalarSliceItemConversion(g, c.attr, c.typeName)
			require.Truef(t, handled, "slice item conversion for %s must be handled", c.name)
		})
		rendered.WriteString("// slice item case: " + c.name + "\n")
		rendered.WriteString(itemCode)
		rendered.WriteString("\n")
	}

	unsupported := &httpcodegen.AttributeData{Name: "v", VarName: "v", TypeRef: "string"}
	require.False(t, writeScalarQueryTypeConversion(jen.NewFile("x").Group, unsupported, "string"))
	require.False(t, writeScalarSliceItemConversion(jen.NewFile("x").Group, unsupported, "string"))

	golden := filepath.Join("testdata", "golden", "jsonrpc-scalar-conversions.golden")
	testutil.AssertString(t, golden, rendered.String())
}

// TestWriteJSONRPCDecodedResponseReturnBranches covers every return shape of
// the decoded response writer: constructor results (with and without tags),
// raw client bodies, header values, cookie values and empty responses.
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
				TagName:  "Status",
				TagValue: "ok",
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
				TagName:    "Status",
				TagValue:   "ok",
				TagPointer: true,
			},
		},
		{
			name: "client-body",
			resp: &httpcodegen.ResponseData{
				ClientBody: &httpcodegen.TypeData{VarName: "string"},
			},
		},
		{
			name: "header-value",
			resp: &httpcodegen.ResponseData{
				Headers: []*httpcodegen.HeaderData{
					makeHeader(&httpcodegen.AttributeData{
						Name:    "location",
						VarName: "location",
						Type:    expr.String,
						TypeRef: "string",
					}, "Location", false, false),
				},
			},
		},
		{
			name: "cookie-value",
			resp: &httpcodegen.ResponseData{
				Cookies: []*httpcodegen.CookieData{
					makeCookie(&httpcodegen.AttributeData{
						Name:    "session",
						VarName: "session",
						Type:    expr.String,
						TypeRef: "string",
					}, "session"),
				},
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
		TagName:      "Status",
		TagValue:     "ok",
		ViewedResult: endpoint.Method.ViewedResult,
	}

	code := renderGroupWriter(t, func(g *jen.Group) {
		writeJSONRPCViewedInitReturn(g, endpoint, resp)
	})

	require.Contains(t, code, `tmp := "ok"`)
	require.Contains(t, code, "p.Status = &tmp")
	require.Contains(t, code, `view := "tiny"`)
	require.Contains(t, code, "vres := &viewerviews.Fixedviewresult{Projected: p, View: view}")
	require.Contains(t, code, "res, err := viewer.NewFixedviewresult(vres)")
}

// TestWriteResultInitReturnBranches covers the error result return writer used
// by the error decode switch.
func TestWriteResultInitReturnBranches(t *testing.T) {
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
			want: "return nil, NewDivideBadRequest(&body)",
		},
		{
			name: "client-body",
			resp: &httpcodegen.ResponseData{
				ClientBody: &httpcodegen.TypeData{VarName: "DivideBadRequestResponseBody"},
			},
			want: "return nil, body",
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
				writeResultInitReturn(g, c.resp)
			})
			require.Contains(t, code, c.want)
		})
	}
}

// TestWriteJSONRPCResponseTagAssignmentBranches covers the tag assignment
// writer, including the viewed-response skip.
func TestWriteJSONRPCResponseTagAssignmentBranches(t *testing.T) {
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
			name: "viewed-skips-assignment",
			resp: &httpcodegen.ResponseData{
				TagName:      "Status",
				TagValue:     "ok",
				ViewedResult: &service.ViewedResultTypeData{},
			},
			wantMissing: []string{
				`res.Status`,
			},
		},
		{
			name: "value-tag",
			resp: &httpcodegen.ResponseData{
				TagName:  "Status",
				TagValue: "ok",
			},
			want: []string{`res.Status = "ok"`},
		},
		{
			name: "pointer-tag",
			resp: &httpcodegen.ResponseData{
				TagName:    "Status",
				TagValue:   "ok",
				TagPointer: true,
			},
			want: []string{`tmp := "ok"`, `res.Status = &tmp`},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code := renderGroupWriter(t, func(g *jen.Group) {
				writeJSONRPCResponseTagAssignment(g, c.resp)
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
	t.Run("literalValue", func(t *testing.T) {
		cases := []struct {
			name     string
			typeName string
			value    any
			want     string
		}{
			{name: "string", typeName: "string", value: "us-east", want: `"us-east"`},
			{name: "int", typeName: "int", value: 42, want: "42"},
			{name: "bool", typeName: "boolean", value: true, want: "true"},
		}
		for _, c := range cases {
			require.Equal(t, c.want, literalValue(c.typeName, c.value), c.name)
		}
	})

	t.Run("stringPointerPrefix", func(t *testing.T) {
		cases := []struct {
			name     string
			typeName string
			pointer  bool
			want     string
		}{
			{name: "string-pointer", typeName: "string", pointer: true, want: "&"},
			{name: "string-value", typeName: "string", pointer: false, want: ""},
			{name: "any-pointer", typeName: "any", pointer: true, want: ""},
		}
		for _, c := range cases {
			require.Equal(t, c.want, stringPointerPrefix(c.typeName, c.pointer), c.name)
		}
	})

	t.Run("isViewedResponse", func(t *testing.T) {
		require.False(t, isViewedResponse(&httpcodegen.ResponseData{}))
		require.True(t, isViewedResponse(&httpcodegen.ResponseData{
			ViewedResult: &service.ViewedResultTypeData{},
		}))
	})

	t.Run("viewedResultPrefix", func(t *testing.T) {
		require.Equal(t, "", viewedResultPrefix(nil))
		require.Equal(t, "", viewedResultPrefix(&service.ViewedResultTypeData{IsCollection: true}))
		require.Equal(t, "&", viewedResultPrefix(&service.ViewedResultTypeData{}))
	})
}

func makeHeader(attr *httpcodegen.AttributeData, httpName string, stringSlice, slice bool) *httpcodegen.HeaderData {
	return &httpcodegen.HeaderData{
		Element: &httpcodegen.Element{
			AttributeData: attr,
			HTTPName:      httpName,
			StringSlice:   stringSlice,
			Slice:         slice,
		},
		CanonicalName: httpName,
	}
}

func makeHeaderWithValidate(attr *httpcodegen.AttributeData, httpName, validate string) *httpcodegen.HeaderData {
	attr.Validate = validate
	return makeHeader(attr, httpName, false, false)
}

func makeCookie(attr *httpcodegen.AttributeData, httpName string) *httpcodegen.CookieData {
	return &httpcodegen.CookieData{
		Element: &httpcodegen.Element{
			AttributeData: attr,
			HTTPName:      httpName,
		},
	}
}

func makeCookieWithValidate(attr *httpcodegen.AttributeData, httpName, validate string) *httpcodegen.CookieData {
	attr.Validate = validate
	return makeCookie(attr, httpName)
}
