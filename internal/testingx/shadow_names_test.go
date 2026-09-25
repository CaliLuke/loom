package testingx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServicePackageShadowNames(t *testing.T) {
	const genpkg = "example.com/m/gen"
	cases := []struct {
		Name  string
		Files map[string]string
		Want  []string
	}{
		{
			Name: "names in scope at a service selector",
			Files: map[string]string{"gen/http/svc/server/a.go": `package server
import svc "example.com/m/gen/svc"
func f(r int) {
	req := 1
	if ok := true; ok {
		_ = svc.X
	}
	late := 2
	_, _, _ = r, req, late
}`},
			Want: []string{"f", "ok", "r", "req"},
		},
		{
			Name: "right-hand side and range expression come before the declaration",
			Files: map[string]string{"gen/http/svc/server/a.go": `package server
import svc "example.com/m/gen/svc"
func f() {
	for k := range svc.Y {
		_ = k
	}
	data := svc.X
	_ = data
}`},
			Want: []string{"f"},
		},
		{
			Name: "var type and value come before the declaration",
			Files: map[string]string{"gen/http/svc/server/a.go": `package server
import svc "example.com/m/gen/svc"
func f() {
	var result *svc.T = nil
	_ = result
}`},
			Want: []string{"f"},
		},
		{
			Name: "views package and function literal parameters",
			Files: map[string]string{"gen/http/svc/server/a.go": `package server
import "example.com/m/gen/svc/views"
var g = func(conn int) { _ = views.X }`},
			Want: []string{"conn", "g"},
		},
		{
			Name: "selector operand in a file without a service import",
			Files: map[string]string{
				"gen/http/svc/client/a.go": `package client
import svc "example.com/m/gen/svc"
func f() { _ = svc.X }`,
				"gen/http/svc/client/b.go": `package client
type conn struct{ ws int }
func (c *conn) g(u conn) {
	ws := c
	_, _ = ws.ws, u
}`,
			},
			Want: []string{"conn", "f", "ws"},
		},
		{
			Name: "struct fields, upper case and marked names are omitted",
			Files: map[string]string{"gen/http/svc/server/a.go": `package server
import svc "example.com/m/gen/svc"
type T struct{ field int }
func F(zzName, Upper int) { _ = svc.X }`},
			Want: []string{},
		},
		{
			Name: "service packages and packages without service imports are skipped",
			Files: map[string]string{
				"gen/svc/a.go":             "package svc\nfunc local() {}",
				"gen/svc/views/a.go":       "package views\nfunc view() {}",
				"gen/http/svc/paths/a.go":  "package paths\nfunc path() {}",
				"gen/http/other/server.go": "package server\nimport \"example.com/m/gen/other/deep\"\nfunc g() { _ = deep.X }",
			},
			Want: []string{},
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range c.Files {
				path := filepath.Join(dir, filepath.FromSlash(name))
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
			}
			assert.Equal(t, c.Want, ServicePackageShadowNames(t, dir, genpkg, "zz"))
		})
	}
}
