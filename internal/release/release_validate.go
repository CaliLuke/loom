package release

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/mod/semver"
)

var (
	shaPattern   = regexp.MustCompile(`^[0-9a-f]{40}$`)
	alphaPattern = regexp.MustCompile(`^(v[0-9]+\.[0-9]+\.[0-9]+)-alpha\.([1-9][0-9]*)$`)
)

func chooseAlpha(base, source string, tags []tag) (string, error) {
	if !semver.IsValid(base) || semver.Canonical(base) != base || semver.Prerelease(base) != "" {
		return "", errors.New("release train must be a canonical stable version")
	}
	next := 1
	existing := ""
	for _, t := range tags {
		if t.version == base {
			return "", nil
		}
		if t.sha == source && semver.Prerelease(t.version) == "" {
			return "", nil
		}
		match := alphaPattern.FindStringSubmatch(t.version)
		if len(match) != 3 || match[1] != base {
			if t.sha == source {
				return "", nil
			}
			continue
		}
		number, err := strconv.Atoi(match[2])
		if err != nil {
			return "", err
		}
		if number >= next {
			next = number + 1
		}
		if t.sha == source && (existing == "" || semver.Compare(t.version, existing) > 0) {
			existing = t.version
		}
	}
	if existing != "" {
		return existing, nil
	}
	return fmt.Sprintf("%s-alpha.%d", base, next), nil
}

func validatePromotion(alpha, stable string) error {
	match := alphaPattern.FindStringSubmatch(alpha)
	if len(match) != 3 || !semver.IsValid(alpha) || match[1] != stable {
		return errors.New("promotion must map vX.Y.Z-alpha.N to vX.Y.Z")
	}
	return nil
}

func findTag(tags []tag, version string) string {
	for _, t := range tags {
		if t.version == version {
			return t.sha
		}
	}
	return ""
}

func (p publisher) tags(ctx context.Context) ([]tag, error) {
	output, err := p.git(ctx, "for-each-ref", "--format=%(refname:short) %(objecttype) %(objectname) %(*objectname)", "refs/tags/v*")
	if err != nil {
		return nil, err
	}
	var tags []tag
	for line := range strings.SplitSeq(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || !semver.IsValid(fields[0]) {
			continue
		}
		sha := fields[2]
		if fields[1] == "tag" {
			if len(fields) != 4 {
				return nil, fmt.Errorf("invalid annotated tag %q", line)
			}
			sha = fields[3]
		} else if fields[1] != "commit" {
			return nil, fmt.Errorf("release tag does not identify commit: %q", line)
		}
		tags = append(tags, tag{fields[0], sha})
	}
	return tags, nil
}

func (p publisher) notes(ctx context.Context, version, source string, tags []tag) (string, error) {
	stable := !strings.Contains(version, "-alpha.")
	base := ""
	for _, t := range tags {
		if t.version == version || semver.Compare(t.version, version) >= 0 || (stable && strings.Contains(t.version, "-")) {
			continue
		}
		if base == "" || semver.Compare(t.version, base) > 0 {
			base = t.version
		}
	}
	selection := source
	if base != "" {
		selection = base + ".." + source
	}
	changes, err := p.git(ctx, "log", "--first-parent", "--format=- %s (%h)", selection)
	if err != nil {
		return "", err
	}
	if changes == "" {
		return "", errors.New("release has no source commits since previous version")
	}
	status := "This alpha is a development snapshot. Generated APIs may change."
	if stable {
		status = "This stable release promotes " + p.config.Alpha + " without changing its source."
	}
	notes := fmt.Sprintf("## Highlights\n\n%s\n\n%s\n\n## Upgrade\n\nInstall the CLI and module at `%s`, then regenerate services with `loom gen <module-import-path>/design`.\n", status, changes, version)
	if base != "" {
		notes += fmt.Sprintf("\n[Full changelog](https://github.com/%s/compare/%s...%s)\n", repository, base, version)
	}
	return notes, nil
}
