package testingx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServicePackageImportNames(t *testing.T) {
	const genpkg = "example.com/m/gen"
	for _, tc := range []struct {
		name  string
		files map[string]string
		want  []string
	}{
		{
			name: "fixed alias shares service import file",
			files: map[string]string{"gen/http/svc/server/a.go": `package server
import svc "example.com/m/gen/svc"
import loomdiagnostics "github.com/CaliLuke/loom/observability/diagnostics"
import "net/http"
`},
			want: []string{"http", "loomdiagnostics"},
		},
		{
			name: "standalone and same-package separate files do not share imports",
			files: map[string]string{
				"gen/http/svc/server/a.go": `package server
import svc "example.com/m/gen/svc"
import "fmt"
`,
				"gen/http/svc/server/b.go": `package server
import loomdiagnostics "github.com/CaliLuke/loom/observability/diagnostics"
`,
				"gen/http/svc/diagnostics.go": `package diagnostics
import loomdiagnostics "github.com/CaliLuke/loom/observability/diagnostics"
`,
			},
			want: []string{"fmt"},
		},
		{
			name: "views import and module imports",
			files: map[string]string{"gen/http/svc/server/a.go": `package server
import views "example.com/m/gen/svc/views"
import implementation "example.com/m"
import arbitrary "example.com/m/custom"
import dynamic "example.com/m/gen/http/svc"
import alias "example.com/more/external"
import "context"
import _ "net/http/pprof"
import . "fmt"
`},
			want: []string{"alias", "context"},
		},
		{
			name: "transport imports alone are not service imports",
			files: map[string]string{"cmd/api/a.go": `package main
import dynamic "example.com/m/gen/http/svc/server"
import "fmt"
`},
			want: []string{},
		},
		{
			name: "service and views package declarations are excluded",
			files: map[string]string{
				"gen/svc/a.go": `package svc
import other "example.com/m/gen/other"
import "fmt"
`,
				"gen/svc/views/a.go": `package views
import other "example.com/m/gen/other/views"
import "context"
`,
			},
			want: []string{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for filename, source := range tc.files {
				filename = filepath.Join(dir, filename)
				require.NoError(t, os.MkdirAll(filepath.Dir(filename), 0o755))
				require.NoError(t, os.WriteFile(filename, []byte(source), 0o600))
			}
			require.Equal(t, tc.want, ServicePackageImportNames(t, dir, genpkg))
		})
	}
}
