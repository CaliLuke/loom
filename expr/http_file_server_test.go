package expr_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/expr/testdata"
)

func TestFilesCatchAllWildcardValidation(t *testing.T) {
	cases := map[string]struct {
		basePath string
		files    string
		errors   []string
	}{
		"trailing catch-all":    {files: "/ui/{*filepath}"},
		"catch-all under base":  {basePath: "/{id}", files: "/ui/{*filepath}"},
		"catch-all in base":     {basePath: "/{*a}", files: "/x/{*p}", errors: []string{`Catch-all wildcard "a" must terminate full path "/{*a}/x/{*p}"`}},
		"non trailing":          {files: "/{*p}/tail", errors: []string{`Catch-all wildcard "p" must terminate full path "/{*p}/tail"`}},
		"absolute non trailing": {basePath: "/svc", files: "//{*p}/tail", errors: []string{`Catch-all wildcard "p" must terminate full path "/{*p}/tail"`}},
		"bare star":             {files: "/ui/*filepath", errors: []string{`Path "/ui/*filepath" uses a bare "*"; use a trailing "/{*name}" catch-all wildcard instead`}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			dsl := func() {
				Service("files-wildcards", func() {
					if tc.basePath != "" {
						HTTP(func() {
							Path(tc.basePath)
						})
					}
					Files(tc.files, "dist")
				})
			}
			if len(tc.errors) == 0 {
				expr.RunDSL(t, dsl)
				return
			}
			got := expr.RunInvalidDSL(t, dsl).Error()
			for _, want := range tc.errors {
				require.Contains(t, got, want)
			}
		})
	}
}

func TestFilesDSL(t *testing.T) {
	cases := []struct {
		Name  string
		DSL   func()
		Error string
	}{
		{Name: "valid", DSL: testdata.FilesValidDSL},
		{Name: "incompatible", DSL: testdata.FilesIncompatibleDSL, Error: "invalid use of Files in API files-incompatile"},
		{Name: "too many arg error", DSL: testdata.FilesTooManyArgErrorDSL, Error: "too many arguments given to Files in API files-too-many-arg-error"},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			if c.Error == "" {
				expr.RunDSL(t, c.DSL)
			} else {
				err := expr.RunInvalidDSL(t, c.DSL)
				if !strings.HasSuffix(err.Error(), c.Error) {
					t.Errorf("got error %q, expected has suffix %q", err.Error(), c.Error)
				}
			}
		})
	}
}
