//go:build windows

package testdatacompile

import "os/exec"

// The full compile tier is skipped on Windows; ordinary catalog checks still
// compile. exec.CommandContext retains its default process cancellation.
func cancelProcessGroup(cmd *exec.Cmd) {
}
