// Package artifactpolicy owns admission of generated artifact write policies.
package artifactpolicy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Validate rejects create-only files in the regenerated gen tree, including
// stable live directory aliases. Path is relative to dir. All case variants of
// the top-level gen name are reserved on every filesystem. Existing ancestors
// must remain stable between validation and installation; this does not defend
// against a process swapping directories concurrently.
func Validate(dir, path string, skipExist bool) error {
	if !skipExist {
		return nil
	}
	path = filepath.Clean(path)
	rejected := fmt.Errorf("create-only artifact %s must be outside gen/", path)
	if reservedGenerationPath(path) {
		return rejected
	}
	base, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	resolvedBase, err := resolvePath(base)
	if err != nil {
		return fmt.Errorf("resolve output directory %s: %w", base, err)
	}
	parent, err := resolvePath(filepath.Join(base, filepath.Dir(path)))
	if err != nil {
		return fmt.Errorf("resolve artifact parent %s: %w", path, err)
	}
	// Also reserve case variants reached through an alias when gen is absent.
	relative, err := filepath.Rel(resolvedBase, parent)
	if err == nil && reservedGenerationPath(relative) {
		return rejected
	}
	generated, err := resolvePath(filepath.Join(base, "gen"))
	if err != nil {
		return fmt.Errorf("resolve generation directory: %w", err)
	}
	relative, err = filepath.Rel(generated, parent)
	if err == nil && (relative == "." || filepath.IsLocal(relative)) {
		return rejected
	}
	// SameFile recognizes filesystem aliases whose spelling survives resolution,
	// such as case aliases and directory mounts exposing the same inode.
	generatedInfo, err := os.Stat(generated)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect generation directory %s: %w", generated, err)
	}
	baseInfo, err := os.Stat(resolvedBase)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect output directory %s: %w", resolvedBase, err)
	}
	child := ""
	for ancestor := parent; ; ancestor = filepath.Dir(ancestor) {
		info, err := os.Stat(ancestor)
		if err == nil {
			if generatedInfo != nil && os.SameFile(generatedInfo, info) {
				return rejected
			}
			// Even absent gen names are reserved when an alias reaches the same
			// output root through a different filesystem spelling.
			if baseInfo != nil && os.SameFile(baseInfo, info) && reservedGenerationPath(child) {
				return rejected
			}
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect artifact ancestor %s: %w", ancestor, err)
		}
		if filepath.Dir(ancestor) == ancestor {
			break
		}
		child = filepath.Base(ancestor)
	}
	return nil
}

func reservedGenerationPath(path string) bool {
	first, _, _ := strings.Cut(filepath.Clean(path), string(filepath.Separator))
	return strings.EqualFold(first, "gen")
}

// resolvePath follows existing aliases while retaining absent suffixes. In
// particular a dangling alias to gen/missing must resolve before missing child
// directories are created. EvalSymlinks alone cannot resolve those paths.
func resolvePath(path string) (string, error) {
	volume := filepath.VolumeName(path)
	resolved := volume + string(filepath.Separator)
	remaining := strings.Split(strings.TrimPrefix(path, volume), string(filepath.Separator))
	links := 0
	for len(remaining) > 0 {
		part := remaining[0]
		remaining = remaining[1:]
		switch part {
		case "", ".":
			continue
		case "..":
			resolved = filepath.Dir(resolved)
			continue
		}
		candidate := filepath.Join(resolved, part)
		info, err := os.Lstat(candidate)
		if errors.Is(err, os.ErrNotExist) {
			resolved = candidate
			continue
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			resolved = candidate
			continue
		}
		links++
		if links > 255 {
			return "", fmt.Errorf("too many symbolic links resolving %s", path)
		}
		target, err := os.Readlink(candidate)
		if err != nil {
			return "", err
		}
		if filepath.IsAbs(target) {
			volume = filepath.VolumeName(target)
			resolved = volume + string(filepath.Separator)
			target = strings.TrimPrefix(target, volume)
		} else if filepath.VolumeName(target) != "" {
			return "", fmt.Errorf("unsupported volume-relative symbolic link %s", candidate)
		}
		// Do not clean target before following its components: link/.. must
		// traverse the link first, even when the remaining descendants are absent.
		remaining = append(strings.Split(target, string(filepath.Separator)), remaining...)
	}
	return resolved, nil
}
