package valuecontract

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type sourceFile struct {
	// SHA256 identifies regular-file bytes; symlinks instead retain their target.
	SHA256 string `json:"sha256,omitempty"`
	// Mode preserves permission and file-type bits.
	Mode uint32 `json:"mode"`
	// Link is the original relative target of an internal symlink.
	Link string `json:"link,omitempty"`
}

func gitSourceFiles(t *testing.T, root string) []string {
	t.Helper()
	result := runPhase(t.Context(), root, nil, "source-files", "git", "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	require.Zero(t, result.Exit, result.Output)
	var paths []string
	for _, path := range strings.Split(result.Output, "\x00") {
		// This absolute, agent-only pointer is not a generator input. All actual
		// .agents source files remain captured. Never follow it outside the snapshot.
		if path == "" || path == ".claude" || strings.HasPrefix(path, ".claude/") {
			continue
		}
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(path))); errors.Is(err, os.ErrNotExist) {
			continue
		} else {
			require.NoError(t, err)
		}
		paths = append(paths, path)
	}
	slices.Sort(paths)
	return slices.Compact(paths)
}

func sourceInventory(root string, paths []string) (map[string]sourceFile, error) {
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	result := make(map[string]sourceFile, len(paths))
	for _, path := range paths {
		clean := filepath.Clean(filepath.FromSlash(path))
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("source path escapes root: %s", path)
		}
		absolute := filepath.Join(root, clean)
		info, err := os.Lstat(absolute)
		if err != nil {
			return nil, err
		}
		entry := sourceFile{Mode: uint32(info.Mode())}
		switch {
		case info.Mode().IsRegular():
			data, err := os.ReadFile(absolute)
			if err != nil {
				return nil, err
			}
			entry.SHA256 = digest(data)
		case info.Mode()&os.ModeSymlink != 0:
			link, err := os.Readlink(absolute)
			if err != nil {
				return nil, err
			}
			target, err := filepath.EvalSymlinks(absolute)
			if err != nil {
				return nil, fmt.Errorf("resolve source link %s: %w", path, err)
			}
			relative, err := filepath.Rel(canonicalRoot, target)
			if err != nil || filepath.IsAbs(link) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				return nil, fmt.Errorf("unsupported source link %s -> %s", path, link)
			}
			entry.Link = link
		default:
			return nil, fmt.Errorf("unsupported source file %s: %s", path, info.Mode())
		}
		result[filepath.ToSlash(clean)] = entry
	}
	return result, nil
}

func materializeSource(root, destination string, inventory map[string]sourceFile) error {
	if err := os.MkdirAll(destination, 0700); err != nil {
		return err
	}
	for path, entry := range inventory {
		target := filepath.Join(destination, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		if entry.Link != "" {
			if err := os.Symlink(entry.Link, target); err != nil {
				return err
			}
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			return err
		}
		if digest(data) != entry.SHA256 {
			return fmt.Errorf("source changed during capture: %s", path)
		}
		if err := os.WriteFile(target, data, os.FileMode(entry.Mode).Perm()); err != nil {
			return err
		}
		if err := os.Chmod(target, os.FileMode(entry.Mode).Perm()); err != nil {
			return err
		}
	}
	return verifySourceSnapshot(destination, inventory)
}

func verifySourceSnapshot(root string, want map[string]sourceFile) error {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return err
	}
	got, err := sourceInventory(root, paths)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(want, got) {
		return errors.New("source snapshot content or mode changed")
	}
	return nil
}

func captureGitSource(t *testing.T, root, destination string) map[string]sourceFile {
	t.Helper()
	before, err := sourceInventory(root, gitSourceFiles(t, root))
	require.NoError(t, err)
	require.NoError(t, materializeSource(root, destination, before))
	after, err := sourceInventory(root, gitSourceFiles(t, root))
	require.NoError(t, err)
	require.Equal(t, before, after, "source changed during capture; retry with stable inputs")
	writeJSON(t, destination+"-files.json", before)
	return before
}

func copyCommonSpecimen(source, destination, originalImport, commonImport string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0700)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unsupported specimen entry %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".go") {
			data, err = rewriteSpecimenImports(data, originalImport, commonImport)
			if err != nil {
				return fmt.Errorf("capture specimen %s: %w", relative, err)
			}
		}
		return os.WriteFile(target, data, 0600)
	})
}

func rewriteSpecimenImports(source []byte, oldPrefix, newPrefix string) ([]byte, error) {
	files := token.NewFileSet()
	parsed, err := parser.ParseFile(files, "specimen.go", source, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	type edit struct {
		start, end  int
		replacement string
	}
	var edits []edit
	add := func(expression ast.Expr) {
		literal, ok := expression.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil {
			return
		}
		if value == oldPrefix || strings.HasPrefix(value, oldPrefix+"/") {
			edits = append(edits, edit{start: files.Position(literal.Pos()).Offset, end: files.Position(literal.End()).Offset, replacement: strconv.Quote(newPrefix + strings.TrimPrefix(value, oldPrefix))})
		}
	}
	ast.Inspect(parsed, func(node ast.Node) bool {
		switch actual := node.(type) {
		case *ast.ImportSpec:
			add(actual.Path)
		case *ast.CallExpr:
			name := ""
			switch function := actual.Fun.(type) {
			case *ast.Ident:
				name = function.Name
			case *ast.SelectorExpr:
				name = function.Sel.Name
			}
			if name != "Meta" || len(actual.Args) < 2 {
				return true
			}
			key, ok := actual.Args[0].(*ast.BasicLit)
			if !ok || key.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(key.Value)
			if err != nil {
				return true
			}
			switch value {
			case "struct:field:type":
				if len(actual.Args) >= 3 {
					add(actual.Args[2])
				}
			case "struct:pkg:path":
				add(actual.Args[1])
			}
		}
		return true
	})
	slices.SortFunc(edits, func(a, b edit) int {
		return b.start - a.start
	})
	for _, edit := range edits {
		source = append(source[:edit.start], append([]byte(edit.replacement), source[edit.end:]...)...)
	}
	return source, nil
}

func snapshotIdentity(inventory map[string]sourceFile) (string, error) {
	encoded, err := json.Marshal(inventory, json.Deterministic(true))
	if err != nil {
		return "", err
	}
	return digest(encoded), nil
}
