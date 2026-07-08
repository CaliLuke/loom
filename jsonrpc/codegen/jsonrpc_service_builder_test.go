package codegen

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func TestJSONRPCCodegenWithSynthesizedService(t *testing.T) {
	serviceExpr := &expr.ServiceExpr{
		Name: "calc",
		Methods: []*expr.MethodExpr{{
			Name:    "add",
			Payload: &expr.AttributeExpr{Type: expr.String},
			Result:  &expr.AttributeExpr{Type: expr.String},
		}},
	}
	httpService := expr.NewJSONRPCHTTPService(serviceExpr, "/rpc")
	root := &expr.RootExpr{
		Services: []*expr.ServiceExpr{serviceExpr},
		API: &expr.APIExpr{
			Name: "rpc",
			ExampleGenerator: &expr.ExampleGenerator{
				Randomizer: expr.NewFakerRandomizer("rpc"),
			},
			HTTP: &expr.HTTPExpr{},
			JSONRPC: &expr.JSONRPCExpr{
				HTTPExpr: expr.HTTPExpr{Services: []*expr.HTTPServiceExpr{httpService}},
			},
			GRPC: &expr.GRPCExpr{},
			Servers: []*expr.ServerExpr{{
				Name:     "calc",
				Services: []string{"calc"},
			}},
		},
	}
	httpService.Root = &root.API.JSONRPC.HTTPExpr

	err := expr.PrepareValidateFinalize(root)
	require.NoError(t, err)

	services := service.NewServicesData(root)
	httpServices := httpcodegen.NewServicesData(services, &root.API.JSONRPC.HTTPExpr)
	files := ServerFiles("loom.design/example", httpServices)

	require.NotEmpty(t, files)
	require.NotNil(t, httpServices.Get("calc"))
}

