package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/v3/codegen"
	"github.com/CaliLuke/loom/v3/http/codegen/testdata"
)

func TestClientStructSection(t *testing.T) {
	t.Run("multiple endpoints", func(t *testing.T) {
		root := RunHTTPDSL(t, testdata.ServerMultiEndpointsDSL)
		services := CreateHTTPServices(root)

		code := codegen.SectionCode(t, clientStructSection(services.Get("ServiceMultiEndpoints")))
		require.Contains(t, code, "MethodMultiEndpoints1Doer goahttp.Doer")
		require.Contains(t, code, "MethodMultiEndpoints2Doer goahttp.Doer")
		require.NotContains(t, code, "dialer goahttp.Dialer")
	})

	t.Run("streaming websocket fields", func(t *testing.T) {
		root := RunHTTPDSL(t, testdata.StreamingResultDSL)
		services := CreateHTTPServices(root)

		code := codegen.SectionCode(t, clientStructSection(services.Get("StreamingResultService")))
		require.Contains(t, code, "goahttp.Dialer")
		require.Contains(t, code, "configurer *ConnConfigurer")
	})
}

func TestClientEndpointSectionsMixedResults(t *testing.T) {
	root := RunHTTPDSL(t, testdata.MixedResultsDSL)
	services := CreateHTTPServices(root)
	endpoint := services.Get("MixedResultsService").Endpoints[0]

	sections := clientEndpointSections(endpoint)
	require.Len(t, sections, 2)

	standard := codegen.SectionCode(t, sections[0])
	stream := codegen.SectionCode(t, sections[1])

	require.Contains(t, standard, "func (c *Client) Create() goa.Endpoint")
	require.NotContains(t, standard, `req.Header.Set("Accept", "text/event-stream")`)
	require.Contains(t, stream, "func (c *Client) CreateStream() goa.Endpoint")
	require.Contains(t, stream, `req.Header.Set("Accept", "text/event-stream")`)
}
