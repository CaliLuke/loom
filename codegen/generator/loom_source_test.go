package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/loomsource"
)

// sharedLoomSource memoizes the Loom module directory that the generated test
// modules of this package replace github.com/CaliLuke/loom with. In remote
// source mode every resolution into a new directory fetches its own pinned
// checkout, and the Go build cache, which keys a package build by its
// directory, then rebuilds every Loom package for each generated module.
// TestMain removes the shared checkout after the tests of the package finish.
var sharedLoomSource struct {
	once sync.Once
	root string
	path string
	err  error
}

func TestMain(m *testing.M) {
	code := m.Run()
	if sharedLoomSource.root != "" {
		if err := os.RemoveAll(sharedLoomSource.root); err != nil {
			fmt.Fprintf(os.Stderr, "remove pinned Loom checkout: %v\n", err)
			if code == 0 {
				code = 1
			}
		}
	}
	os.Exit(code)
}

// loomModuleSource returns the Loom module directory selected by LOOM_DIR or
// the persisted source mode. Every test of the package shares one resolution,
// so remote mode fetches and builds the pinned commit once per test process.
func loomModuleSource(t *testing.T) string {
	t.Helper()
	sharedLoomSource.once.Do(func() {
		repoRoot, err := loomsource.RepositoryRoot(".")
		if err != nil {
			sharedLoomSource.err = err
			return
		}
		root, err := os.MkdirTemp("", "loom-generator-source-")
		if err != nil {
			sharedLoomSource.err = fmt.Errorf("create Loom source checkout directory: %w", err)
			return
		}
		sharedLoomSource.root = root
		sharedLoomSource.path, sharedLoomSource.err = loomsource.Resolve(repoRoot, filepath.Join(root, "loom-pinned"))
	})
	require.NoError(t, sharedLoomSource.err)
	return sharedLoomSource.path
}
