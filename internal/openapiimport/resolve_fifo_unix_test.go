//go:build unix

package openapiimport

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResolverNonRegularFiles(t *testing.T) {
	if path := os.Getenv("LOOM_TEST_REFERENCE_FIFO"); path != "" {
		if os.Getenv("LOOM_TEST_REFERENCE_CONFINED") == "true" {
			root, err := os.OpenRoot(filepath.Dir(path))
			require.NoError(t, err)
			defer func() {
				require.NoError(t, root.Close())
			}()
			_, _, err = readConfinedDescription(root, filepath.Dir(path), filepath.Base(path))
			require.ErrorContains(t, err, "regular file")
		} else {
			_, _, err := readDescriptionFile(path)
			require.ErrorContains(t, err, "regular file")
		}
		return
	}
	for _, confined := range []string{"false", "true"} {
		t.Run("confined="+confined, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "document.pipe")
			require.NoError(t, syscall.Mkfifo(path, 0o600))
			ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestResolverNonRegularFiles$")
			cmd.Env = append(os.Environ(), "LOOM_TEST_REFERENCE_FIFO="+path, "LOOM_TEST_REFERENCE_CONFINED="+confined)
			output, err := cmd.CombinedOutput()
			require.NoError(t, ctx.Err(), "opening a FIFO must not wait for a writer")
			require.NoError(t, err, "%s", output)
		})
	}
}
