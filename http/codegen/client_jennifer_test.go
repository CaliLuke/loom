package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestClientStructSection(t *testing.T) {
	t.Run("multiple endpoints", func(t *testing.T) {
		root := RunHTTPDSL(t, testdata.ServerMultiEndpointsDSL)
		services := CreateHTTPServices(root)

		code := codegen.SectionCode(t, clientStructSection(services.Get("ServiceMultiEndpoints")))
		require.Contains(t, code, "MethodMultiEndpoints1Doer loomhttp.Doer")
		require.Contains(t, code, "MethodMultiEndpoints2Doer loomhttp.Doer")
		require.NotContains(t, code, "dialer loomhttp.Dialer")
	})

	t.Run("streaming websocket fields", func(t *testing.T) {
		root := RunHTTPDSL(t, testdata.StreamingResultDSL)
		services := CreateHTTPServices(root)

		code := codegen.SectionCode(t, clientStructSection(services.Get("StreamingResultService")))
		require.Contains(t, code, "loomhttp.Dialer")
		require.Contains(t, code, "configurer *ConnConfigurer")
	})
}

func TestLargeClientOperationGroups(t *testing.T) {
	root := RunHTTPDSL(t, largeTypeFileDSL)
	services := CreateHTTPServices(root)
	data := services.Get("LargeTypes")

	structCode := codegen.SectionCode(t, clientStructSection(data))
	require.Contains(t, structCode, "Items *ItemsClient")

	initCode := codegen.SectionCode(t, clientInitSection(data))
	require.Contains(t, initCode, "client := &Client{")
	require.Contains(t, initCode, "client.Items = &ItemsClient{client: client}")

	groupCode := codegen.SectionCode(t, clientOperationGroupSection(data))
	require.Contains(t, groupCode, "type ItemsClient struct")
	require.Contains(t, groupCode, "func (g *ItemsClient) Method0() loom.Endpoint")
	require.Contains(t, groupCode, "return g.client.Method0()")
	require.Contains(t, groupCode, "func (g *ItemsClient) BuildMethod0Request(ctx context.Context, v any) (*http.Request, error)")
	require.Contains(t, groupCode, "return g.client.BuildMethod0Request(ctx, v)")
}

func TestClientEndpointSectionsMixedResults(t *testing.T) {
	root := RunHTTPDSL(t, testdata.MixedResultsDSL)
	services := CreateHTTPServices(root)
	endpoint := services.Get("MixedResultsService").Endpoints[0]

	sections := clientEndpointSections(endpoint)
	require.Len(t, sections, 2)

	standard := codegen.SectionCode(t, sections[0])
	stream := codegen.SectionCode(t, sections[1])

	require.Contains(t, standard, "func (c *Client) Create() loom.Endpoint")
	require.NotContains(t, standard, `req.Header.Set("Accept", "text/event-stream")`)
	require.Contains(t, stream, "func (c *Client) CreateStream() loom.Endpoint")
	require.Contains(t, stream, `req.Header.Set("Accept", "text/event-stream")`)
}

func TestClientEndpointSectionSSEDecodesTypedErrorResponses(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		Service("SSEErrorService", func() {
			Method("SSEErrorMethod", func() {
				StreamingResult(String)
				Error("unauthorized")
				HTTP(func() {
					GET("/sse-error")
					Response(StatusOK)
					Response("unauthorized", StatusUnauthorized)
					ServerSentEvents()
				})
			})
		})
	})
	services := CreateHTTPServices(root)
	endpoint := services.Get("SSEErrorService").Endpoints[0]

	code := codegen.SectionCode(t, clientEndpointSection(endpoint))

	require.Contains(t, code, "decodeResponse = DecodeSSEErrorMethodResponse(c.decoder, c.RestoreResponseBody)")
	require.Contains(t, code, "return decodeResponse(resp)")
	require.NotContains(t, code, "unexpected status from SSE endpoint")
}

func TestClientWebSocketServerStreamingEndpointDoesNotLeakContextWatcher(t *testing.T) {
	root := RunHTTPDSL(t, testdata.StreamingResultDSL)
	services := CreateHTTPServices(root)
	endpoint := services.Get("StreamingResultService").Endpoints[0]

	code := codegen.SectionCode(t, clientEndpointSection(endpoint))

	require.Contains(t, code, "done := make(chan struct{})")
	require.Contains(t, code, "case <-done:")
	require.Contains(t, code, "done: done")
	require.NotContains(t, code, "<-ctx.Done()\n\t\t\tconn.WriteControl")
}
