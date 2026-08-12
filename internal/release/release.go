// Package release implements Loom's transactional release workflow.
package release

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// Config controls a release run.
type Config struct {
	// Root is the clean Loom repository to release.
	Root string
	// Version is the exact semantic version tag, including its v prefix.
	Version string
	// CanonicalRemote overrides the expected origin URL. Tests use a local bare repository.
	CanonicalRemote string
	// GitHubRepo overrides the repository passed to gh.
	GitHubRepo string
	// PreflightCommand overrides the make executable used for release-preflight.
	PreflightCommand string
	// GitHubCommand overrides the gh executable used to inspect the published release.
	GitHubCommand string
	// PollAttempts controls how many times GitHub release metadata is inspected.
	PollAttempts int
	// PollInterval controls the delay between GitHub release inspections.
	PollInterval time.Duration
}

type githubRelease struct {
	TagName    string `json:"tagName"`
	Body       string `json:"body"`
	Draft      bool   `json:"isDraft"`
	Prerelease bool   `json:"isPrerelease"`
}

type releaseRun struct {
	config        Config
	root          string
	head          string
	stage         string
	cacheDir      string
	worktreeAdded bool
	tagCreated    bool
	pushed        bool
}

const (
	defaultCanonicalRemote = "github.com/CaliLuke/loom"
	defaultGitHubRepo      = "CaliLuke/loom"
	defaultPollAttempts    = 60
	defaultPollInterval    = 5 * time.Second
	cleanupTimeout         = 30 * time.Second
)

var (
	versionPattern = regexp.MustCompile(
		`^v([0-9]+)\.([0-9]+)\.([0-9]+)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`,
	)
	versionFields = map[string]*regexp.Regexp{
		"Major": regexp.MustCompile(`(?m)^(\s*Major\s*=\s*)[0-9]+`),
		"Minor": regexp.MustCompile(`(?m)^(\s*Minor\s*=\s*)[0-9]+`),
		"Build": regexp.MustCompile(`(?m)^(\s*Build\s*=\s*)[0-9]+`),
	}
	versionSuffixPattern = regexp.MustCompile(`(?m)^(\s*Suffix\s*=\s*)"([^"]*)"`)
	readmeVersionPattern = regexp.MustCompile(
		`go install github\.com/CaliLuke/loom/cmd/loom@v[0-9]+\.[0-9]+\.[0-9]+` +
			`(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?`,
	)
	fixtureVersionPattern = regexp.MustCompile(
		`"loom_version"\s*:\s*"v[0-9]+\.[0-9]+\.[0-9]+` +
			`(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?"`,
	)
)

// Run validates, stages, verifies, publishes, and confirms one Loom release.
// Preflight preparation occurs in a temporary detached worktree, so a failed
// preflight cannot modify the caller's checkout.
func Run(ctx context.Context, config Config) error {
	config = withDefaults(config)
	if err := validateConfig(config); err != nil {
		return err
	}
	root, err := filepath.Abs(config.Root)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	config.Root = root
	if err := validateVersion(config.Version); err != nil {
		return err
	}
	head, err := validateRepository(ctx, config)
	if err != nil {
		return err
	}
	release := &releaseRun{config: config, root: root, head: head}
	return release.execute(ctx)
}

func (run *releaseRun) execute(ctx context.Context) (resultErr error) {
	stage, err := os.MkdirTemp(filepath.Dir(run.root), "loom-release-")
	if err != nil {
		return fmt.Errorf("create release staging path: %w", err)
	}
	run.stage = stage
	run.cacheDir = stage + "-cache"
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
		defer cancel()
		resultErr = errors.Join(resultErr, run.cleanup(cleanupCtx))
	}()
	if err := run.createWorktree(ctx); err != nil {
		return err
	}
	changedFiles, err := run.prepare(ctx)
	if err != nil {
		return err
	}
	return run.publish(ctx, changedFiles)
}

func (run *releaseRun) createWorktree(ctx context.Context) error {
	if err := os.Remove(run.stage); err != nil {
		return fmt.Errorf("prepare release staging path: %w", err)
	}
	if _, err := runCommand(ctx, run.root, "git", "worktree", "add", "--detach", run.stage, run.head); err != nil {
		return fmt.Errorf("create release worktree: %w", err)
	}
	run.worktreeAdded = true
	status, err := gitCommandOutput(ctx, run.stage, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return fmt.Errorf("inspect clean release worktree: %w", err)
	}
	if status != "" {
		return fmt.Errorf("release worktree is not clean before preparation:\n%s", status)
	}
	return nil
}

