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
			require.Contains(t, code, `sendSSEEvent("response", message)`)
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
	services := CreateJSONRPCServices(root)
	fs := ServerFiles("", services)
	require.NotEmpty(t, fs, "expected JSON-RPC server files to be generated")

	var serviceStreamCode string
	for _, f := range fs {
		if filepath.Base(f.Path) != "sse.go" || filepath.Base(filepath.Dir(f.Path)) != "server" {
			continue
		}
		for _, s := range f.AllSections() {
			if s.SectionName() != "jsonrpc-server-sse-stream-impl" {
				continue
			}
			serviceStreamCode = codegen.SectionCode(t, s)
			break
		}
	}

	require.NotEmpty(t, serviceStreamCode, "jsonrpc-server-sse-stream-impl section not found")
	require.Contains(t, serviceStreamCode, `eventType = "response"`)
	require.Contains(t, serviceStreamCode, `eventType = "message"`)
	require.Contains(t, serviceStreamCode, `return s.sendSSEEvent("message", response)`)
}

func TestJSONRPCSSEEndpointStreamsRemainLazyByDefault(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEObjectDSL)
	services := CreateJSONRPCServices(root)

	streamFiles := SSEServerFiles("", services)
	require.NotEmpty(t, streamFiles, "expected SSE stream files to be generated")

	var endpointStreamCode string
	for _, f := range streamFiles {
		if filepath.Base(f.Path) != "stream.go" || filepath.Base(filepath.Dir(f.Path)) != "server" {
			continue
		}
		for _, s := range f.AllSections() {
			if s.SectionName() != "jsonrpc-sse-server-stream" {
				continue
			}
			endpointStreamCode = codegen.SectionCode(t, s)
			break
		}
	}
	require.NotEmpty(t, endpointStreamCode, "jsonrpc-sse-server-stream section not found")
	require.Contains(t, endpointStreamCode, `func (s *StreamServerStream) open() error {`)
	require.Contains(t, endpointStreamCode, `return s.sendSSEEvent("message", response)`)

	serverFiles := ServerFiles("", services)
	require.NotEmpty(t, serverFiles, "expected JSON-RPC server files to be generated")

	var handlerInitCode string
	for _, f := range serverFiles {
		if filepath.Base(f.Path) != "server.go" || filepath.Base(filepath.Dir(f.Path)) != "server" {
			continue
		}
		for _, s := range f.AllSections() {
			if s.SectionName() != "jsonrpc-server-handler-init" {
				continue
			}
			code := codegen.SectionCode(t, s)
			if !strings.Contains(code, "StreamServerStream") {
				continue
			}
			handlerInitCode = code
			break
		}
	}
	require.NotEmpty(t, handlerInitCode, "jsonrpc-server-handler-init section for SSE stream not found")
	require.Contains(t, handlerInitCode, `if r.Method == http.MethodGet && req.Method == "events/stream" {`)
	require.NotContains(t, handlerInitCode, `if err := strm.open(); err != nil {
			return err
		}
		decodeParams :=`)
}

func TestJSONRPCSSEEventsStreamGETOpensBeforeFirstFrame(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEEventsStreamDSL)
	services := CreateJSONRPCServices(root)

	serverFiles := ServerFiles("", services)
	require.NotEmpty(t, serverFiles, "expected JSON-RPC server files to be generated")

	var handlerInitCode string
	for _, f := range serverFiles {
		if filepath.Base(f.Path) != "server.go" || filepath.Base(filepath.Dir(f.Path)) != "server" {
			continue
		}
		for _, s := range f.AllSections() {
			if s.SectionName() != "jsonrpc-server-handler-init" {
				continue
			}
			code := codegen.SectionCode(t, s)
			if !strings.Contains(code, `"events/stream"`) {
				continue
			}
			handlerInitCode = code
			break
		}
	}

	require.NotEmpty(t, handlerInitCode, "jsonrpc-server-handler-init section for events/stream not found")
	require.Contains(t, handlerInitCode, `if r.Method == http.MethodGet && req.Method == "events/stream" {`)
	require.Contains(t, handlerInitCode, `if err := strm.open(); err != nil {`)
}

