package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	httpdata "github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/stretchr/testify/require"
)

// TestLayoutCaptureArtifacts renders a bounded set of complete contracts for
// independent-process byte comparison and compilation of those exact outputs.
func TestLayoutCaptureArtifacts(t *testing.T) {
	output := os.Getenv("LOOM_LAYOUT_CAPTURE_ARTIFACTS")
	if output == "" {
		t.Skip("set LOOM_LAYOUT_CAPTURE_ARTIFACTS for layout capture qualification")
	}
	for _, test := range []struct {
		name   string
		design func()
	}{
		{"catalog", layoutCatalogDSL},
		{"composition", httpdata.ReferenceCompositionDSL},
		{"media", httpdata.StyledMediaContractsDSL},
		{"presence", httpdata.UntaggedNullFieldsDSL},
		{"views", httpdata.MultipleViewsDSL},
		{"discriminator", httpdata.DiscriminatorHintDSL},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := filepath.Join(output, test.name)
			require.NoError(t, os.MkdirAll(dir, 0o700))
			root := codegen.RunDSL(t, test.design)
			files, err := generationFiles("example.com/layoutcapture/gen", []eval.Root{root})
			require.NoError(t, err)
			for _, file := range mergeFilesByPath(files) {
				_, err := file.Render(dir)
				require.NoError(t, err, file.Path)
			}
			source := os.Getenv("LOOM_DIR")
			if source == "" {
				source = loomModuleSource(t)
			}
			require.True(t, filepath.IsAbs(source))
			module := fmt.Sprintf("module example.com/layoutcapture\n\ngo 1.27.3\n\nrequire github.com/CaliLuke/loom v1.10.0-alpha.30\n\nreplace github.com/CaliLuke/loom => %s\n", source)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0o600))
		})
	}
}

func layoutCatalogDSL() {
	dsl.API("catalog", func() {
		dsl.Meta("http:generate", "server")
		for index := range 8 {
			dsl.Error(fmt.Sprintf("failure_%d", index), dsl.ErrorResult)
		}
		dsl.HTTP(func() {
			for index := range 8 {
				dsl.Response(fmt.Sprintf("failure_%d", index), 400)
			}
		})
	})
	dsl.Service("catalog", func() {
		for index := range 8 {
			dsl.Error(fmt.Sprintf("failure_%d", index))
		}
		for index := range 4 {
			dsl.Method(fmt.Sprintf("method_%d", index), func() {
				dsl.Result(dsl.String)
				dsl.HTTP(func() {
					dsl.GET(fmt.Sprintf("/method/%d", index))
				})
			})
		}
	})
}
