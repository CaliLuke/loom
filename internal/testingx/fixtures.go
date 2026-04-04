package testingx

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// RepoRoot returns the absolute repository root for the current checkout.
func RepoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

// CopyTree copies a directory tree from src to dst.
func CopyTree(src string, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("rel path %s: %w", path, err)
		}
		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0o750)
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat %s: %w", path, err)
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		return copyFile(path, target, info.Mode())
	})
}

// RunCmd runs a command in dir and returns its combined output.
func RunCmd(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GO111MODULE=on", "GOWORK=off")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%s %v failed: %w\n%s", name, args, err, output)
	}
	return string(output), nil
}

// PinLocalReplace rewrites the copied fixture go.mod to point the github.com/CaliLuke/loom
// replace directive at repoRoot.
func PinLocalReplace(dir string, repoRoot string) error {
	if _, err := RunCmd(dir, "go", "mod", "edit", fmt.Sprintf("-replace=github.com/CaliLuke/loom=%s", repoRoot)); err != nil {
		return err
	}
	_, err := RunCmd(dir, "go", "mod", "tidy")
	return err
}

func copyFile(src string, dst string, mode fs.FileMode) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %s: %w", src, err)
	}
	defer func() {
		if closeErr := in.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close %s: %w", src, closeErr)
		}
	}()

	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(dst), err)
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode.Perm())
	if err != nil {
		return fmt.Errorf("create %s: %w", dst, err)
	}
	defer func() {
		if closeErr := out.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close %s: %w", dst, closeErr)
		}
	}()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy %s -> %s: %w", src, dst, err)
	}

	return nil
}
