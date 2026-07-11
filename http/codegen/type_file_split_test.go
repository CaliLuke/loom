package codegen

import (
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/stretchr/testify/require"
)

func TestLargeHTTPTypeFilesSplitByConcern(t *testing.T) {
	root := RunHTTPDSL(t, largeTypeFileDSL)
	services := CreateHTTPServices(root)

	clientFiles := ClientTypeFiles("gen", services)
	serverFiles := ServerTypeFiles("gen", services)

	requireTypeFilePaths(t, clientFiles, "client")
	requireTypeFilePaths(t, serverFiles, "server")

	dir := t.TempDir()
	for _, file := range append(clientFiles, serverFiles...) {
		_, err := file.Render(dir)
		require.NoError(t, err)
	}
}

func requireTypeFilePaths(t *testing.T, files []*codegen.File, pkg string) {
	t.Helper()
	paths := make([]string, len(files))
	for i, file := range files {
		paths[i] = filepath.Base(file.Path)
		require.Contains(t, filepath.ToSlash(file.Path), "/"+pkg+"/")
	}
	require.NotContains(t, paths, "types.go")
	for _, want := range []string{"types_requests.go", "types_responses.go", "types_validation.go"} {
		require.True(t, slices.Contains(paths, want), "missing %s in %#v", want, paths)
	}
}

func largeTypeFileDSL() {
	Service("LargeTypes", func() {
		for i := range 10 {
			idx := i
			Method(fmt.Sprintf("Method%d", idx), func() {
				Payload(func() {
					Attribute("name", String, func() {
						MinLength(1)
					})
					Required("name")
				})
				Result(func() {
					Attribute("value", String, func() {
						MinLength(1)
					})
					Required("value")
				})
				HTTP(func() {
					POST(fmt.Sprintf("/items/%d", idx))
					Response(StatusOK)
				})
			})
		}
	})
}
