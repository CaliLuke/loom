package service

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestClient(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
		Code string
	}{
		{"client-single", testdata.SingleEndpointDSL, testdata.SingleMethodClient},
		{"client-use", testdata.UseEndpointDSL, testdata.UseMethodClient},
		{"client-multiple", testdata.MultipleEndpointsDSL, testdata.MultipleMethodsClient},
		{"client-no-payload", testdata.NoPayloadEndpointDSL, testdata.NoPayloadMethodsClient},
		{"client-with-result", testdata.WithResultEndpointDSL, testdata.WithResultMethodClient},
		{"client-streaming-result", testdata.StreamingResultMethodDSL, testdata.StreamingResultMethodClient},
		{"client-mixed-results", testdata.MixedResultsEndpointDSL, testdata.MixedResultsMethodClient},
		{"client-streaming-result-no-payload", testdata.StreamingResultNoPayloadMethodDSL, testdata.StreamingResultNoPayloadMethodClient},
		{"client-streaming-payload", testdata.StreamingPayloadMethodDSL, testdata.StreamingPayloadMethodClient},
		{"client-streaming-payload-no-payload", testdata.StreamingPayloadNoPayloadMethodDSL, testdata.StreamingPayloadNoPayloadMethodClient},
		{"client-streaming-payload-no-result", testdata.StreamingPayloadNoResultMethodDSL, testdata.StreamingPayloadNoResultMethodClient},
		{"client-bidirectional-streaming", testdata.BidirectionalStreamingMethodDSL, testdata.BidirectionalStreamingMethodClient},
		{"client-bidirectional-streaming-no-payload", testdata.BidirectionalStreamingNoPayloadMethodDSL, testdata.BidirectionalStreamingNoPayloadMethodClient},
		{"client-interceptor", testdata.EndpointWithClientInterceptorDSL, testdata.InterceptorClient},
		{"client-interceptor-no-method", testdata.NoMethodClientInterceptorDSL, testdata.NoMethodInterceptorClient},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := codegen.RunDSL(t, c.DSL)
			services := NewServicesData(root)
			require.Len(t, root.Services, 1)
			fs := ClientFile("test/gen", root.Services[0], services)
			require.NotNil(t, fs)
			buf := new(bytes.Buffer)
			for _, s := range fs.AllSections()[1:] {
				require.NoError(t, s.Write(buf))
			}
			code := strings.ReplaceAll(buf.String(), "\r\n", "\n")
			assert.Equal(t, canonicalGoFragment(t, c.Code), canonicalGoFragment(t, code))
		})
	}
}

func TestClientLargeErrorSetCommentsRemainValidGo(t *testing.T) {
	root := codegen.RunDSL(t, testdata.LargeErrorSetClientDSL)
	services := NewServicesData(root)
	require.Len(t, root.Services, 1)
	fs := ClientFile("test/gen", root.Services[0], services)
	require.NotNil(t, fs)
	buf := new(bytes.Buffer)
	for _, s := range fs.AllSections()[1:] {
		require.NoError(t, s.Write(buf))
	}
	code := strings.ReplaceAll(buf.String(), "\r\n", "\n")
	require.NotContains(t, code, "internal error func")
	require.Contains(t, code, `func (c *Client) ListAccounts(`)
	_ = canonicalGoFragment(t, code)
}

func canonicalGoFragment(t *testing.T, code string) string {
	t.Helper()
	wrapped := "package fragment\n\n" + code
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fragment.go", wrapped, parser.ParseComments)
	require.NoError(t, err, wrapped)
	var buf bytes.Buffer
	require.NoError(t, ast.Fprint(&buf, fset, file, func(name string, value reflect.Value) bool {
		switch value.Interface().(type) {
		case token.Pos, *ast.Object, *ast.Scope:
			return false
		default:
			return true
		}
	}))
	return buf.String()
}
