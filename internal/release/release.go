// Package release publishes immutable, CI-verified source commits on GitHub.
package release

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type (
	// Config selects an alpha source or an existing alpha to promote.
	Config struct {
		// Mode is alpha, daily, or promote.
		Mode string
		// Source is the exact full commit SHA for alpha and daily publication.
		Source string
		// Alpha is the existing prerelease tag to promote.
		Alpha string
		// Version optionally names an alpha explicitly; promotion requires a stable version.
		Version string
		// Root is the full-history checkout containing the publisher.
		Root string
	}
	command   func(context.Context, string, ...string) ([]byte, error)
	publisher struct {
		config Config
		run    command
	}
	tag             struct{ version, sha string }
	releaseMetadata struct {
		Tag        string `json:"tagName"`
		Body       string `json:"body"`
		Draft      bool   `json:"isDraft"`
		Prerelease bool   `json:"isPrerelease"`
	}
)

const repository = "CaliLuke/loom"

// Run verifies eligibility and publishes without modifying source or main.
// The caller must serialize publication using the workflow's repository-wide lock.
func Run(ctx context.Context, config Config) error {
	if config.Root == "" {
		config.Root = "."
	}
	p := publisher{config: config}
	p.run = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir = config.Root
		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("%s: %w: %s", name, err, output)
		}
		return output, nil
	}
	return p.publish(ctx)
}

// Dispatch starts the trusted main-branch workflow; local callers never publish tags.
func Dispatch(ctx context.Context, config Config) error {
	if config.Mode != "alpha" && config.Mode != "promote" {
		return errors.New("mode must be alpha or promote")
	}
	if config.Mode == "alpha" && !shaPattern.MatchString(config.Source) {
		return errors.New("alpha requires a full source SHA")
	}
	if config.Mode == "promote" {
		if err := validatePromotion(config.Alpha, config.Version); err != nil {
			return err
		}
	}
	cmd := exec.CommandContext(ctx, "gh", "workflow", "run", "release.yml", "--repo", repository, "--ref", "main",
		"-f", "mode="+config.Mode, "-f", "source="+config.Source, "-f", "alpha="+config.Alpha, "-f", "version="+config.Version)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("dispatch release: %w", err)
	}
	return nil
}

func (p publisher) publish(ctx context.Context) error {
	if p.config.Mode != "alpha" && p.config.Mode != "daily" && p.config.Mode != "promote" {
		return errors.New("invalid release mode")
	}
	remote, err := p.git(ctx, "remote", "get-url", "origin")
	if err != nil {
		return err
	}
	if remote != "https://github.com/"+repository+".git" && remote != "https://github.com/"+repository && remote != "git@github.com:"+repository+".git" {
		return errors.New("origin is not canonical Loom")
	}
	if _, err := p.git(ctx, "fetch", "origin", "main", "--tags"); err != nil {
		return err
	}
	tags, err := p.tags(ctx)
	if err != nil {
		return err
	}
	source, base, err := p.selectSource(ctx, tags)
	if err != nil {
		return err
	}
	version := p.config.Version
	if p.config.Mode != "promote" {
		selected, err := chooseAlpha(base, source, tags)
		if err != nil {
			return err
		}
		if selected == "" {
			fmt.Println("No alpha: this source is released or the release train is closed.")
			return nil
		}
		if version != "" && version != selected {
			return fmt.Errorf("alpha version must be %s for this source", selected)
		}
		version = selected
	}
	if err := p.validateTarget(ctx, source, version, tags); err != nil {
		return err
	}
	existing, err := p.releaseIfExists(ctx, version)
	if err != nil {
		return err
	}
	if existing != nil && !existing.Draft {
		if err := p.verifyPublication(ctx, *existing, version, source); err != nil {
			return err
		}
		fmt.Printf("Release %s already published; no new commits to release.\n", version)
		return nil
	}
	evidence, err := p.waitForCI(ctx, source)
	if err != nil {
		return err
	}
	return p.publishRelease(ctx, source, version, tags, existing, evidence)
}

func (p publisher) git(ctx context.Context, args ...string) (string, error) {
	data, err := p.run(ctx, "git", args...)
	return strings.TrimSpace(string(data)), err
}

func (p publisher) release(ctx context.Context, version string) (releaseMetadata, error) {
	data, err := p.run(ctx, "gh", "release", "view", version, "--repo", repository, "--json", "tagName,body,isDraft,isPrerelease")
	if err != nil {
		return releaseMetadata{}, err
	}
	var result releaseMetadata
	if err := json.Unmarshal(data, &result); err != nil {
		return result, err
	}
	return result, nil
}

func (p publisher) releaseIfExists(ctx context.Context, version string) (*releaseMetadata, error) {
	// Enumerate via a successful API response so authentication/network errors are never absence.
	data, err := p.run(ctx, "gh", "api", "repos/"+repository+"/releases?per_page=100", "--paginate", "--slurp")
	if err != nil {
		return nil, err
	}
	var pages [][]struct {
		Tag string `json:"tag_name"`
	}
	if err := json.Unmarshal(data, &pages); err != nil {
		return nil, err
	}
	for _, page := range pages {
		for _, r := range page {
			if r.Tag == version {
				result, err := p.release(ctx, version)
				return &result, err
			}
		}
	}
	return nil, nil
}

func (p publisher) verifyEvidence(ctx context.Context, version, source string) error {
	data, err := p.run(ctx, "gh", "release", "download", version, "--repo", repository, "--pattern", "release-evidence.json", "--output", "-")
	if err != nil {
		return err
	}
	var evidence ciEvidence
	if err := json.Unmarshal(data, &evidence); err != nil {
		return err
	}
	if evidence.Source != source || len(evidence.Runs) != 2 {
		return errors.New("release evidence does not match source and required workflows")
	}
	for _, workflow := range []string{"test.yml", "codeql.yml"} {
		run, ok := evidence.Runs[workflow]
		if !ok || run.SHA != source || run.Branch != "main" || run.Event != "push" || run.Status != "completed" || run.Conclusion != "success" || run.ID <= 0 || run.Attempt <= 0 {
			return fmt.Errorf("invalid %s release evidence", workflow)
		}
	}
	return nil
}

func validatePublished(r releaseMetadata, version string) error {
	if r.Tag != version || r.Draft || r.Prerelease != strings.Contains(version, "-alpha.") || len(strings.Fields(r.Body)) < 20 {
		return errors.New("release metadata is incomplete or inconsistent")
	}
	return nil
}

// pollPause keeps cancellation responsive while CI completes.
func pollPause(ctx context.Context) error {
	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
