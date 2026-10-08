package release

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (p publisher) publishRelease(ctx context.Context, source, version string, tags []tag, existing *releaseMetadata, evidence ciEvidence) (resultErr error) {
	notes, err := p.notes(ctx, version, source, tags)
	if err != nil {
		return err
	}
	temp, err := os.MkdirTemp("", "loom-publication-")
	if err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(temp); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("remove publication files: %w", err))
		}
	}()
	if err := writeReleaseFiles(temp, notes, evidence); err != nil {
		return err
	}
	notesPath := filepath.Join(temp, "notes.md")
	evidencePath := filepath.Join(temp, "release-evidence.json")
	if findTag(tags, version) == "" {
		if _, err := p.git(ctx, "tag", "-a", version, source, "-m", "Release "+version); err != nil {
			return err
		}
		if _, err := p.git(ctx, "push", "origin", "refs/tags/"+version); err != nil {
			return err
		}
	}
	if existing == nil {
		if _, err := p.run(ctx, "gh", "release", "create", version, "--repo", repository, "--verify-tag", "--draft", "--title", version, "--notes-file", notesPath); err != nil {
			return err
		}
	}
	if _, err := p.run(ctx, "gh", "release", "upload", version, evidencePath, "--repo", repository, "--clobber"); err != nil {
		return err
	}
	prerelease := p.config.Mode != "promote"
	if _, err := p.run(ctx, "gh", "release", "edit", version, "--repo", repository, "--notes-file", notesPath, "--draft=false", fmt.Sprintf("--prerelease=%t", prerelease), fmt.Sprintf("--latest=%t", !prerelease)); err != nil {
		return err
	}
	published, err := p.release(ctx, version)
	if err != nil {
		return err
	}
	if err := p.verifyPublication(ctx, published, version, source); err != nil {
		return err
	}
	fmt.Printf("Published https://github.com/%s/releases/tag/%s at %s\n", repository, version, source)
	return nil
}

func writeReleaseFiles(temp, notes string, evidence ciEvidence) error {
	notesPath := filepath.Join(temp, "notes.md")
	if err := os.WriteFile(notesPath, []byte(notes), 0o600); err != nil {
		return err
	}
	evidencePath := filepath.Join(temp, "release-evidence.json")
	encoded, err := json.Marshal(evidence, json.Deterministic(true))
	if err != nil {
		return err
	}
	if err := os.WriteFile(evidencePath, encoded, 0o600); err != nil {
		return err
	}
	return nil
}

func (p publisher) verifyPublication(ctx context.Context, published releaseMetadata, version, source string) error {
	if err := validatePublished(published, version); err != nil {
		return err
	}
	if err := p.verifyEvidence(ctx, version, source); err != nil {
		return err
	}
	refs, err := p.git(ctx, "ls-remote", "origin", "refs/tags/"+version+"^{}")
	if err != nil {
		return err
	}
	if fields := strings.Fields(refs); len(fields) != 2 || fields[0] != source {
		return errors.New("published tag source mismatch")
	}
	return nil
}