func (run *releaseRun) prepare(ctx context.Context) ([]string, error) {
	changedFiles, err := updateVersionFiles(run.stage, run.config.Version)
	if err != nil {
		return nil, err
	}
	if err := streamCommand(ctx, run.stage, []string{
		"LOOM_DIR=" + run.stage,
		"GOLANGCI_LINT_CACHE=" + run.cacheDir,
	},
		run.config.PreflightCommand, "release-preflight"); err != nil {
		return nil, fmt.Errorf("release preflight: %w", err)
	}
	if err := validateStagedChanges(ctx, run.stage, changedFiles); err != nil {
		return nil, err
	}
	return changedFiles, nil
}

func (run *releaseRun) publish(ctx context.Context, changedFiles []string) error {
	addArgs := append([]string{"add", "--"}, changedFiles...)
	if _, err := runCommand(ctx, run.stage, "git", addArgs...); err != nil {
		return fmt.Errorf("stage release files: %w", err)
	}
	if _, err := runCommand(ctx, run.stage, "git", "diff", "--cached", "--check"); err != nil {
		return fmt.Errorf("validate staged release diff: %w", err)
	}
	if _, err := runCommand(ctx, run.stage, "git", "commit", "-m", "Release "+run.config.Version); err != nil {
		return fmt.Errorf("create release commit: %w", err)
	}
	releaseCommit, err := gitCommandOutput(ctx, run.stage, "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("resolve release commit: %w", err)
	}
	if _, err := runCommand(ctx, run.stage, "git", "tag", "-a", run.config.Version,
		"-m", "Release "+run.config.Version); err != nil {
		return fmt.Errorf("create release tag: %w", err)
	}
	run.tagCreated = true
	if err := streamCommand(ctx, run.stage, nil, "git", "push", "--atomic", "origin",
		"HEAD:refs/heads/main", "refs/tags/"+run.config.Version); err != nil {
		return fmt.Errorf("atomically push release branch and tag: %w", err)
	}
	run.pushed = true
	if err := verifyRemoteRefs(ctx, run.stage, run.config.Version, releaseCommit); err != nil {
		return err
	}
	if err := waitForRelease(ctx, run.config); err != nil {
		return err
	}
	if _, err := runCommand(ctx, run.root, "git", "merge", "--ff-only", releaseCommit); err != nil {
		return fmt.Errorf("fast-forward caller checkout to release commit: %w", err)
	}
	return nil
}

func (run *releaseRun) cleanup(ctx context.Context) error {
	var cleanupErr error
	if run.tagCreated && !run.pushed {
		if _, err := runCommand(ctx, run.root, "git", "tag", "-d", run.config.Version); err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("remove unpublished release tag: %w", err))
		}
	}
	if run.worktreeAdded {
		if _, err := runCommand(ctx, run.root, "git", "worktree", "remove", "--force", run.stage); err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("remove release worktree: %w", err))
		}
	}
	if err := os.RemoveAll(run.stage); err != nil {
		cleanupErr = errors.Join(cleanupErr, fmt.Errorf("remove release staging directory: %w", err))
	}
	if err := os.RemoveAll(run.cacheDir); err != nil {
		cleanupErr = errors.Join(cleanupErr, fmt.Errorf("remove release cache directory: %w", err))
	}
	return cleanupErr
}

func validateConfig(config Config) error {
	if config.PollAttempts < 1 {
		return errors.New("GitHub Release poll attempts must be positive")
	}
	if config.PollInterval < 0 {
		return errors.New("GitHub Release poll interval cannot be negative")
	}
	return validateVersion(config.Version)
}

func withDefaults(config Config) Config {
	if config.Root == "" {
		config.Root = "."
	}
	if config.CanonicalRemote == "" {
		config.CanonicalRemote = defaultCanonicalRemote
	}
	if config.GitHubRepo == "" {
		config.GitHubRepo = defaultGitHubRepo
	}
	if config.PreflightCommand == "" {
		config.PreflightCommand = "make"
	}
	if config.GitHubCommand == "" {
		config.GitHubCommand = "gh"
	}
	if config.PollAttempts == 0 {
		config.PollAttempts = defaultPollAttempts
	}
	if config.PollInterval == 0 {
		config.PollInterval = defaultPollInterval
	}
	return config
}