func TestJSONRPCMixedHTTPAndSSEHandlerRoutesByMethod(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcMixedInitializeAndEventsStreamDSL)
	services := CreateJSONRPCServices(root)

	serverFiles := ServerFiles("", services)
	require.NotEmpty(t, serverFiles, "expected JSON-RPC server files to be generated")

	var mixedHandlerCode string
	for _, f := range serverFiles {
		if filepath.Base(f.Path) != "server.go" || filepath.Base(filepath.Dir(f.Path)) != "server" {
			continue
		}
		for _, s := range f.AllSections() {
			if s.SectionName() != "jsonrpc-mixed-server-handler" {
				continue
			}
			mixedHandlerCode = codegen.SectionCode(t, s)
			break
		}
	}

	require.NotEmpty(t, mixedHandlerCode, "jsonrpc-mixed-server-handler section not found")
	require.Contains(t, mixedHandlerCode, `if !strings.Contains(accept, "text/event-stream") {`)
	require.Contains(t, mixedHandlerCode, `case http.MethodGet:`)
	require.Contains(t, mixedHandlerCode, `req := &jsonrpc.RawRequest{JSONRPC: "2.0", Method: "events/stream"}`)
	require.NotContains(t, mixedHandlerCode, `"events-stream"`)
	require.Contains(t, mixedHandlerCode, `var req jsonrpc.RawRequest`)
	require.Contains(t, mixedHandlerCode, `switch req.Method {`)
	require.Contains(t, mixedHandlerCode, `case "events/stream":`)
	require.Contains(t, mixedHandlerCode, `if err := s.EventsStream(r.Context(), r, req, w); err != nil {`)
	require.Contains(t, mixedHandlerCode, `if err := s.EventsStream(r.Context(), r, &req, w); err != nil {`)
	require.Contains(t, mixedHandlerCode, `s.handleHTTP(w, r)`)
	require.NotContains(t, mixedHandlerCode, "if strings.Contains(accept, \"text/event-stream\") {\n\t\ts.handleSSE(w, r)\n\t\treturn\n\t}")
}

func TestJSONRPCMixedServerInitUsesServeHTTP(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcMixedInitializeAndEventsStreamDSL)
	services := CreateJSONRPCServices(root)

	serverFiles := ServerFiles("", services)
	require.NotEmpty(t, serverFiles, "expected JSON-RPC server files to be generated")

	var serverInitCode string
	for _, f := range serverFiles {
		if filepath.Base(f.Path) != "server.go" || filepath.Base(filepath.Dir(f.Path)) != "server" {
			continue
		}
		for _, s := range f.AllSections() {
			if s.SectionName() != "jsonrpc-server-init" {
				continue
			}
			serverInitCode = codegen.SectionCode(t, s)
			break
		}
	}

	require.NotEmpty(t, serverInitCode, "jsonrpc-server-init section not found")
	require.Contains(t, serverInitCode, `Mixed HTTP/SSE services negotiate transports in ServeHTTP`)
	require.Contains(t, serverInitCode, `s.Handler = http.HandlerFunc(s.ServeHTTP)`)
	require.NotContains(t, serverInitCode, `s.Handler = http.HandlerFunc(s.handleSSE)`)
}

func TestJSONRPCMixedServerMountIncludesGET(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcMixedInitializeAndEventsStreamDSL)
	services := CreateJSONRPCServices(root)

	serverFiles := ServerFiles("", services)
	require.NotEmpty(t, serverFiles, "expected JSON-RPC server files to be generated")

	var mountCode string
	for _, f := range serverFiles {
		if filepath.Base(f.Path) != "server.go" || filepath.Base(filepath.Dir(f.Path)) != "server" {
			continue
		}
		for _, s := range f.AllSections() {
			if s.SectionName() != "jsonrpc-server-mount" {
				continue
			}
			mountCode = codegen.SectionCode(t, s)
			break
		}
	}

	require.NotEmpty(t, mountCode, "jsonrpc-server-mount section not found")
	require.Contains(t, mountCode, `mux.Handle("POST", "/rpc", h.ServeHTTP)`)
	require.Contains(t, mountCode, `mux.Handle("GET", "/rpc", h.ServeHTTP)`)
	require.Equal(t, 1, strings.Count(mountCode, `mux.Handle("POST", "/rpc", h.ServeHTTP)`))
	require.Equal(t, 1, strings.Count(mountCode, `mux.Handle("GET", "/rpc", h.ServeHTTP)`))
}

