package codegen

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/codegen/testutil"
	"goa.design/goa/v3/jsonrpc/codegen/testdata"
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
	require.Contains(t, sseHandlerCode, `if req.ID == nil || req.ID == "" {`)
	require.Contains(t, sseHandlerCode, `w.WriteHeader(http.StatusNoContent)`)
}
