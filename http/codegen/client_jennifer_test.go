package codegen

import (
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testutil"
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
	testutil.NewGoldenFile(t, filepath.Join("testdata", "golden")).
		StringContent(structCode).
		Path("client_struct_large-types.golden").
		CompareContent()

	initCode := codegen.SectionCode(t, clientInitSection(data))
	testutil.NewGoldenFile(t, filepath.Join("testdata", "golden")).
		StringContent(initCode).
		Path("client_init_large-types.golden").
		CompareContent()

	groupCode := codegen.SectionCode(t, clientOperationGroupSection(data))
	require.Contains(t, groupCode, "type ItemsClient struct")
	require.Contains(t, groupCode, "func (g *ItemsClient) Method0() loom.Endpoint")
	testutil.NewGoldenFile(t, filepath.Join("testdata", "golden")).
		StringContent(groupCode).
		Path("client_operation_group_large-types.golden").
		CompareContent()
}

func TestNumericPathClientOperationGroupsUseValidIdentifiers(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		Service("NumericPathGroups", func() {
			for index := range 7 {
				Method(fmt.Sprintf("Method%d", index), func() {
					HTTP(func() {
						GET(fmt.Sprintf("/0/items/%d", index))
						Response(StatusOK)
					})
				})
			}
		})
	})
	services := CreateHTTPServices(root)
	data := services.Get("NumericPathGroups")

	groups := clientOperationGroups(data)
	require.Len(t, groups, 1)
	require.Equal(t, "Val0", groups[0].FieldName)
	require.Equal(t, "Val0Client", groups[0].Name)
	require.Contains(t, codegen.SectionCode(t, clientStructSection(data)), "Val0 *Val0Client")
	require.Contains(t, codegen.SectionCode(t, clientOperationGroupSection(data)), "type Val0Client struct")
	outputDir := t.TempDir()
	for _, file := range ClientFiles("gen", services) {
		_, err := file.Render(outputDir)
		require.NoError(t, err)
	}
}

// TestClientOperationGroupsAvoidClientMethodNames covers operation groups whose
// path segment Goifies to the name of a method of the client struct: the
// endpoint method, the stream method of a mixed-results endpoint or the
// request builder. A struct field and a method cannot share a name, so the
// group field takes the Operations suffix and the generated client compiles.
func TestClientOperationGroupsAvoidClientMethodNames(t *testing.T) {
	cases := []struct {
		name   string
		dsl    func()
		fields []string
	}{
		{
			name: "endpoint method",
			dsl: func() {
				Service("MethodSegments", func() {
					for _, name := range []string{"a", "b", "c", "d", "e", "f", "g"} {
						Method(name, func() {
							HTTP(func() {
								POST("/" + name)
							})
						})
					}
				})
			},
			fields: []string{"AOperations", "BOperations", "COperations", "DOperations", "EOperations", "FOperations", "GOperations"},
		},
		{
			name: "stream method and request builder",
			dsl: func() {
				Service("BuilderSegments", func() {
					Method("watch", func() {
						Result(String)
						StreamingResult(String)
						HTTP(func() {
							GET("/items/watch")
							ServerSentEvents()
						})
					})
					Method("stream", func() {
						HTTP(func() {
							GET("/watch_stream")
						})
					})
					Method("build", func() {
						HTTP(func() {
							GET("/build_stream_request")
						})
					})
					for index := range 4 {
						Method(fmt.Sprintf("method%d", index), func() {
							HTTP(func() {
								GET(fmt.Sprintf("/items/%d", index))
							})
						})
					}
				})
			},
			fields: []string{"BuildStreamRequestOperations", "Items", "WatchStreamOperations"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, tc.dsl)
			services := CreateHTTPServices(root)
			data := services.Get(root.Services[0].Name)

			fields := make([]string, 0)
			for _, group := range clientOperationGroups(data) {
				fields = append(fields, group.FieldName)
			}
			if !slices.Equal(tc.fields, fields) {
				t.Errorf("group fields = %v, want %v", fields, tc.fields)
			}

			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/operationgroups", root)
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "build", "./...")
			runGoCommand(t, dir, "vet", "./...")
		})
	}
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
	require.Contains(t, code, "wsconn := loomhttp.NewWebSocketStream(conn)")
	require.Contains(t, code, "conn: wsconn")
	require.NotContains(t, code, "<-ctx.Done()\n\t\t\tconn.WriteControl")
}