func TestJSONRPCMixedServerMountDedupesRouteVerbGET(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcMixedInitializeAndEventsStreamDSL)
	services := CreateJSONRPCServices(root)
	service := services.Get("JSONRPCMixedInitializeEventsStreamService")
	require.NotNil(t, service)
	for _, endpoint := range service.Endpoints {
		for _, route := range endpoint.Routes {
			route.Verb = "GET"
		}
	}

	serverFiles := ServerFiles("", services)
	require.NotEmpty(t, serverFiles, "expected JSON-RPC server files to be generated")

	var mountCode string
	for _, f := range serverFiles {
		if filepath.Base(f.Path) != "server.go" || filepath.Base(filepath.Dir(f.Path)) != "server" {
			continue
		}
		for _, s := range f.AllSections() {
			if s.SectionName() != "jsonrpc-server-mount" {
				continue
			}
			mountCode = codegen.SectionCode(t, s)
			break
		}
	}

	require.NotEmpty(t, mountCode, "jsonrpc-server-mount section not found")
	require.Equal(t, 1, strings.Count(mountCode, `mux.Handle("GET", "/rpc", h.ServeHTTP)`))
}

func TestJSONRPCSSEOnlyServerMountDedupesRoutesAndIncludesGETListener(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEEventsStreamDSL)
	services := CreateJSONRPCServices(root)

	serverFiles := ServerFiles("", services)
	require.NotEmpty(t, serverFiles, "expected JSON-RPC server files to be generated")

	var mountCode string
	for _, f := range serverFiles {
		if filepath.Base(f.Path) != "server.go" || filepath.Base(filepath.Dir(f.Path)) != "server" {
			continue
		}
		for _, s := range f.AllSections() {
			if s.SectionName() != "jsonrpc-server-mount" {
				continue
			}
			mountCode = codegen.SectionCode(t, s)
			break
		}
	}

	require.NotEmpty(t, mountCode, "jsonrpc-server-mount section not found")
	require.Equal(t, 1, strings.Count(mountCode, `mux.Handle("POST", "/rpc", h.handleSSE)`))
	require.Equal(t, 1, strings.Count(mountCode, `mux.Handle("GET", "/rpc", h.handleSSE)`))
}

func TestJSONRPCMixedHandlerAvoidsFullBodyReadForNegotiation(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcMixedInitializeAndEventsStreamDSL)
	services := CreateJSONRPCServices(root)

	serverFiles := ServerFiles("", services)
	require.NotEmpty(t, serverFiles, "expected JSON-RPC server files to be generated")

	var mixedHandlerCode string
	for _, f := range serverFiles {
		if filepath.Base(f.Path) != "server.go" || filepath.Base(filepath.Dir(f.Path)) != "server" {
			continue
		}
		for _, s := range f.AllSections() {
			if s.SectionName() != "jsonrpc-mixed-server-handler" {
				continue
			}
			mixedHandlerCode = codegen.SectionCode(t, s)
			break
		}
	}

	require.NotEmpty(t, mixedHandlerCode, "jsonrpc-mixed-server-handler section not found")
	require.NotContains(t, mixedHandlerCode, `io.ReadAll(r.Body)`)
	require.Contains(t, mixedHandlerCode, `reader := bufio.NewReader(r.Body)`)
	require.Contains(t, mixedHandlerCode, `const maxNegotiationWhitespace = 4096`)
	require.Contains(t, mixedHandlerCode, `reader.Peek(1)`)
	require.Contains(t, mixedHandlerCode, `reader.Discard(1)`)
	require.Contains(t, mixedHandlerCode, `sniffed < maxNegotiationWhitespace`)
	require.Contains(t, mixedHandlerCode, `first == byte(0x5b)`)
	require.Contains(t, mixedHandlerCode, `sniffed >= maxNegotiationWhitespace`)
}

