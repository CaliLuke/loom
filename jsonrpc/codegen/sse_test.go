package codegen

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/jsonrpc/codegen/testdata"
)

func TestJSONRPCSSE(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"string", testdata.JSONRPCSSEStringDSL},
		{"object", testdata.JSONRPCSSEObjectDSL},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunJSONRPCDSL(t, c.DSL)
			services := CreateJSONRPCServices(root)

			// Generate SSE files
			fs := SSEServerFiles("", services)
			require.NotEmpty(t, fs, "expected SSE files to be generated")

			// Debug: print all generated files
			for _, f := range fs {
				t.Logf("Generated file: %s", f.Path)
			}

			// Find the server stream file
			var serverStreamFile *codegen.File
			for _, f := range fs {
				if filepath.Base(f.Path) == "stream.go" && filepath.Base(filepath.Dir(f.Path)) == "server" {
					serverStreamFile = f
					break
				}
			}
			require.NotNil(t, serverStreamFile, "server stream file not found")

			// Find the jsonrpc-sse-server-stream section
			var streamSection codegen.Section
			for _, s := range serverStreamFile.AllSections() {
				if s.SectionName() == "jsonrpc-sse-server-stream" {
					streamSection = s
					break
				}
			}
			require.NotNil(t, streamSection, "jsonrpc-sse-server-stream section not found")

			// Compare with golden file
			code := codegen.SectionCode(t, streamSection)
			require.NotContains(t, code, `sendSSEEvent("notification",`)
			require.NotContains(t, code, `sendSSEEvent("response", message)`)
			require.Equal(t, 2, strings.Count(code, `sendSSEEvent("message", message)`))
			golden := filepath.Join("testdata", "golden", "jsonrpc-sse-"+c.Name+".golden")
			testutil.AssertGo(t, golden, code)

			// Find the client stream file/section and verify it accepts MCP-compatible
			// default or "message" SSE events while remaining backward compatible.
			var clientStreamFile *codegen.File
			for _, f := range fs {
				if filepath.Base(f.Path) == "stream.go" && filepath.Base(filepath.Dir(f.Path)) == "client" {
					clientStreamFile = f
					break
				}
			}
			require.NotNil(t, clientStreamFile, "client stream file not found")

			var clientSection codegen.Section
			for _, s := range clientStreamFile.AllSections() {
				if s.SectionName() == "jsonrpc-sse-client-stream" {
					clientSection = s
					break
				}
			}
			require.NotNil(t, clientSection, "jsonrpc-sse-client-stream section not found")

			clientCode := codegen.SectionCode(t, clientSection)
			require.Contains(t, clientCode, `case "", "message":`)
			clientGolden := filepath.Join("testdata", "golden", "jsonrpc-sse-client-"+c.Name+".golden")
			testutil.AssertGo(t, clientGolden, clientCode)
		})
	}
}

func TestJSONRPCSSEServiceStreamUsesTypedResponseEvents(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEObjectDSL)
	serviceStreamCode := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "sse.go", "jsonrpc-server-sse-stream-impl")

	require.NotContains(t, serviceStreamCode, `eventType = "response"`)
	require.NotContains(t, serviceStreamCode, `var eventType string`)
	require.Contains(t, serviceStreamCode, `return s.sendSSEEvent("message", message)`)
	require.Contains(t, serviceStreamCode, `return s.sendSSEEvent("message", response)`)
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-sse-stream-impl-object.golden"), serviceStreamCode)
}

func TestJSONRPCSSEEndpointStreamsRemainLazyByDefault(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEObjectDSL)
	services := CreateJSONRPCServices(root)

	endpointStreamCode := fileSectionCode(t, SSEServerFiles("", services), "stream.go", "jsonrpc-sse-server-stream")
	require.Contains(t, endpointStreamCode, `func (s *StreamServerStream) Open(ctx context.Context) error {`)
	require.Contains(t, endpointStreamCode, `func (s *StreamServerStream) SendComment(ctx context.Context, text string) error {`)
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-sse-object.golden"), endpointStreamCode)

	handlerInitCode := fileSectionCode(t, ServerFiles("", services), "server.go", "jsonrpc-server-handler-init")
	require.NotContains(t, handlerInitCode, `if err := strm.open(); err != nil {
			return err
		}
		decodeParams :=`)
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-sse-handler-init-object.golden"), handlerInitCode)
}

