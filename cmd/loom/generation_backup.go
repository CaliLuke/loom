package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type (
	// generationBackup protects one original until restoration or complete commit.
	// Disposable publication candidates are owned separately by the transaction.
	generationBackup struct {
		path        string
		destination string
		tree        bool
		discardable bool
		released    bool
	}
)

func (t *generationTransaction) preserveOriginal(destination string) (*generationBackup, error) {
	info, err := os.Lstat(destination)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect original output %s: %w", destination, err)
	}
	path, err := reserveRenamePath(filepath.Dir(destination))
	if err != nil {
		return nil, err
	}
	if err := os.Rename(destination, path); err != nil {
		return nil, fmt.Errorf("preserve original output %s: %w", destination, err)
	}
	backup := &generationBackup{path: path, destination: destination, tree: info.IsDir()}
	t.backups = append(t.backups, backup)
	return backup, nil
}

// discardOriginals is called only after every new output has been installed.
func (t *generationTransaction) discardOriginals() error {
	for _, backup := range t.backups {
		backup.discardable = true
	}
	return t.cleanupBackups()
}

func (t *generationTransaction) cleanupBackups() error {
	var result error
	for _, backup := range t.backups {
		result = errors.Join(result, backup.cleanup())
	}
	return result
}

func (b *generationBackup) restore() error {
	if b == nil || b.released {
		return nil
	}
	if b.discardable {
		return fmt.Errorf("cannot restore original %s after installation committed", b.destination)
	}
	if err := os.Rename(b.path, b.destination); err != nil {
		return fmt.Errorf("restore original %s from preserved backup %s: %w", b.destination, b.path, err)
	}
	b.released = true
	return nil
}

func (b *generationBackup) cleanup() error {
	if b.released {
		return nil
	}
	if !b.discardable {
		return fmt.Errorf("original output %s is preserved at %s; manual recovery required", b.destination, b.path)
	}
	var err error
	if b.tree {
		err = os.RemoveAll(b.path)
	} else {
		err = os.Remove(b.path)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove committed backup %s for %s: %w", b.path, b.destination, err)
	}
	b.released = true
	return nil
}