func TestJSONRPCTopLevelSections(t *testing.T) {
	t.Run("plain HTTP service emits converted top-level sections", func(t *testing.T) {
		root := RunJSONRPCDSL(t, func() {
			dsl.API("jsonrpc-top-level-http-test", func() {
				dsl.JSONRPC(func() {})
			})
			dsl.Service("calc", func() {
				dsl.JSONRPC(func() {
					dsl.POST("/rpc")
				})
				dsl.Method("add", func() {
					dsl.Payload(func() {
						dsl.ID("id", dsl.String)
						dsl.Attribute("value", dsl.Int)
					})
					dsl.Result(func() {
						dsl.ID("id", dsl.String)
						dsl.Attribute("sum", dsl.Int)
					})
					dsl.JSONRPC(func() {})
				})
			})
		})

		serverCode := topLevelSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "jsonrpc-server-struct", "jsonrpc-server-init")
		clientCode := topLevelSectionCode(t, ClientFiles("", CreateJSONRPCServices(root)), "jsonrpc-client-struct", "jsonrpc-client-init")

		require.Contains(t, serverCode, "type Server struct")
		require.Contains(t, serverCode, "Methods []string")
		require.Contains(t, serverCode, "Handler = http.NewCrossOriginProtection().Handler(http.HandlerFunc(s.ServeHTTP))")
		require.NotContains(t, serverCode, "StreamHandler func")
		require.Contains(t, clientCode, "type Client struct")
		require.Contains(t, clientCode, "var bufferPool = sync.Pool")
		require.NotContains(t, clientCode, "streamConfig *jsonrpc.StreamConfig")
	})

	t.Run("mixed SSE service keeps top-level mixed transport wiring", func(t *testing.T) {
		root := RunJSONRPCDSL(t, jsonrpcMixedInitializeAndEventsStreamDSL)
		services := CreateJSONRPCServices(root)
		serverCode := topLevelSectionCode(t, ServerFiles("", services), "jsonrpc-server-init", "jsonrpc-mixed-server-handler")
		clientCode := topLevelSectionCode(t, ClientFiles("", services), "jsonrpc-client-struct")

		require.Contains(t, serverCode, "Mixed HTTP/SSE services negotiate transports in ServeHTTP")
		require.Contains(t, serverCode, `req := &jsonrpc.RawRequest{JSONRPC: "2.0", Method: "events/stream"}`)
		require.NotContains(t, serverCode, `"events-stream"`)
		require.Contains(t, serverCode, `case "events/stream":`)
		require.Contains(t, clientCode, "EventsStreamDoer loomhttp.Doer")
		require.Contains(t, clientCode, "RestoreResponseBody bool")
	})

	t.Run("websocket service keeps stream-specific top-level state", func(t *testing.T) {
		root := RunJSONRPCDSL(t, func() {
			dsl.API("jsonrpc-top-level-websocket-test", func() {
				dsl.JSONRPC(func() {})
			})
			dsl.Service("stream", func() {
				dsl.JSONRPC(func() {})
				dsl.Method("echo", func() {
					dsl.StreamingPayload(func() {
						dsl.ID("id", dsl.String)
						dsl.Attribute("msg", dsl.String)
					})
					dsl.StreamingResult(func() {
						dsl.ID("id", dsl.String)
						dsl.Attribute("echo", dsl.String)
					})
					dsl.JSONRPC(func() {})
				})
			})
		})

		services := CreateJSONRPCServices(root)
		serverCode := topLevelSectionCode(t, ServerFiles("", services), "jsonrpc-server-struct", "jsonrpc-server-init")
		clientCode := topLevelSectionCode(t, ClientFiles("", services), "jsonrpc-client-struct", "jsonrpc-client-init")

		require.Contains(t, serverCode, "StreamHandler func(context.Context, stream.Stream) error")
		require.Contains(t, serverCode, "upgrader loomhttp.Upgrader")
		require.Contains(t, clientCode, "dialer loomhttp.Dialer")
		require.Contains(t, clientCode, "streamConfig *jsonrpc.StreamConfig")
		require.Contains(t, clientCode, "streamConfig := jsonrpc.NewStreamConfig(streamOpts...)")
	})

	t.Run("websocket stream send helpers keep doc comments above signatures", func(t *testing.T) {
		root := RunJSONRPCDSL(t, func() {
			dsl.API("jsonrpc-top-level-websocket-send-doc-test", func() {
				dsl.JSONRPC(func() {})
			})
			dsl.Service("stream", func() {
				dsl.JSONRPC(func() {})
				dsl.Method("echo", func() {
					dsl.StreamingPayload(func() {
						dsl.ID("id", dsl.String)
						dsl.Attribute("msg", dsl.String)
					})
					dsl.StreamingResult(func() {
						dsl.ID("id", dsl.String)
						dsl.Attribute("echo", dsl.String)
					})
					dsl.JSONRPC(func() {})
				})
			})
		})

		services := CreateJSONRPCServices(root)
		code := fileSectionCode(t, ServerFiles("", services), "websocket.go", "jsonrpc-server-websocket-send")

		require.Contains(t, code, "// SendEchoNotification sends a JSON-RPC notification for the echo method.")
		require.Contains(t, code, "func (s *streamStream) SendEchoNotification(")
		require.NotContains(t, code, "SendEchoNotification //")
		require.Contains(t, code, "// SendEchoResponse sends a JSON-RPC response for the echo method.")
		require.Contains(t, code, "func (s *streamStream) SendEchoResponse(")
		require.NotContains(t, code, "SendEchoResponse //")
		require.True(t, strings.Index(code, "// SendEchoNotification sends a JSON-RPC notification for the echo method.") < strings.Index(code, "func (s *streamStream) SendEchoNotification("))
		require.True(t, strings.Index(code, "// SendEchoResponse sends a JSON-RPC response for the echo method.") < strings.Index(code, "func (s *streamStream) SendEchoResponse("))
	})

	t.Run("websocket server emits service error classifier", func(t *testing.T) {
		root := RunJSONRPCDSL(t, func() {
			dsl.API("jsonrpc-websocket-error-classifier-test", func() {
				dsl.JSONRPC(func() {})
			})
			dsl.Service("stream", func() {
				dsl.JSONRPC(func() {})
				dsl.Method("echo", func() {
					dsl.StreamingPayload(dsl.String)
					dsl.StreamingResult(dsl.String)
					dsl.JSONRPC(func() {})
				})
			})
		})

		services := CreateJSONRPCServices(root)
		files := ServerFiles("", services)
		sendCode := fileSectionCode(t, files, "websocket.go", "jsonrpc-server-websocket-send")
		classifierCode := fileSectionCode(t, files, "websocket.go", "jsonrpc-server-websocket-service-error-classifier")

		require.Contains(t, sendCode, "code = jsonrpcErrorCodeForServiceError(serviceError)")
		require.Contains(t, classifierCode, "func jsonrpcErrorCodeForServiceError(err *loom.ServiceError) jsonrpc.Code")
	})
}

func topLevelSectionCode(t *testing.T, files []*codegen.File, sectionNames ...string) string {
	t.Helper()

	sections := make(map[string]codegen.Section, len(sectionNames))
	for _, file := range files {
		if filepath.Base(file.Path) != "server.go" && filepath.Base(file.Path) != "client.go" {
			continue
		}
		for _, section := range file.AllSections() {
			for _, name := range sectionNames {
				if section.SectionName() == name {
					sections[name] = section
				}
			}
		}
	}

	var rendered string
	for _, name := range sectionNames {
		section, ok := sections[name]
		require.Truef(t, ok, "section %s not found", name)
		rendered += codegen.SectionCode(t, section)
	}
	return rendered
}

func fileSectionCode(t *testing.T, files []*codegen.File, baseName string, sectionName string) string {
	t.Helper()

	for _, file := range files {
		if filepath.Base(file.Path) != baseName {
			continue
		}
		for _, section := range file.AllSections() {
			if section.SectionName() == sectionName {
				return codegen.SectionCode(t, section)
			}
		}
	}

	t.Fatalf("section %q not found in %q", sectionName, baseName)
	return ""
}
