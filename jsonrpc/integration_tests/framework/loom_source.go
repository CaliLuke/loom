package framework

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// repoRootReplace returns a path suitable for the replace directive in go.mod.
// Prefer LOOM_REPO env var for tests; otherwise fall back to relative path.
func (g *Generator) repoRootReplace() string {
	if p := os.Getenv("LOOM_REPO"); p != "" {
		return validatedLocalLoomSource(p)
	}
	if p := configuredLocalLoomSource(); p != "" {
		return p
	}
	dest := filepath.Join(g.workDir, ".loom-pinned")
	if fi, err := os.Stat(dest); err == nil && fi.IsDir() {
		return dest
	}
	remote, commit, err := resolvePinnedLoomSource()
	if err == nil {
		if err := checkoutPinnedLoomSource(dest, remote, commit); err == nil {
			return dest
		}
	}
	root, err := runCommandOutput("", "git", "rev-parse", "--show-toplevel")
	if err == nil && strings.TrimSpace(root) != "" {
		return strings.TrimSpace(root)
	}
	absPath, err := filepath.Abs("../../..")
	if err != nil {
		return "../../../.."
	}
	return absPath
}

func configuredLocalLoomSource() string {
	repoRoot, err := repoTopLevel()
	if err != nil || repoRoot == "" {
		return ""
	}
	return localLoomSourceFromModeFile(filepath.Join(repoRoot, "jsonrpc", "integration_tests", ".loom_source_mode"), repoRoot)
}

func localLoomSourceFromModeFile(path string, defaultLocalSource string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return ""
	}
	if fields[0] != "local" {
		return ""
	}
	if len(fields) < 2 {
		return validatedLocalLoomSource(defaultLocalSource)
	}
	return validatedLocalLoomSource(fields[1])
}

func validatedLocalLoomSource(path string) string {
	if path == "" {
		return ""
	}
	cleaned := filepath.Clean(path)
	goModPath := filepath.Join(cleaned, "go.mod")
	info, err := os.Stat(goModPath)
	if err != nil || info.IsDir() {
		return ""
	}
	return cleaned
}

func repoTopLevel() (string, error) {
	root, err := runCommandOutput("", "git", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(root), nil
}

func resolvePinnedLoomSource() (string, string, error) {
	remoteNames := []string{"fork", "origin"}
	var remote string
	for _, name := range remoteNames {
		out, err := runCommandOutput("", "git", "remote", "get-url", name)
		if err == nil && strings.TrimSpace(out) != "" {
			remote = strings.TrimSpace(out)
			break
		}
	}
	if remote == "" {
		return "", "", fmt.Errorf("resolve git remote: no fork/origin URL available")
	}
	commit, err := runCommandOutput("", "git", "rev-parse", "HEAD")
	if err != nil {
		return "", "", fmt.Errorf("resolve git commit: %w", err)
	}
	return remote, strings.TrimSpace(commit), nil
}

func checkoutPinnedLoomSource(dest, remote, commit string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	if _, err := runCommandOutput("", "git", "init", dest); err != nil {
		return err
	}
	if _, err := runCommandOutput(dest, "git", "remote", "add", "origin", remote); err != nil {
		return err
	}
	if _, err := runCommandOutput(dest, "git", "fetch", "--depth", "1", "origin", commit); err != nil {
		return err
	}
	if _, err := runCommandOutput(dest, "git", "checkout", "--detach", "FETCH_HEAD"); err != nil {
		return err
	}
	return nil
}

func runCommandOutput(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}