// TestJSONRPCSSEStreamSuppressedFinalResponseIsObservable asserts that the
// generated SendAndClose makes its ID-less suppression branch observable:
// when a stream carries no JSON-RPC request ID (a notification or the raw
// GET events/stream listener) the final value is discarded per protocol
// rules, and the generated code must emit a transport event instead of
// dropping the data silently.
func TestJSONRPCSSEStreamSuppressedFinalResponseIsObservable(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEObjectDSL)
	code := fileSectionCode(t, SSEServerFiles("", CreateJSONRPCServices(root)), "stream.go", "jsonrpc-sse-server-stream")

	require.Contains(t, code, "if !s.requestHasID {")
	require.Contains(t, code, "loomtransport.ReasonStreamFinalResponseSuppressed")
	require.Contains(t, code, "loomtransport.EventKindStreamClose")
}

func TestJSONRPCSSEEventsStreamGETOpensBeforeFirstFrame(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEEventsStreamDSL)
	handlerInitCode := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-handler-init")

	require.Contains(t, handlerInitCode, `if r.Method == http.MethodGet && req.Method == "events/stream" {`)
	require.Contains(t, handlerInitCode, `if err := strm.Open(r.Context()); err != nil {`)
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-sse-handler-init-events-stream.golden"), handlerInitCode)
}

func TestJSONRPCMixedServerHandler(t *testing.T) {
	cases := []struct {
		Name        string
		DSL         func()
		Golden      string
		Contains    []string
		NotContains []string
	}{
		{
			Name:   "routes by method",
			DSL:    jsonrpcMixedInitializeAndEventsStreamDSL,
			Golden: "jsonrpc-mixed-server-handler.golden",
			Contains: []string{
				`switch req.Method {`,
				`case http.MethodGet:`,
			},
			NotContains: []string{
				`"events-stream"`,
				"if strings.Contains(accept, \"text/event-stream\") {\n\t\ts.handleSSE(w, r)\n\t\treturn\n\t}",
			},
		},
		{
			Name:   "avoids full body read for negotiation",
			DSL:    jsonrpcMixedInitializeAndEventsStreamDSL,
			Golden: "jsonrpc-mixed-server-handler.golden",
			Contains: []string{
				`reader.Peek(1)`,
			},
			NotContains: []string{
				`io.ReadAll(r.Body)`,
			},
		},
		{
			Name:   "groups multiple sse methods",
			DSL:    jsonrpcMixedMultipleSSEMethodsDSL,
			Golden: "jsonrpc-mixed-server-handler-multi-sse.golden",
			Contains: []string{
				`case "tools/call":`,
				`case "events/stream":`,
			},
			NotContains: []string{
				"case \"tools/call\":\n\t\tcase \"events/stream\":",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunJSONRPCDSL(t, c.DSL)
			code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-mixed-server-handler")
			for _, want := range c.Contains {
				require.Contains(t, code, want)
			}
			for _, unwanted := range c.NotContains {
				require.NotContains(t, code, unwanted)
			}
			testutil.AssertGo(t, filepath.Join("testdata", "golden", c.Golden), code)
		})
	}
}

func TestJSONRPCSSEOnlyHandlerOmitsEmptyNotificationSwitch(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEObjectDSL)
	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-sse-server-handler")

	require.NotContains(t, code, "switch req.Method {\n\t}")
	require.Contains(t, code, `jsonrpcEnvelopeDecodeError(err)`)
}

func TestJSONRPCMixedHandlerClassifiesEnvelopeDecodeErrors(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcMixedInitializeAndEventsStreamDSL)
	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-mixed-server-handler")

	require.Contains(t, code, `jsonrpcEnvelopeDecodeError(err)`)
}

