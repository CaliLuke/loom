package scripts_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLintToolchainVersionParity(t *testing.T) {
	for _, c := range []struct {
		name, version string
		valid         bool
	}{
		{"pinned", "2.12.2", true},
		{"newer", "2.13.2", false},
		{"older", "2.11.0", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			bin := filepath.Join(t.TempDir(), "golangci-lint")
			require.NoError(t, os.WriteFile(bin, []byte("#!/bin/sh\nprintf '%s\\n' '"+c.version+"'\n"), 0o700))
			cmd := exec.CommandContext(t.Context(), "bash", "lint_toolchain.sh", "../Makefile")
			cmd.Env = append(os.Environ(), "GOLANGCI_LINT="+bin, "GOLANGCI_LINT_VERSION=v2.12.2")
			out, err := cmd.CombinedOutput()
			if c.valid {
				require.NoError(t, err, "%s", out)
			} else {
				require.Error(t, err)
				require.Contains(t, string(out), "does not match v2.12.2; run make depend")
			}
		})
	}
}
