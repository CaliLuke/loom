package release

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

var (
	// generatedNotesHeadingPattern matches the section headings GitHub's
	// `--generate-notes` output uses and nothing else, so a heading alone
	// never counts as substantive prose.
	generatedNotesHeadingPattern = regexp.MustCompile(`(?i)^#{1,6}\s*(what's changed|new contributors)\s*:?\s*$`)
	// generatedNotesBulletPattern matches an auto-generated "What's Changed" or
	// "New Contributors" bullet line, which always ends in a link to the pull
	// request, commit, or first-contribution reference on GitHub.
	generatedNotesBulletPattern = regexp.MustCompile(`(?i)^[-*]\s+\S.*\bhttps://github\.com/\S+/(?:pull|commit)/\S+$`)
	// generatedNotesChangelogPattern matches the trailing "Full Changelog"
	// comparison link line GitHub appends to generated notes.
	generatedNotesChangelogPattern = regexp.MustCompile(`(?i)^\*{0,2}full changelog\*{0,2}:\s*https://\S+$`)
)

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
		output, err := runCommand(ctx, config.Root, nil, config.GitHubCommand, "release", "view", config.Version,
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
	expectPrerelease := semver.Prerelease(version) != ""
	if release.Prerelease != expectPrerelease {
		if expectPrerelease {
			return errors.New("GitHub Release is not marked as a prerelease")
		}
		return errors.New("stable GitHub Release is marked as a prerelease")
	}
	body := strings.TrimSpace(release.Body)
	lowerBody := strings.ToLower(body)
	words := strings.Fields(body)
	if len(words) < 12 || (!strings.Contains(lowerBody, "what's changed") &&
		!strings.Contains(lowerBody, "highlights")) {
		return errors.New("GitHub Release does not contain substantive release notes")
	}
	if isGeneratedNotesOnly(body) {
		return errors.New(
			"GitHub Release body is only auto-generated changelog notes; " +
				"add substantive highlights before the release is considered done",
		)
	}
	return nil
}

// isGeneratedNotesOnly reports whether body consists entirely of GitHub's
// `--generate-notes` output: a "What's Changed"/"New Contributors" heading,
// per-PR or per-contributor bullet lines that end in a GitHub link, and a
// trailing "Full Changelog" comparison link, with no other prose. A body that
// merely includes one of those elements alongside real highlights, upgrade
// notes, or other authored text is not flagged, since at least one line will
// fail to match every pattern below.
func isGeneratedNotesOnly(body string) bool {
	sawLine := false
	for line := range strings.SplitSeq(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		switch {
		case generatedNotesHeadingPattern.MatchString(trimmed):
		case generatedNotesBulletPattern.MatchString(trimmed):
		case generatedNotesChangelogPattern.MatchString(trimmed):
		default:
			return false
		}
		sawLine = true
	}
	return sawLine
}

func gitCommandOutput(ctx context.Context, dir string, args ...string) (string, error) {
	output, err := runCommand(ctx, dir, nil, "git", args...)
	return strings.TrimSpace(string(output)), err
}

func runCommand(ctx context.Context, dir string, environment []string, name string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = dir
	command.Env = append(os.Environ(), environment...)
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
