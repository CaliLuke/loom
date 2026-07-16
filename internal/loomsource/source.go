// Package loomsource resolves the Loom checkout used by temporary integration
// modules. It keeps local-vs-remote policy consistent across transport harnesses.
package loomsource

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Mode selects whether temporary modules use a local checkout or a checkout of
// the current commit fetched from the canonical remote.
type Mode string

const (
	// ModeLocal uses the configured local checkout.
	ModeLocal Mode = "local"
	// ModeRemote fetches the current commit from the canonical remote.
	ModeRemote Mode = "remote"
)

// Config is the persisted Loom source selection for one Git worktree.
type Config struct {
	Mode     Mode
	LocalDir string
}

// RepositoryRoot returns the top-level directory for the Git worktree that
// contains dir.
func RepositoryRoot(dir string) (string, error) {
	out, err := runGit(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("resolve repository root: %w", err)
	}
	root := strings.TrimSpace(out)
	if root == "" {
		return "", errors.New("resolve repository root: git returned an empty path")
	}
	return filepath.Clean(root), nil
}

// Resolve returns the checkout to place in a temporary module's Loom replace
// directive. LOOM_DIR overrides the persisted mode for a single invocation.
func Resolve(repoRoot, checkoutDir string) (string, error) {
	if override := os.Getenv("LOOM_DIR"); override != "" {
		path, err := validateLocalSource(override)
		if err != nil {
			return "", fmt.Errorf("LOOM_DIR: %w", err)
		}
		return path, nil
	}

	config, err := ReadMode(repoRoot)
	if err != nil {
		return "", err
	}
	if config.Mode == ModeLocal {
		path := config.LocalDir
		if path == "" {
			path = repoRoot
		}
		localSource, err := validateLocalSource(path)
		if err != nil {
			return "", fmt.Errorf("configured local Loom source: %w", err)
		}
		return localSource, nil
	}

	path, err := checkoutPinnedSource(repoRoot, checkoutDir)
	if err != nil {
		return "", fmt.Errorf("resolve remote Loom source: %w", err)
	}
	return path, nil
}

// ModePath returns the worktree-local, untracked Git metadata path used to
// persist source selection.
func ModePath(repoRoot string) (string, error) {
	out, err := runGit(repoRoot, "rev-parse", "--git-path", "loom-source-mode")
	if err != nil {
		return "", fmt.Errorf("resolve Loom source mode path: %w", err)
	}
	path := strings.TrimSpace(out)
	if path == "" {
		return "", errors.New("resolve Loom source mode path: git returned an empty path")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(repoRoot, path)
	}
	return filepath.Clean(path), nil
}

// ReadMode reads source selection for the given worktree. A missing setting is
// remote mode so CI and fresh checkouts exercise a pushed, reproducible commit.
func ReadMode(repoRoot string) (Config, error) {
	path, err := ModePath(repoRoot)
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{Mode: ModeRemote}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read Loom source mode: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	switch {
	case len(lines) == 1 && lines[0] == string(ModeLocal):
		return Config{Mode: ModeLocal}, nil
	case len(lines) == 2 && lines[0] == string(ModeLocal) && strings.TrimSpace(lines[1]) != "":
		return Config{Mode: ModeLocal, LocalDir: filepath.Clean(lines[1])}, nil
	case len(lines) == 1 && lines[0] == string(ModeRemote):
		return Config{Mode: ModeRemote}, nil
	default:
		return Config{}, fmt.Errorf("invalid Loom source mode in %s", path)
	}
}

// SetMode persists source selection in worktree-local Git metadata. localDir
// may be empty in local mode to select repoRoot.
func SetMode(repoRoot string, mode Mode, localDir string) error {
	var data string
	switch mode {
	case ModeLocal:
		path := localDir
		if path == "" {
			path = repoRoot
		}
		validated, err := validateLocalSource(path)
		if err != nil {
			return fmt.Errorf("set local Loom source: %w", err)
		}
		if filepath.Clean(validated) == filepath.Clean(repoRoot) {
			data = string(ModeLocal) + "\n"
		} else {
			data = string(ModeLocal) + "\n" + validated + "\n"
		}
	case ModeRemote:
		if localDir != "" {
			return errors.New("remote Loom source mode does not accept a local directory")
		}
		data = string(ModeRemote) + "\n"
	default:
		return fmt.Errorf("unsupported Loom source mode %q", mode)
	}

	path, err := ModePath(repoRoot)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		return fmt.Errorf("write Loom source mode: %w", err)
	}
	return nil
}

func validateLocalSource(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", path, err)
	}
	info, err := os.Stat(filepath.Join(absPath, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("Loom checkout %q has no go.mod: %w", absPath, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("Loom checkout %q has a go.mod directory", absPath)
	}
	return filepath.Clean(absPath), nil
}

func checkoutPinnedSource(repoRoot, checkoutDir string) (source string, err error) {
	remote, err := runGit(repoRoot, "remote", "get-url", "origin")
	if err != nil {
		return "", fmt.Errorf("resolve canonical origin remote: %w", err)
	}
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return "", errors.New("resolve canonical origin remote: git returned an empty URL")
	}
	commit, err := runGit(repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("resolve current commit: %w", err)
	}
	commit = strings.TrimSpace(commit)

	if _, err := os.Stat(checkoutDir); err == nil {
		checkedOutCommit, gitErr := runGit(checkoutDir, "rev-parse", "HEAD")
		if gitErr != nil || strings.TrimSpace(checkedOutCommit) != commit {
			return "", fmt.Errorf("pinned Loom checkout %q does not match commit %s", checkoutDir, commit)
		}
		return validateLocalSource(checkoutDir)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect pinned Loom checkout: %w", err)
	}

	parent := filepath.Dir(checkoutDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", fmt.Errorf("create pinned Loom checkout parent: %w", err)
	}
	tempDir, err := os.MkdirTemp(parent, ".loom-pinned-")
	if err != nil {
		return "", fmt.Errorf("create temporary pinned Loom checkout: %w", err)
	}
	defer func() {
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			source = ""
			err = errors.Join(err, fmt.Errorf("remove temporary pinned checkout %s: %w", tempDir, removeErr))
		}
	}()

	if _, err := runGit(tempDir, "init"); err != nil {
		return "", fmt.Errorf("initialize pinned Loom checkout: %w", err)
	}
	if _, err := runGit(tempDir, "remote", "add", "origin", remote); err != nil {
		return "", fmt.Errorf("configure pinned Loom checkout: %w", err)
	}
	if _, err := runGit(tempDir, "fetch", "--depth", "1", "origin", commit); err != nil {
		return "", fmt.Errorf("fetch commit %s from origin: %w", commit, err)
	}
	if _, err := runGit(tempDir, "checkout", "--detach", "FETCH_HEAD"); err != nil {
		return "", fmt.Errorf("check out fetched Loom commit: %w", err)
	}
	if err := os.Rename(tempDir, checkoutDir); err != nil {
		return "", fmt.Errorf("install pinned Loom checkout: %w", err)
	}
	return validateLocalSource(checkoutDir)
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		if message == "" {
			return "", err
		}
		return "", fmt.Errorf("%w: %s", err, message)
	}
	return string(out), nil
}
