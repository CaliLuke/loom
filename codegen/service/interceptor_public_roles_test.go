package service

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testdata"
)

func TestInterceptorPublicRoles(t *testing.T) {
	for _, tc := range []struct {
		name             string
		streaming, cross bool
		side             string
		names            []string
	}{
		{"unary", false, false, "both", []string{"AuditInfo", "AuditPayload", "AuditResult"}},
		{"stream", true, false, "both", []string{"AuditInfo", "AuditPayload", "AuditStreamingPayload", "AuditStreamingResult"}},
		{"cross", true, true, "both", []string{"AuditStreamingPayload", "AuditStreamingPayload2"}},
		{"suffix", false, false, "both", []string{"AuditPayload", "AuditPayload2"}},
		{"disabled", false, false, "both", []string{"AuditStreamingPayload", "AuditStreamingResult"}},
		{"server", false, false, "server", []string{"AuditInfo"}},
		{"client", false, false, "client", []string{"AuditInfo"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				testdata.InterceptorPublicRolesDSL(tc.streaming, tc.cross, tc.side, tc.names)
			})
			services := NewServicesData(root)
			data := services.Get("roles")
			declared := make(map[string]bool)
			files := Files("example.com/roles/gen", root.Services[0], services, make(map[string][]string))
			for _, file := range files {
				var source bytes.Buffer
				for _, section := range file.AllSections() {
					require.NoError(t, section.Write(&source))
				}
				parsed, err := parser.ParseFile(token.NewFileSet(), file.Path, source.Bytes(), 0)
				require.NoError(t, err)
				for _, declaration := range parsed.Decls {
					if gen, ok := declaration.(*ast.GenDecl); ok && gen.Tok == token.TYPE {
						for _, spec := range gen.Specs {
							name := spec.(*ast.TypeSpec).Name.Name
							key := filepath.Dir(file.Path) + "/" + name
							require.False(t, declared[key], "duplicate declaration %s", key)
							declared[key] = true
						}
					}
				}
			}
			if tc.side == "both" {
				require.Equal(t, data.ServerInterceptors[0].InfoName, data.ClientInterceptors[0].InfoName)
				require.Equal(t, data.ServerInterceptors[0].PayloadInterface, data.ClientInterceptors[0].PayloadInterface)
			}
			if tc.cross {
				first, second := data.ServerInterceptors[0], data.ServerInterceptors[1]
				require.Equal(t, "AuditStreamingPayload", first.StreamingPayloadInterface)
				require.Equal(t, "AuditStreamingPayload2", second.PayloadInterface)
			}
			if tc.name == "disabled" {
				require.Empty(t, data.ServerInterceptors[0].StreamingPayloadInterface)
				require.Empty(t, data.ServerInterceptors[0].StreamingResultInterface)
				for _, typ := range data.userTypes {
					require.Equal(t, typ.Name, typ.VarName, "absent roles must not reserve nominal names")
				}
			}
		})
	}
}
