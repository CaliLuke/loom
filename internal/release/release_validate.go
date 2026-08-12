package release

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/mod/semver"
)

func validateVersion(version string) error {
	if !versionPattern.MatchString(version) || !semver.IsValid(version) {
		return fmt.Errorf("VERSION %q must match vX.Y.Z or vX.Y.Z-prerelease", version)
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
	if err := validateVersion(current); err != nil {
		return fmt.Errorf("compare release versions: invalid current %q or target %q", current, target)
	}
	if err := validateVersion(target); err != nil {
		return fmt.Errorf("compare release versions: invalid current %q or target %q", current, target)
	}
	if semver.Compare(target, current) > 0 {
		return nil
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
	version := "v" + strings.Join(parts, ".")
	suffixMatches := versionSuffixPattern.FindSubmatch(contents)
	if len(suffixMatches) != 3 {
		return "", errors.New("read current version: missing Suffix field")
	}
	if suffix := string(suffixMatches[2]); suffix != "" {
		version += "-" + suffix
	}
	return version, nil
}

func updateVersionFiles(root, version string) ([]string, error) {
	matches := versionPattern.FindStringSubmatch(version)
	if len(matches) != 5 {
		return nil, fmt.Errorf("VERSION %q must match vX.Y.Z or vX.Y.Z-prerelease", version)
	}
	if err := updatePackageVersion(root, matches[1:4], matches[4]); err != nil {
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

func updatePackageVersion(root string, parts []string, suffix string) error {
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
	contents, err = replaceExactlyOne(contents, versionSuffixPattern, `${1}"`+suffix+`"`, versionPath)
	if err != nil {
		return err
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