// TestJSONRPCMixedHandlerStreamsEnvelopeDecodeErrorsOverSSE asserts that once
// SSE is negotiated, envelope decode failures are streamed as SSE message
// events instead of being encoded as plain JSON HTTP bodies.
func TestJSONRPCMixedHandlerStreamsEnvelopeDecodeErrorsOverSSE(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcMixedInitializeAndEventsStreamDSL)
	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-mixed-server-handler")

	require.Contains(t, code, `writer := loomhttp.NewSSEStreamWriter(w, r.Context(), loomtransport.TransportJSONRPC, s.streamWritePolicy)`)
	require.Contains(t, code, `loomhttp.WriteJSONSSEEvent(w, loomhttp.SSEMessage{Type: "message"}, response)`)
	require.NotContains(t, code, "s.encoder(r.Context(), w).Encode(response)")
}

func TestJSONRPCSSEServiceStreamSendOmitsResponseBranchWithoutID(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEStringDSL)
	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "sse.go", "jsonrpc-server-sse-stream-impl")

	require.NotContains(t, code, "var isResponse bool")
	require.NotContains(t, code, "jsonrpc.MakeSuccessResponse")
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-sse-stream-impl-string.golden"), code)
}

func TestJSONRPCMixedServerInitUsesServeHTTP(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcMixedInitializeAndEventsStreamDSL)
	serverInitCode := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-init")

	require.Contains(t, serverInitCode, `s.Handler = http.NewCrossOriginProtection().Handler(http.HandlerFunc(s.serveHTTP))`)
	require.NotContains(t, serverInitCode, `s.Handler = http.HandlerFunc(s.handleSSE)`)
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-server-init-mixed.golden"), serverInitCode)
}

func TestJSONRPCMixedServerOmitsStandaloneSSEPath(t *testing.T) {
	files := ServerFiles("", CreateJSONRPCServices(RunJSONRPCDSL(t, jsonrpcMixedMultipleSSEMethodsDSL)))
	for _, file := range files {
		require.NotEqual(t, "sse.go", filepath.Base(file.Path))
		for _, section := range file.AllSections() {
			require.NotEqual(t, "jsonrpc-sse-server-handler", section.SectionName())
		}
	}

	code := allRenderedSections(t, files)
	require.NotContains(t, code, "func (s *JSONRPCMixedMultipleSSEMethodsServiceServer) handleSSE")
	require.Equal(t, 1, strings.Count(code, `r.Method == http.MethodGet && req.Method == "events/stream"`))
}

func TestJSONRPCSSEOnlyServerInitUsesOriginProtection(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEEventsStreamDSL)
	serverInitCode := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-init")

	require.Contains(t, serverInitCode, `s.Handler = http.NewCrossOriginProtection().Handler(http.HandlerFunc(s.handleSSE))`)
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-server-init-sse-only.golden"), serverInitCode)
}

func TestJSONRPCServerMountRoutes(t *testing.T) {
	const (
		postMount = `mux.Handle("POST", "/rpc", h.ServeHTTP)`
		getMount  = `mux.Handle("GET", "/rpc", h.ServeHTTP)`
	)
	cases := []struct {
		Name          string
		DSL           func()
		ForceGETVerbs string
		Contains      []string
		CountOne      []string
	}{
		{
			Name:     "mixed includes GET",
			DSL:      jsonrpcMixedInitializeAndEventsStreamDSL,
			Contains: []string{postMount, getMount},
			CountOne: []string{postMount, getMount},
		},
		{
			Name:          "mixed dedupes route verb GET",
			DSL:           jsonrpcMixedInitializeAndEventsStreamDSL,
			ForceGETVerbs: "JSONRPCMixedInitializeEventsStreamService",
			CountOne:      []string{getMount},
		},
		{
			Name:     "sse-only dedupes routes and includes GET listener",
			DSL:      testdata.JSONRPCSSEEventsStreamDSL,
			CountOne: []string{postMount, getMount},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunJSONRPCDSL(t, c.DSL)
			services := CreateJSONRPCServices(root)
			if c.ForceGETVerbs != "" {
				service := services.Get(c.ForceGETVerbs)
				require.NotNil(t, service)
				for _, endpoint := range service.Endpoints {
					for _, route := range endpoint.Routes {
						route.Verb = "GET"
					}
				}
			}
			mountCode := fileSectionCode(t, ServerFiles("", services), "server.go", "jsonrpc-server-mount")
			for _, want := range c.Contains {
				require.Contains(t, mountCode, want)
			}
			for _, want := range c.CountOne {
				require.Equal(t, 1, strings.Count(mountCode, want))
			}
		})
	}
}

