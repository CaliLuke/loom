package release

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"

	"golang.org/x/mod/semver"
)

func (p publisher) selectSource(ctx context.Context, tags []tag) (string, string, error) {
	source := p.config.Source
	if p.config.Mode == "promote" {
		if err := validatePromotion(p.config.Alpha, p.config.Version); err != nil {
			return "", "", err
		}
		source = findTag(tags, p.config.Alpha)
		if source == "" {
			return "", "", errors.New("promotion alpha tag does not exist")
		}
		metadata, err := p.release(ctx, p.config.Alpha)
		if err != nil {
			return "", "", err
		}
		if metadata.Tag != p.config.Alpha || metadata.Draft || !metadata.Prerelease {
			return "", "", errors.New("promotion requires a published alpha release")
		}
	}
	if !shaPattern.MatchString(source) {
		return "", "", errors.New("release requires an exact full commit SHA")
	}
	if _, err := p.git(ctx, "merge-base", "--is-ancestor", source, "origin/main"); err != nil {
		return "", "", fmt.Errorf("source is not on main: %w", err)
	}
	// This marker also rejects historical commits whose hardcoded version cannot be promoted.
	trainData, err := p.git(ctx, "show", source+":internal/release/train.json")
	if err != nil {
		return "", "", err
	}
	var train struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal([]byte(trainData), &train); err != nil {
		return "", "", fmt.Errorf("read release train: %w", err)
	}
	return source, train.Version, nil
}

func (p publisher) validateTarget(ctx context.Context, source, version string, tags []tag) error {
	if previous := findTag(tags, version); previous != "" && previous != source {
		return errors.New("release tag already identifies another source")
	}
	if findTag(tags, version) != "" {
		kind, err := p.git(ctx, "cat-file", "-t", "refs/tags/"+version)
		if err != nil {
			return err
		}
		if kind != "tag" {
			return errors.New("release tag must be annotated")
		}
		return nil
	}
	if p.config.Mode != "promote" {
		for _, previous := range tags {
			if _, err := p.git(ctx, "merge-base", "--is-ancestor", previous.sha, source); err != nil {
				return fmt.Errorf("alpha source must include previous release %s: %w", previous.version, err)
			}
		}
	}
	// An older workflow retry must not release behind a newer published source.
	for _, t := range tags {
		if t.sha == source {
			continue
		}
		if semver.Compare(t.version, version) > 0 {
			return fmt.Errorf("newer release %s already exists", t.version)
		}
	}

	return nil
}
