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
			require.Contains(t, code, `sendSSEEvent("message", message)`)
			require.NotContains(t, code, `sendSSEEvent("response", message)`)
			require.Equal(t, 2, strings.Count(code, `sendSSEEvent("message", message)`))
			require.Contains(t, code, `sendSSEEvent("message", response)`)
			golden := filepath.Join("testdata", "golden", "jsonrpc-sse-"+c.Name+".golden")
			testutil.CompareOrUpdateGolden(t, code, golden)

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
			require.Contains(t, clientCode, `case "notification":`)
			require.Contains(t, clientCode, `case "response":`)
			require.Contains(t, clientCode, `case "error":`)
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
}

func TestJSONRPCSSEEndpointStreamsRemainLazyByDefault(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEObjectDSL)
	services := CreateJSONRPCServices(root)

	endpointStreamCode := fileSectionCode(t, SSEServerFiles("", services), "stream.go", "jsonrpc-sse-server-stream")
	require.Contains(t, endpointStreamCode, `func (s *StreamServerStream) open() error {`)
	require.Contains(t, endpointStreamCode, `return s.sendSSEEvent("message", response)`)

	handlerInitCode := fileSectionCode(t, ServerFiles("", services), "server.go", "jsonrpc-server-handler-init")
	require.Contains(t, handlerInitCode, "StreamServerStream")
	require.Contains(t, handlerInitCode, `if r.Method == http.MethodGet && req.Method == "events/stream" {`)
	require.NotContains(t, handlerInitCode, `if err := strm.open(); err != nil {
			return err
		}
		decodeParams :=`)
}

func TestJSONRPCSSEEventsStreamGETOpensBeforeFirstFrame(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEEventsStreamDSL)
	handlerInitCode := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-handler-init")

	require.Contains(t, handlerInitCode, `if r.Method == http.MethodGet && req.Method == "events/stream" {`)
	require.Contains(t, handlerInitCode, `if err := strm.open(); err != nil {`)
}

func TestJSONRPCMixedServerHandler(t *testing.T) {
	cases := []struct {
		Name        string
		DSL         func()
		Contains    []string
		NotContains []string
	}{
		{
			Name: "routes by method",
			DSL:  jsonrpcMixedInitializeAndEventsStreamDSL,
			Contains: []string{
				`if !strings.Contains(accept, "text/event-stream") {`,
				`case http.MethodGet:`,
				`req := &jsonrpc.RawRequest{JSONRPC: "2.0", Method: "events/stream"}`,
				`var req jsonrpc.RawRequest`,
				`switch req.Method {`,
				`case "events/stream":`,
				`if err := s.EventsStream(r.Context(), r, req, w); err != nil {`,
				`if err := s.EventsStream(r.Context(), r, &req, w); err != nil {`,
				`s.handleHTTP(w, r)`,
			},
			NotContains: []string{
				`"events-stream"`,
				"if strings.Contains(accept, \"text/event-stream\") {\n\t\ts.handleSSE(w, r)\n\t\treturn\n\t}",
			},
		},
		{
			Name: "avoids full body read for negotiation",
			DSL:  jsonrpcMixedInitializeAndEventsStreamDSL,
			Contains: []string{
				`reader := bufio.NewReader(r.Body)`,
				`const maxNegotiationWhitespace = 4096`,
				`reader.Peek(1)`,
				`reader.Discard(1)`,
				`sniffed < maxNegotiationWhitespace`,
				`first == byte(0x5b)`,
				`sniffed >= maxNegotiationWhitespace`,
			},
			NotContains: []string{
				`io.ReadAll(r.Body)`,
			},
		},
		{
			Name: "groups multiple sse methods",
			DSL:  jsonrpcMixedMultipleSSEMethodsDSL,
			Contains: []string{
				`case "tools/call":`,
				`if err := s.ToolsCall(r.Context(), r, &req, w); err != nil {`,
				`case "events/stream":`,
				`if err := s.EventsStream(r.Context(), r, &req, w); err != nil {`,
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
		})
	}
}

