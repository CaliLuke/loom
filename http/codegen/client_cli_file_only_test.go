package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestClientCLIFileOnlyServices(t *testing.T) {
	for _, tc := range []struct {
		name      string
		design    func()
		endpoints bool
	}{
		{"files", testdata.FileServiceDSL, false},
		{"wildcard", testdata.FileServiceWildcardDSL, false},
		{"multiple files", testdata.ServerMultipleFilesDSL, false},
		{"redirected files", testdata.ServerFileServerWithRedirectDSL, false},
		{"files and endpoints", testdata.ServerMixedDSL, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := ClientCLIFiles("gen", CreateHTTPServices(RunHTTPDSL(t, tc.design)))
			parser := findFileWithSection(t, files, "parse-endpoint")
			code := codegen.SectionCode(t, parser.Section("parse-endpoint")[0])
			require.Contains(t, code, "func ParseEndpoint(")
			if tc.endpoints {
				require.Contains(t, code, "switch epn {")
			} else {
				require.NotContains(t, code, "epn")
			}
		})
	}
}