func TestJSONRPCSSEClientRequestsAcceptEventStream(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEEventsStreamDSL)
	clientCode := fileSectionCode(t, ClientFiles("", CreateJSONRPCServices(root)), "client.go", "jsonrpc-client-endpoint-init")

	require.Contains(t, clientCode, `req.Header.Set("Accept", "text/event-stream")`)
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-client-endpoint-init-events-stream.golden"), clientCode)
}

func TestJSONRPCSSENotificationErrorsDoNotEmitFrames(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEObjectDSL)
	sseHandlerCode := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-sse-server-handler")

	require.Contains(t, sseHandlerCode, `w.WriteHeader(http.StatusNoContent)`)
	require.NotContains(t, sseHandlerCode, `req.ID == ""`)
	require.NotContains(t, sseHandlerCode, `req.ID != ""`)
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-sse-server-handler-object.golden"), sseHandlerCode)
}

func TestJSONRPCSSEStreamPreservesRequestIDAndSkipsNotificationCloseResponse(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEObjectDSL)
	endpointStreamCode := fileSectionCode(t, SSEServerFiles("", CreateJSONRPCServices(root)), "stream.go", "jsonrpc-sse-server-stream")

	require.Contains(t, endpointStreamCode, `if !s.requestHasID {`)
	require.Contains(t, endpointStreamCode, `var id any = s.requestID`)
	require.NotContains(t, endpointStreamCode, `id = result.ID`)
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-sse-object.golden"), endpointStreamCode)
}

func TestJSONRPCSSEHandlerPassesOriginalRequestIDToErrors(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEObjectDSL)
	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-handler-init")

	require.Contains(t, code, `strm.sendError(ctx, req.ID,`)
	require.NotContains(t, code, `req.ID != ""`)
	require.NotContains(t, code, `strm.SendError(ctx, jsonrpc.IDToString(req.ID), err)`)
	require.NotContains(t, code, `strm.sendError(ctx, jsonrpc.IDToString(req.ID),`)
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-sse-handler-init-object.golden"), code)
}

var jsonrpcMixedInitializeAndEventsStreamDSL = func() {
	dsl.API("jsonrpc-mixed-initialize-events-stream-test", func() {
		dsl.JSONRPC(func() {})
	})
	dsl.Service("JSONRPCMixedInitializeEventsStreamService", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("initialize", func() {
			dsl.Payload(func() {
				dsl.ID("id", dsl.String, "Request ID")
			})
			dsl.Result(func() {
				dsl.ID("id", dsl.String, "Request ID")
				dsl.Attribute("protocol_version", dsl.String)
			})
			dsl.JSONRPC(func() {})
		})
		dsl.Method("events/stream", func() {
			dsl.Payload(func() {
				dsl.ID("id", dsl.String, "Request ID")
			})
			dsl.StreamingResult(func() {
				dsl.Attribute("value", dsl.String)
			})
			dsl.JSONRPC(func() {
				dsl.ServerSentEvents()
			})
		})
	})
}

var jsonrpcMixedMultipleSSEMethodsDSL = func() {
	dsl.API("jsonrpc-mixed-multiple-sse-test", func() {
		dsl.JSONRPC(func() {})
	})
	dsl.Service("JSONRPCMixedMultipleSSEMethodsService", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("initialize", func() {
			dsl.Payload(func() {
				dsl.ID("id", dsl.String, "Request ID")
			})
			dsl.Result(func() {
				dsl.ID("id", dsl.String, "Request ID")
			})
			dsl.JSONRPC(func() {})
		})
		dsl.Method("tools/call", func() {
			dsl.Payload(func() {
				dsl.ID("id", dsl.String, "Request ID")
			})
			dsl.StreamingResult(func() {
				dsl.Attribute("value", dsl.String)
			})
			dsl.JSONRPC(func() {
				dsl.ServerSentEvents()
			})
		})
		dsl.Method("events/stream", func() {
			dsl.Payload(func() {
				dsl.ID("id", dsl.String, "Request ID")
			})
			dsl.StreamingResult(func() {
				dsl.Attribute("value", dsl.String)
			})
			dsl.JSONRPC(func() {
				dsl.ServerSentEvents()
			})
		})
	})
}