func TestJSONRPCSSEOnlyHandlerOmitsEmptyNotificationSwitch(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEObjectDSL)
	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-sse-server-handler")

	require.NotContains(t, code, "switch req.Method {\n\t}")
}

func TestJSONRPCSSEServiceStreamSendOmitsResponseBranchWithoutID(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEStringDSL)
	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "sse.go", "jsonrpc-server-sse-stream-impl")

	require.NotContains(t, code, "var isResponse bool")
	require.NotContains(t, code, "jsonrpc.MakeSuccessResponse")
	require.Contains(t, code, `"method":  "JSONRPCSSEStringService/stream.event",`)
}

func TestJSONRPCMixedServerInitUsesServeHTTP(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcMixedInitializeAndEventsStreamDSL)
	serverInitCode := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-init")

	require.Contains(t, serverInitCode, `Mixed HTTP/SSE services negotiate transports in ServeHTTP`)
	require.Contains(t, serverInitCode, `s.Handler = http.NewCrossOriginProtection().Handler(http.HandlerFunc(s.ServeHTTP))`)
	require.NotContains(t, serverInitCode, `s.Handler = http.HandlerFunc(s.handleSSE)`)
}

func TestJSONRPCSSEOnlyServerInitUsesOriginProtection(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEEventsStreamDSL)
	serverInitCode := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-init")

	require.Contains(t, serverInitCode, `s.Handler = http.NewCrossOriginProtection().Handler(http.HandlerFunc(s.handleSSE))`)
}

func TestJSONRPCServerMountRoutes(t *testing.T) {
	const (
		postMount = `mux.Handle("POST", "/rpc", h.Handler.ServeHTTP)`
		getMount  = `mux.Handle("GET", "/rpc", h.Handler.ServeHTTP)`
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

	require.Contains(t, clientCode, "EventsStream")
	require.Contains(t, clientCode, `req.Header.Set("Accept", "text/event-stream")`)
}

func TestJSONRPCSSENotificationErrorsDoNotEmitFrames(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEObjectDSL)
	sseHandlerCode := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-sse-server-handler")

	require.Contains(t, sseHandlerCode, `if req.Invalid {`)
	require.Contains(t, sseHandlerCode, `if !req.HasID {`)
	require.Contains(t, sseHandlerCode, `w.WriteHeader(http.StatusNoContent)`)
	require.NotContains(t, sseHandlerCode, `req.ID == ""`)
	require.NotContains(t, sseHandlerCode, `req.ID != ""`)
}

func TestJSONRPCSSEStreamPreservesRequestIDAndSkipsNotificationCloseResponse(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEObjectDSL)
	endpointStreamCode := fileSectionCode(t, SSEServerFiles("", CreateJSONRPCServices(root)), "stream.go", "jsonrpc-sse-server-stream")

	require.Contains(t, endpointStreamCode, `requestHasID bool`)
	require.Contains(t, endpointStreamCode, `if !s.requestHasID {`)
	require.Contains(t, endpointStreamCode, `return nil`)
	require.Contains(t, endpointStreamCode, `var id any = s.requestID`)
	require.NotContains(t, endpointStreamCode, `id = result.ID`)
}

func TestJSONRPCSSEHandlerPassesOriginalRequestIDToErrors(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEObjectDSL)
	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-handler-init")

	require.Contains(t, code, `requestHasID: req.HasID,`)
	require.Contains(t, code, `if req.HasID {`)
	require.Contains(t, code, `strm.sendError(ctx, req.ID,`)
	require.NotContains(t, code, `req.ID != ""`)
	require.NotContains(t, code, `strm.SendError(ctx, jsonrpc.IDToString(req.ID), err)`)
	require.NotContains(t, code, `strm.sendError(ctx, jsonrpc.IDToString(req.ID),`)
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
