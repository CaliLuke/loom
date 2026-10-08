// Package docsmeta owns the checked-in documentation version metadata.
package docsmeta

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
	documentationReferencePattern = regexp.MustCompile(
		`github\.com/CaliLuke/loom(?:/cmd/loom)?@(` + semanticVersionExpression + `)`,
	)
	documentationBannerPattern = regexp.MustCompile(
		"Recommended release: `(" + semanticVersionExpression + ")`",
	)
	documentationVersionTargets = []versionTarget{
		{path: "README.md", markers: []versionMarker{{pattern: documentationReferencePattern, count: 2}}},
		{path: ".agents/skills/loom/SKILL.md", markers: []versionMarker{{pattern: documentationReferencePattern, count: 1}}},
		{path: "docs/_index.md", markers: []versionMarker{{pattern: documentationBannerPattern, count: 1}}},
		{path: "docs/code-generation.md", markers: []versionMarker{{pattern: documentationReferencePattern, count: 3}}},
		{path: "docs/quickstart.md", markers: []versionMarker{{pattern: documentationReferencePattern, count: 2}}},
	}
)

// RecommendedVersion reads the explicitly curated documentation recommendation.
// Daily publication never changes this pin or historical feature annotations.
func RecommendedVersion(root string) (string, error) {
	contents, err := os.ReadFile(filepath.Join(root, "docs/_index.md"))
	if err != nil {
		return "", fmt.Errorf("read recommended version: %w", err)
	}
	matches := documentationBannerPattern.FindAllSubmatch(contents, -1)
	if len(matches) != 1 {
		return "", errors.New("expected exactly one recommended release banner")
	}
	return string(matches[0][1]), nil
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
						"%s: recommended version %s does not match recommended version %s",
						target.path, actual, expected))
				}
			}
		}
	}
	sort.Strings(issues)
	return issues
}
