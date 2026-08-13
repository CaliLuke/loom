// Package docsmeta owns the checked-in documentation version metadata.
package docsmeta

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type versionMarker struct {
	pattern *regexp.Regexp
	count   int
}

type versionTarget struct {
	path    string
	markers []versionMarker
}

const semanticVersionExpression = `v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?`

var (
	semanticVersionPattern = regexp.MustCompile(`^` + semanticVersionExpression + `$`)
	packageVersionFields   = map[string]*regexp.Regexp{
		"Major": regexp.MustCompile(`(?m)^(\s*Major\s*=\s*)[0-9]+`),
		"Minor": regexp.MustCompile(`(?m)^(\s*Minor\s*=\s*)[0-9]+`),
		"Build": regexp.MustCompile(`(?m)^(\s*Build\s*=\s*)[0-9]+`),
	}
	packageVersionSuffixPattern   = regexp.MustCompile(`(?m)^(\s*Suffix\s*=\s*)"([^"]*)"`)
	documentationReferencePattern = regexp.MustCompile(
		`github\.com/CaliLuke/loom(?:/cmd/loom)?@(` + semanticVersionExpression + `)`,
	)
	documentationBannerPattern = regexp.MustCompile(
		"Recommended release: `(" + semanticVersionExpression + ")`",
	)
	unreleasedSincePattern      = regexp.MustCompile(`\*\*Since: unreleased\.\*\*`)
	documentationVersionTargets = []versionTarget{
		{path: "README.md", markers: []versionMarker{{pattern: documentationReferencePattern, count: 2}}},
		{path: ".agents/skills/loom/SKILL.md", markers: []versionMarker{{pattern: documentationReferencePattern, count: 1}}},
		{path: "docs/_index.md", markers: []versionMarker{{pattern: documentationBannerPattern, count: 1}}},
		{path: "docs/code-generation.md", markers: []versionMarker{{pattern: documentationReferencePattern, count: 3}}},
		{path: "docs/quickstart.md", markers: []versionMarker{{pattern: documentationReferencePattern, count: 2}}},
	}
)

// ReadPackageVersion reads the semantic version declared by pkg/version.go.
func ReadPackageVersion(path string) (string, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read current version: %w", err)
	}
	parts := make([]string, 0, 3)
	for _, name := range []string{"Major", "Minor", "Build"} {
		matches := packageVersionFields[name].FindSubmatch(contents)
		if len(matches) != 2 {
			return "", fmt.Errorf("read current version: missing %s field", name)
		}
		value := strings.TrimSpace(strings.TrimPrefix(string(matches[0]), string(matches[1])))
		parts = append(parts, value)
	}
	version := "v" + strings.Join(parts, ".")
	suffixMatches := packageVersionSuffixPattern.FindSubmatch(contents)
	if len(suffixMatches) != 3 {
		return "", errors.New("read current version: missing Suffix field")
	}
	if suffix := string(suffixMatches[2]); suffix != "" {
		version += "-" + suffix
	}
	return version, nil
}

// UpdateVersionMetadata stamps recommendations and unreleased feature notes with version.
func UpdateVersionMetadata(root, version string) ([]string, error) {
	if !semanticVersionPattern.MatchString(version) {
		return nil, fmt.Errorf("update documentation: invalid semantic version %q", version)
	}
	changedSet := make(map[string]struct{}, len(documentationVersionTargets))
	for _, target := range documentationVersionTargets {
		path := filepath.Join(root, filepath.FromSlash(target.path))
		contents, mode, err := readFile(path)
		if err != nil {
			return nil, err
		}
		for _, marker := range target.markers {
			matches := marker.pattern.FindAllSubmatchIndex(contents, -1)
			if len(matches) != marker.count {
				return nil, fmt.Errorf("update %s: expected %d version markers, found %d",
					target.path, marker.count, len(matches))
			}
			contents = marker.pattern.ReplaceAllFunc(contents, func(match []byte) []byte {
				versionRange := marker.pattern.FindSubmatchIndex(match)
				updated := make([]byte, 0, len(match)-versionRange[3]+versionRange[2]+len(version))
				updated = append(updated, match[:versionRange[2]]...)
				updated = append(updated, version...)
				updated = append(updated, match[versionRange[3]:]...)
				return updated
			})
		}
		if err := os.WriteFile(path, contents, mode); err != nil {
			return nil, fmt.Errorf("write %s: %w", target.path, err)
		}
		changedSet[target.path] = struct{}{}
	}
	sinceFiles, err := stampUnreleasedSince(root, version)
	if err != nil {
		return nil, err
	}
	for _, path := range sinceFiles {
		changedSet[path] = struct{}{}
	}
	changed := make([]string, 0, len(changedSet))
	for path := range changedSet {
		changed = append(changed, path)
	}
	sort.Strings(changed)
	return changed, nil
}

// CheckRecommendedVersion reports missing or divergent maintained recommendation markers.
func CheckRecommendedVersion(root, expected string) []string {
	var issues []string
	for _, target := range documentationVersionTargets {
		path := filepath.Join(root, filepath.FromSlash(target.path))
		contents, err := os.ReadFile(path)
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: read version metadata: %v", target.path, err))
			continue
		}
		for _, marker := range target.markers {
			matches := marker.pattern.FindAllSubmatch(contents, -1)
			if len(matches) != marker.count {
				issues = append(issues, fmt.Sprintf("%s: expected %d recommended-version markers, found %d",
					target.path, marker.count, len(matches)))
				continue
			}
			for _, match := range matches {
				if actual := string(match[1]); actual != expected {
					issues = append(issues, fmt.Sprintf(
						"%s: recommended version %s does not match package version %s",
						target.path, actual, expected))
				}
			}
		}
	}
	sort.Strings(issues)
	return issues
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

func stampUnreleasedSince(root, version string) ([]string, error) {
	var changed []string
	docsRoot := filepath.Join(root, "docs")
	err := filepath.WalkDir(docsRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".md") {
			return nil
		}
		contents, mode, err := readFile(path)
		if err != nil {
			return err
		}
		if !unreleasedSincePattern.Match(contents) {
			return nil
		}
		contents = unreleasedSincePattern.ReplaceAll(contents, []byte("**Since: `"+version+"`.**"))
		if err := os.WriteFile(path, contents, mode); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("resolve documentation path %s: %w", path, err)
		}
		changed = append(changed, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("stamp unreleased feature versions: %w", err)
	}
	return changed, nil
}