func TestJSONRPCSSEClientRequestsAcceptEventStream(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEEventsStreamDSL)
	services := CreateJSONRPCServices(root)

	clientFiles := ClientFiles("", services)
	require.NotEmpty(t, clientFiles, "expected JSON-RPC client files to be generated")

	var clientCode string
	for _, f := range clientFiles {
		if filepath.Base(f.Path) != "client.go" || filepath.Base(filepath.Dir(f.Path)) != "client" {
			continue
		}
		for _, s := range f.AllSections() {
			if s.SectionName() != "jsonrpc-client-endpoint-init" {
				continue
			}
			code := codegen.SectionCode(t, s)
			if !strings.Contains(code, "EventsStream") {
				continue
			}
			clientCode = code
			break
		}
	}

	require.NotEmpty(t, clientCode, "jsonrpc-client-endpoint-init section for events/stream not found")
	require.Contains(t, clientCode, `req.Header.Set("Accept", "text/event-stream")`)
}

func TestJSONRPCMixedHandlerGroupsMultipleSSEMethods(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcMixedMultipleSSEMethodsDSL)
	services := CreateJSONRPCServices(root)

	serverFiles := ServerFiles("", services)
	require.NotEmpty(t, serverFiles, "expected JSON-RPC server files to be generated")

	var mixedHandlerCode string
	for _, f := range serverFiles {
		if filepath.Base(f.Path) != "server.go" || filepath.Base(filepath.Dir(f.Path)) != "server" {
			continue
		}
		for _, s := range f.AllSections() {
			if s.SectionName() != "jsonrpc-mixed-server-handler" {
				continue
			}
			mixedHandlerCode = codegen.SectionCode(t, s)
			break
		}
	}

	require.NotEmpty(t, mixedHandlerCode, "jsonrpc-mixed-server-handler section not found")
	require.Contains(t, mixedHandlerCode, `case "tools/call":`)
	require.Contains(t, mixedHandlerCode, `if err := s.ToolsCall(r.Context(), r, &req, w); err != nil {`)
	require.Contains(t, mixedHandlerCode, `case "events/stream":`)
	require.Contains(t, mixedHandlerCode, `if err := s.EventsStream(r.Context(), r, &req, w); err != nil {`)
	require.NotContains(t, mixedHandlerCode, "case \"tools/call\":\n\t\tcase \"events/stream\":")
}

func TestJSONRPCSSENotificationErrorsDoNotEmitFrames(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEObjectDSL)
	services := CreateJSONRPCServices(root)

	serverFiles := ServerFiles("", services)
	require.NotEmpty(t, serverFiles, "expected JSON-RPC server files to be generated")

	var sseHandlerCode string
	for _, f := range serverFiles {
		if filepath.Base(f.Path) != "server.go" || filepath.Base(filepath.Dir(f.Path)) != "server" {
			continue
		}
		for _, s := range f.AllSections() {
			if s.SectionName() != "jsonrpc-sse-server-handler" {
				continue
			}
			sseHandlerCode = codegen.SectionCode(t, s)
			break
		}
	}

	require.NotEmpty(t, sseHandlerCode, "jsonrpc-sse-server-handler section not found")
	require.Contains(t, sseHandlerCode, `if req.Invalid {`)
	require.Contains(t, sseHandlerCode, `if !req.HasID {`)
	require.Contains(t, sseHandlerCode, `w.WriteHeader(http.StatusNoContent)`)
	require.NotContains(t, sseHandlerCode, `req.ID == ""`)
	require.NotContains(t, sseHandlerCode, `req.ID != ""`)
}

func TestJSONRPCSSEStreamPreservesRequestIDAndSkipsNotificationCloseResponse(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEObjectDSL)
	files := SSEServerFiles("", CreateJSONRPCServices(root))

	var endpointStreamCode string
	for _, file := range files {
		for _, section := range file.AllSections() {
			if section.SectionName() == "jsonrpc-sse-server-stream" {
				endpointStreamCode = codegen.SectionCode(t, section)
			}
		}
	}
	require.NotEmpty(t, endpointStreamCode, "jsonrpc-sse-server-stream section not found")

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
