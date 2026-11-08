package codegen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestInformationalClientSymbols(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		testdata.InformationalServer(false)
	})
	data := CreateHTTPServices(root).Get("selection")
	endpoint := data.Endpoint("read")
	require.NotEmpty(t, endpoint.InformationalObservers)
	seen := make(map[string]bool)
	for _, info := range endpoint.Informational {
		for _, name := range []string{info.Observe, info.ObserveKey, info.Decode} {
			require.NotEmpty(t, name)
			require.False(t, seen[name], "unique private and public occurrence symbols")
			seen[name] = true
		}
	}
	require.Empty(t, data.Endpoint("mapped").InformationalObservers)
}

func TestInformationalClientDeclarations(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		testdata.InformationalServer(false)
	})
	data := CreateHTTPServices(root).Get("selection")
	endpoint := data.Endpoint("read")
	actual := codegen.SectionsCode(t, informationalClientSections(endpoint, data))
	require.Contains(t, actual, "func "+endpoint.Informational[0].Observe+"(ctx context.Context, fn func(context.Context, *selection.Hints, error) error) context.Context")
	require.Contains(t, actual, "[]loomhttp.InformationalObserver")
	require.Equal(t, len(endpoint.Informational), strings.Count(actual, "Selector:"))
	require.NotContains(t, actual, "ObserveInformational(")
	require.NotContains(t, actual, "httptrace")
	rpc := *endpoint
	rpc.IsJSONRPC = true
	require.Empty(t, informationalClientSections(&rpc, data))
	require.Empty(t, informationalClientSections(data.Endpoint("mapped"), data))
}

func TestInformationalHeaderDecodeSource(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		testdata.InformationalServer(false)
	})
	response := CreateHTTPServices(root).Get("selection").Endpoint("read").Informational[0].Response
	for _, header := range response.Headers {
		if !header.Styled {
			continue
		}
		t.Run(header.Name, func(t *testing.T) {
			ordinary := renderStyledDecode(header.Element, "response-header", "resp.Header")
			interim := renderStyledDecode(header.Element, "response-header", "informationalHeaders")
			require.Contains(t, interim, "informationalHeaders")
			require.NotContains(t, interim, "resp.Header")
			require.Equal(t, ordinary, strings.ReplaceAll(interim, "informationalHeaders", "resp.Header"))
		})
	}
}

func TestInformationalConstructorStatusArgument(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		testdata.InformationalServer(false)
	})
	response := CreateHTTPServices(root).Get("selection").Endpoint("read").Informational[0].Response
	require.NotNil(t, response.ResultInit)
	args := response.ResultInit.ClientArgs
	require.NotEmpty(t, args)
	refs := responseConstructorArgRefs(args, "actualStatus")
	require.Equal(t, "actualStatus", refs[len(refs)-1])
	for i := range len(args) - 1 {
		require.Equal(t, args[i].Ref, refs[i])
	}
	// A ref with identical spelling is not a status argument without its role.
	refs = responseConstructorArgRefs([]*InitArgData{{Ref: "resp.StatusCode"}}, "actualStatus")
	require.Equal(t, []string{"resp.StatusCode"}, refs)
}
