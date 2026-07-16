// Package release implements Loom's transactional release workflow.
package release

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Config controls a release run.
type Config struct {
	// Root is the clean Loom repository to release.
	Root string
	// Version is the exact stable semantic version tag, including its v prefix.
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
	versionPattern = regexp.MustCompile(`^v([0-9]+)\.([0-9]+)\.([0-9]+)$`)
	versionFields  = map[string]*regexp.Regexp{
		"Major": regexp.MustCompile(`(?m)^(\s*Major\s*=\s*)[0-9]+`),
		"Minor": regexp.MustCompile(`(?m)^(\s*Minor\s*=\s*)[0-9]+`),
		"Build": regexp.MustCompile(`(?m)^(\s*Build\s*=\s*)[0-9]+`),
	}
	readmeVersionPattern = regexp.MustCompile(
		`go install github\.com/CaliLuke/loom/cmd/loom@v[0-9]+\.[0-9]+\.[0-9]+`,
	)
	fixtureVersionPattern = regexp.MustCompile(`"loom_version"\s*:\s*"v[0-9]+\.[0-9]+\.[0-9]+"`)
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
	if err := streamCommand(ctx, run.stage, []string{"LOOM_DIR=" + run.stage},
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

func validateVersion(version string) error {
	if !versionPattern.MatchString(version) {
		return fmt.Errorf("VERSION %q must match vX.Y.Z", version)
	}
	return nil
}

func validateRepository(ctx context.Context, config Config) (string, error) {
	status, err := gitCommandOutput(ctx, config.Root, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return "", fmt.Errorf("inspect repository status: %w", err)
	}
	if status != "" {
		return "", fmt.Errorf("loom repository has uncommitted changes:\n%s", status)
	}
	branch, err := gitCommandOutput(ctx, config.Root, "branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("inspect current branch: %w", err)
	}
	if branch != "main" {
		return "", fmt.Errorf("release must run from main, found %q", branch)
	}
	remote, err := gitCommandOutput(ctx, config.Root, "remote", "get-url", "origin")
	if err != nil {
		return "", fmt.Errorf("inspect origin: %w", err)
	}
	if !sameRemote(remote, config.CanonicalRemote) {
		return "", fmt.Errorf("origin %q is not canonical Loom remote %q", remote, config.CanonicalRemote)
	}
	head, err := gitCommandOutput(ctx, config.Root, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("resolve HEAD: %w", err)
	}
	remoteMain, err := remoteRef(ctx, config.Root, "refs/heads/main")
	if err != nil {
		return "", fmt.Errorf("resolve origin/main: %w", err)
	}
	if head != remoteMain {
		return "", fmt.Errorf("main HEAD %s does not match origin/main %s", head, remoteMain)
	}
	currentVersion, err := readCurrentVersion(filepath.Join(config.Root, "pkg", "version.go"))
	if err != nil {
		return "", err
	}
	if err := validateVersionAdvance(currentVersion, config.Version); err != nil {
		return "", err
	}
	localTag, err := gitCommandOutput(ctx, config.Root, "tag", "--list", config.Version)
	if err != nil {
		return "", fmt.Errorf("inspect local release tag: %w", err)
	}
	if localTag != "" {
		return "", fmt.Errorf("release tag %s already exists locally", config.Version)
	}
	remoteTag, err := gitCommandOutput(ctx, config.Root, "ls-remote", "--tags", "origin", "refs/tags/"+config.Version)
	if err != nil {
		return "", fmt.Errorf("inspect remote release tag: %w", err)
	}
	if remoteTag != "" {
		return "", fmt.Errorf("release tag %s already exists on origin", config.Version)
	}
	return head, nil
}

func validateVersionAdvance(current, target string) error {
	currentParts := versionPattern.FindStringSubmatch(current)
	targetParts := versionPattern.FindStringSubmatch(target)
	if len(currentParts) != 4 || len(targetParts) != 4 {
		return fmt.Errorf("compare release versions: invalid current %q or target %q", current, target)
	}
	for index := 1; index < 4; index++ {
		currentPart, err := strconv.Atoi(currentParts[index])
		if err != nil {
			return fmt.Errorf("parse current release version %q: %w", current, err)
		}
		targetPart, err := strconv.Atoi(targetParts[index])
		if err != nil {
			return fmt.Errorf("parse target release version %q: %w", target, err)
		}
		if targetPart > currentPart {
			return nil
		}
		if targetPart < currentPart {
			break
		}
	}
	return fmt.Errorf("release VERSION %s must be greater than current version %s", target, current)
}

func sameRemote(actual, canonical string) bool {
	if filepath.IsAbs(canonical) {
		actualPath, actualErr := filepath.Abs(actual)
		canonicalPath, canonicalErr := filepath.Abs(canonical)
		return actualErr == nil && canonicalErr == nil && filepath.Clean(actualPath) == filepath.Clean(canonicalPath)
	}
	normalize := func(value string) string {
		value = strings.TrimSpace(value)
		value = strings.TrimSuffix(value, ".git")
		value = strings.TrimPrefix(value, "https://")
		value = strings.TrimPrefix(value, "ssh://git@")
		value = strings.TrimPrefix(value, "git@")
		value = strings.Replace(value, "github.com:", "github.com/", 1)
		return value
	}
	return normalize(actual) == normalize(canonical)
}

func readCurrentVersion(path string) (string, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read current version: %w", err)
	}
	parts := make([]string, 0, 3)
	for _, name := range []string{"Major", "Minor", "Build"} {
		matches := versionFields[name].FindSubmatch(contents)
		if len(matches) != 2 {
			return "", fmt.Errorf("read current version: missing %s field", name)
		}
		value := strings.TrimSpace(strings.TrimPrefix(string(matches[0]), string(matches[1])))
		parts = append(parts, value)
	}
	return "v" + strings.Join(parts, "."), nil
}

func updateVersionFiles(root, version string) ([]string, error) {
	matches := versionPattern.FindStringSubmatch(version)
	if len(matches) != 4 {
		return nil, fmt.Errorf("VERSION %q must match vX.Y.Z", version)
	}
	if err := updatePackageVersion(root, matches[1:]); err != nil {
		return nil, err
	}
	if err := updateReadmeVersion(root, version); err != nil {
		return nil, err
	}
	fixtures, err := updateFixtureVersions(root, version)
	if err != nil {
		return nil, err
	}
	changed := append([]string{"README.md", "pkg/version.go"}, fixtures...)
	sort.Strings(changed)
	return changed, nil
}

func updatePackageVersion(root string, parts []string) error {
	versionPath := filepath.Join(root, "pkg", "version.go")
	contents, mode, err := readFile(versionPath)
	if err != nil {
		return err
	}
	for index, name := range []string{"Major", "Minor", "Build"} {
		contents, err = replaceExactlyOne(contents, versionFields[name], "${1}"+parts[index], versionPath)
		if err != nil {
			return err
		}
	}
	if err := os.WriteFile(versionPath, contents, mode); err != nil {
		return fmt.Errorf("write %s: %w", versionPath, err)
	}
	return nil
}

func updateReadmeVersion(root, version string) error {
	readmePath := filepath.Join(root, "README.md")
	contents, mode, err := readFile(readmePath)
	if err != nil {
		return err
	}
	contents, err = replaceExactlyOne(contents, readmeVersionPattern,
		"go install github.com/CaliLuke/loom/cmd/loom@"+version, readmePath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(readmePath, contents, mode); err != nil {
		return fmt.Errorf("write %s: %w", readmePath, err)
	}
	return nil
}

func updateFixtureVersions(root, version string) ([]string, error) {
	fixtures := make([]string, 0, 4)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && entry.Name() == ".git" {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		slashed := filepath.ToSlash(relative)
		if !strings.Contains(slashed, "/integration_tests/fixtures/") || !strings.HasSuffix(slashed, "/gen/loom.json") {
			return nil
		}
		contents, mode, err := readFile(path)
		if err != nil {
			return err
		}
		contents, err = replaceExactlyOne(contents, fixtureVersionPattern,
			`"loom_version": "`+version+`"`, path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, contents, mode); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		fixtures = append(fixtures, slashed)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("update integration fixture versions: %w", err)
	}
	if len(fixtures) == 0 {
		return nil, errors.New("update integration fixture versions: no loom.json fixtures found")
	}
	return fixtures, nil
}

func readFile(path string) ([]byte, fs.FileMode, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, 0, fmt.Errorf("inspect %s: %w", path, err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, fmt.Errorf("read %s: %w", path, err)
	}
	return contents, info.Mode().Perm(), nil
}

func replaceExactlyOne(contents []byte, pattern *regexp.Regexp, replacement, path string) ([]byte, error) {
	if count := len(pattern.FindAll(contents, -1)); count != 1 {
		return nil, fmt.Errorf("update %s: expected exactly one version marker, found %d", path, count)
	}
	return pattern.ReplaceAll(contents, []byte(replacement)), nil
}

func validateStagedChanges(ctx context.Context, root string, expected []string) error {
	output, err := runCommand(ctx, root, "git", "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return fmt.Errorf("inspect staged release changes: %w", err)
	}
	status := strings.TrimSuffix(string(output), "\n")
	actual := make([]string, 0, len(expected))
	for line := range strings.SplitSeq(status, "\n") {
		if line == "" {
			continue
		}
		if len(line) < 4 {
			return fmt.Errorf("inspect staged release changes: malformed git status line %q", line)
		}
		actual = append(actual, filepath.ToSlash(strings.TrimSpace(line[3:])))
	}
	sort.Strings(actual)
	if strings.Join(actual, "\n") != strings.Join(expected, "\n") {
		return fmt.Errorf("release preflight changed unexpected files:\nexpected:\n%s\nactual:\n%s",
			strings.Join(expected, "\n"), strings.Join(actual, "\n"))
	}
	return nil
}

func verifyRemoteRefs(ctx context.Context, root, version, commit string) error {
	mainCommit, err := remoteRef(ctx, root, "refs/heads/main")
	if err != nil {
		return fmt.Errorf("verify pushed main: %w", err)
	}
	if mainCommit != commit {
		return fmt.Errorf("verify pushed main: expected %s, found %s", commit, mainCommit)
	}
	tagCommit, err := remoteRef(ctx, root, "refs/tags/"+version+"^{}")
	if err != nil {
		return fmt.Errorf("verify pushed release tag: %w", err)
	}
	if tagCommit != commit {
		return fmt.Errorf("verify pushed release tag: expected %s, found %s", commit, tagCommit)
	}
	return nil
}

func remoteRef(ctx context.Context, root, ref string) (string, error) {
	output, err := gitCommandOutput(ctx, root, "ls-remote", "origin", ref)
	if err != nil {
		return "", err
	}
	fields := strings.Fields(output)
	if len(fields) != 2 {
		return "", fmt.Errorf("remote ref %s not found", ref)
	}
	return fields[0], nil
}

func waitForRelease(ctx context.Context, config Config) error {
	var lastErr error
	for attempt := 1; attempt <= config.PollAttempts; attempt++ {
		output, err := runCommand(ctx, config.Root, config.GitHubCommand, "release", "view", config.Version,
			"--repo", config.GitHubRepo, "--json", "tagName,body,isDraft,isPrerelease")
		if err == nil {
			err = validateRelease(config.Version, output)
		}
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt == config.PollAttempts {
			break
		}
		timer := time.NewTimer(config.PollInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return fmt.Errorf("wait for GitHub Release: %w", ctx.Err())
		case <-timer.C:
		}
	}
	return fmt.Errorf("GitHub Release %s was not published with substantive notes after %d attempts: %w",
		config.Version, config.PollAttempts, lastErr)
}

func validateRelease(version string, data []byte) error {
	var release githubRelease
	if err := json.Unmarshal(data, &release); err != nil {
		return fmt.Errorf("decode GitHub Release metadata: %w", err)
	}
	if release.TagName != version {
		return fmt.Errorf("GitHub Release tag %q does not match %q", release.TagName, version)
	}
	if release.Draft {
		return errors.New("GitHub Release is still a draft")
	}
	if release.Prerelease {
		return errors.New("stable GitHub Release is marked as a prerelease")
	}
	body := strings.TrimSpace(release.Body)
	lowerBody := strings.ToLower(body)
	words := strings.Fields(body)
	if len(words) < 12 || (!strings.Contains(lowerBody, "what's changed") &&
		!strings.Contains(lowerBody, "highlights")) {
		return errors.New("GitHub Release does not contain substantive release notes")
	}
	return nil
}

func gitCommandOutput(ctx context.Context, dir string, args ...string) (string, error) {
	output, err := runCommand(ctx, dir, "git", args...)
	return strings.TrimSpace(string(output)), err
}

func runCommand(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func streamCommand(ctx context.Context, dir string, environment []string, name string, args ...string) error {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = dir
	command.Env = append(os.Environ(), environment...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}
