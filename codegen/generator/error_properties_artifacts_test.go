package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	d "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestErrorPropertyArtifacts(t *testing.T) {
	output := os.Getenv("LOOM_ERROR_PROPERTY_ARTIFACTS")
	if output == "" {
		t.Skip("set LOOM_ERROR_PROPERTY_ARTIFACTS for bounded output comparison")
	}
	for _, name := range []string{"transports", "full", "pruned", "detail", "empty", "header", "custom"} {
		t.Run(name, func(t *testing.T) {
			design := errorPropertyProjectionDSL(name)
			if name == "transports" {
				design = errorPropertiesDSL
			}
			dir := filepath.Join(output, name)
			require.NoError(t, os.MkdirAll(dir, 0o700))
			renderDesignModule(t, dir, "example.com/errorproperties", design)
		})
	}
}

func TestErrorPropertyArtifactBuilds(t *testing.T) {
	output := os.Getenv("LOOM_ERROR_PROPERTY_ARTIFACTS")
	if output == "" {
		t.Skip("set LOOM_ERROR_PROPERTY_ARTIFACTS to compile comparison modules")
	}
	for _, name := range []string{"transports", "full", "pruned", "detail", "empty", "header", "custom"} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(output, name)
			for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}} {
				logs, err := testingx.RunCmd(dir, "go", args...)
				require.NoError(t, err, logs)
			}
		})
	}
}

func errorPropertyProjectionDSL(name string) func() {
	return func() {
		d.Service("projection", func() {
			d.Method("fail", func() {
				if name == "custom" {
					d.Error("oops", func() {
						d.Attribute("reason", d.String)
						d.Required("reason")
					})
				} else {
					d.Error("oops", d.ErrorResult)
				}
				d.HTTP(func() {
					d.GET("/fail")
					d.Response("oops", d.StatusBadRequest, func() {
						switch name {
						case "pruned":
							d.Body(func() {
								d.Attribute("detail")
							})
						case "detail":
							d.Body("detail")
						case "empty":
							d.Body(d.Empty)
						case "header":
							d.Header("code:X-Code")
						}
					})
				})
			})
		})
	}
}
