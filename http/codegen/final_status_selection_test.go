package codegen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestFinalStatusSelectionRendering(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ResponseStatusSelectionDSL)
	services := CreateHTTPServices(root)
	for _, side := range []string{"server", "client"} {
		t.Run(side, func(t *testing.T) {
			var files []*codegen.File
			require.NotPanics(t, func() {
				if side == "server" {
					files = ServerFiles("example.com/status/gen", services)
				} else {
					files = ClientFiles("example.com/status/gen", services)
				}
			})
			var output strings.Builder
			require.NotPanics(t, func() {
				for _, file := range files {
					for _, name := range []string{"response-encoder", "error-encoder", "response-decoder"} {
						for _, section := range file.Section(name) {
							output.WriteString(codegen.SectionCode(t, section))
						}
					}
				}
			})
			require.Contains(t, output.String(), "SelectFinal")
			require.Contains(t, output.String(), "StatusSelector")
			if side == "server" {
				require.Contains(t, output.String(), "StatusViolationError")
			}
		})
	}
}
