package main

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// preparePublicationFile copies a fully rendered external artifact into its
// destination filesystem before any live publication. The transaction owns the
// candidate immediately, including when copying, chmod or closing fails.
func (t *generationTransaction) preparePublicationFile(source, destinationParent string) (path string, err error) {
	input, err := os.Open(source)
	if err != nil {
		return "", fmt.Errorf("open staged output %s: %w", source, err)
	}
	defer func() {
		if closeErr := input.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close staged output %s: %w", source, closeErr))
		}
	}()
	info, err := input.Stat()
	if err != nil {
		return "", fmt.Errorf("inspect staged output %s: %w", source, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("staged output %s is not a regular file", source)
	}
	output, err := os.CreateTemp(destinationParent, ".loom-gen-file-")
	if err != nil {
		return "", fmt.Errorf("prepare output in %s: %w", destinationParent, err)
	}
	path = output.Name()
	t.externalCandidates = append(t.externalCandidates, path)
	defer func() {
		if closeErr := output.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close prepared output %s: %w", path, closeErr))
		}
	}()
	if _, err := io.Copy(output, input); err != nil {
		return path, fmt.Errorf("copy staged output %s: %w", source, err)
	}
	if err := output.Chmod(info.Mode().Perm()); err != nil {
		return path, fmt.Errorf("set prepared output mode %s: %w", path, err)
	}
	return path, nil
}
