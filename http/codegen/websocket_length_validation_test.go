package codegen

import (
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	cg "github.com/CaliLuke/loom/codegen"

	. "github.com/CaliLuke/loom/dsl"
)

// webSocketLengthValidationCase is a service with one WebSocket method whose
// streaming messages have string length validations.
type webSocketLengthValidationCase struct {
	service string
	method  func()
}

// TestWebSocketFilesImportUTF8 renders the WebSocket server and client files
// of every case and checks that each file that counts runes imports
// unicode/utf8. Each case counts runes in at least one of the files.
func TestWebSocketFilesImportUTF8(t *testing.T) {
	root := RunHTTPDSL(t, webSocketLengthValidationDSL)
	services := CreateHTTPServices(root)
	for _, c := range webSocketLengthValidationCases {
		t.Run(c.service, func(t *testing.T) {
			svc := root.API.HTTP.Service(c.service)
			require.NotNil(t, svc)
			counted := false
			files := map[string]*cg.File{
				"server": websocketServerFile("example.com/gen", svc, services),
				"client": websocketClientFile("example.com/gen", svc, services),
			}
			for side, file := range files {
				src := renderWebSocketFile(t, file)
				if !strings.Contains(src, "utf8.RuneCountInString") {
					continue
				}
				counted = true
				parsed, err := parser.ParseFile(token.NewFileSet(), "websocket.go", src, parser.ImportsOnly)
				require.NoError(t, err)
				imported := false
				for _, spec := range parsed.Imports {
					path, err := strconv.Unquote(spec.Path.Value)
					require.NoError(t, err)
					imported = imported || path == "unicode/utf8"
				}
				if !imported {
					t.Errorf("%s websocket.go counts runes without importing unicode/utf8", side)
				}
			}
			require.True(t, counted, "no websocket.go of the case counts runes")
		})
	}
}

// TestWebSocketLengthValidationModuleCompiles generates every WebSocket
// length validation case in one module, then compiles and vets it.
func TestWebSocketLengthValidationModuleCompiles(t *testing.T) {
	root := RunHTTPDSL(t, webSocketLengthValidationDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/websocketlength", root)
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
}

var webSocketLengthValidationCases = []webSocketLengthValidationCase{
	{service: "string_messages", method: func() {
		StreamingPayload(String, func() {
			MinLength(1)
		})
		StreamingResult(String, func() {
			MinLength(1)
		})
	}},
	{service: "string_array_messages", method: func() {
		StreamingPayload(ArrayOf(String, func() {
			MinLength(1)
		}))
		StreamingResult(ArrayOf(String, func() {
			MaxLength(8)
		}))
	}},
	{service: "string_map_messages", method: func() {
		StreamingPayload(MapOf(String, String, func() {
			Key(func() {
				MinLength(1)
			})
		}))
		StreamingResult(MapOf(String, String, func() {
			Elem(func() {
				MinLength(1)
			})
		}))
	}},
	{service: "pattern_result", method: func() {
		StreamingPayload(String, func() {
			Pattern("^a")
		})
		StreamingResult(String, func() {
			Pattern("^b")
			MaxLength(9)
		})
	}},
	{service: "unary_result", method: func() {
		StreamingPayload(String, func() {
			MinLength(1)
		})
		Result(String, func() {
			MinLength(2)
		})
	}},
}

func webSocketLengthValidationDSL() {
	for _, c := range webSocketLengthValidationCases {
		Service(c.service, func() {
			Method("talk", func() {
				c.method()
				HTTP(func() {
					GET("/" + c.service)
				})
			})
		})
	}
}

// renderWebSocketFile renders file in a temporary directory and returns its
// source.
func renderWebSocketFile(t *testing.T, file *cg.File) string {
	t.Helper()
	path, err := file.Render(t.TempDir())
	require.NoError(t, err)
	src, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(src)
}
