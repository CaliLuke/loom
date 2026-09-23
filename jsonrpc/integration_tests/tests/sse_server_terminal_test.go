package tests

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/testingx"
)

// TestJSONRPCSSEServerTerminalResponseUnderRace regenerates the ticktock
// fixture against the current repository and runs its generated-server
// terminal-response tests with the race detector. Those tests force concurrent
// Send calls to queue behind a blocked write while SendAndClose runs, and
// require that no event, comment, open, or error response follows the
// terminal response, that SendError is terminal, and that stream writes honor
// the call context.
func TestJSONRPCSSEServerTerminalResponseUnderRace(t *testing.T) {
	t.Parallel()

	srcDir := filepath.Join("..", "fixtures", "ticktock")
	workDir := filepath.Join(t.TempDir(), "ticktock")
	require.NoError(t, testingx.CopyTree(srcDir, workDir))
	require.NoError(t, testingx.PinLocalReplace(workDir, testingx.RepoRoot()))

	_, err := testingx.RunCmd(workDir, "go", "run", "-mod=mod", "github.com/CaliLuke/loom/cmd/loom", "gen", "example.com/ticktock/design", "-o", ".")
	require.NoError(t, err)

	out, err := testingx.RunCmd(workDir, "go", "test", "-race", "-count=1", "-run", "^TestJSONRPCGeneratedSSEServer", ".")
	require.NoError(t, err, out)
}
