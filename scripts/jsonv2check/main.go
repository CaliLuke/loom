// Command jsonv2check rejects custom JSON marshalers that discard nested
// deterministic-ordering requirements.
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type finding struct {
	Path   string
	Line   int
	Column int
}

func main() {
	root, err := repositoryRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "JSON v2 custom-marshaler lint:", err)
		os.Exit(1)
	}
	findings, err := scanRepository(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "JSON v2 custom-marshaler lint:", err)
		os.Exit(1)
	}
	for _, finding := range findings {
		fmt.Fprintf(
			os.Stderr,
			"%s:%d:%d: json.Marshal inside MarshalJSON must include json.Deterministic(true)\n",
			finding.Path,
			finding.Line,
			finding.Column,
		)
	}
	if len(findings) > 0 {
		fmt.Fprintln(os.Stderr, "Pass explicit JSON v2 options or use an option-aware marshaler.")
		os.Exit(1)
	}
}

func repositoryRoot() (string, error) {
	command := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("locate repository root: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func scanRepository(root string) ([]finding, error) {
	paths, err := repositorySourcePaths(root)
	if err != nil {
		return nil, err
	}
	var findings []finding
	for _, path := range paths {
		data, readErr := os.ReadFile(filepath.Join(root, path))
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", path, readErr)
		}
		fileFindings, inspectErr := inspectSource(path, data)
		if inspectErr != nil {
			return nil, inspectErr
		}
		findings = append(findings, fileFindings...)
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		return findings[i].Column < findings[j].Column
	})
	return findings, nil
}

func repositorySourcePaths(root string) ([]string, error) {
	command := exec.Command(
		"git",
		"ls-files",
		"-z",
		"--cached",
		"--others",
		"--exclude-standard",
		"--",
		"*.go",
		"*.golden",
	)
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("list repository sources: %w", err)
	}
	rawPaths := bytes.Split(output, []byte{0})
	paths := make([]string, 0, len(rawPaths))
	for _, rawPath := range rawPaths {
		if len(rawPath) > 0 {
			paths = append(paths, string(rawPath))
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func inspectSource(path string, data []byte) ([]finding, error) {
	if !bytes.Contains(data, []byte("MarshalJSON")) {
		return nil, nil
	}
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, path, data, parser.SkipObjectResolution)
	lineOffset := 0
	if err != nil && filepath.Ext(path) == ".golden" {
		const generatedPackage = "package generated\n"
		parsed, err = parser.ParseFile(
			fset,
			path,
			append([]byte(generatedPackage), data...),
			parser.SkipObjectResolution,
		)
		lineOffset = 1
	}
	if err != nil {
		return nil, fmt.Errorf("parse custom-marshaler source %s: %w", path, err)
	}
	jsonAliases := jsonV2Aliases(parsed)
	if filepath.Ext(path) == ".golden" && bytes.Contains(data, []byte("json.Marshal")) {
		jsonAliases["json"] = true
	}
	if len(jsonAliases) == 0 {
		return nil, nil
	}

	var findings []finding
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv == nil || function.Name.Name != "MarshalJSON" || function.Body == nil {
			continue
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "Marshal" {
				return true
			}
			identifier, ok := selector.X.(*ast.Ident)
			if !ok || !jsonAliases[identifier.Name] {
				return true
			}
			if hasDeterministicOption(call, jsonAliases) {
				return true
			}
			position := fset.Position(call.Pos())
			findings = append(findings, finding{
				Path:   path,
				Line:   position.Line - lineOffset,
				Column: position.Column,
			})
			return true
		})
	}
	return findings, nil
}

func hasDeterministicOption(call *ast.CallExpr, jsonAliases map[string]bool) bool {
	for _, argument := range call.Args[1:] {
		optionCall, ok := argument.(*ast.CallExpr)
		if !ok || len(optionCall.Args) != 1 {
			continue
		}
		selector, ok := optionCall.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Deterministic" {
			continue
		}
		identifier, ok := selector.X.(*ast.Ident)
		if !ok || !jsonAliases[identifier.Name] {
			continue
		}
		value, ok := optionCall.Args[0].(*ast.Ident)
		if ok && value.Name == "true" {
			return true
		}
	}
	return false
}

func jsonV2Aliases(file *ast.File) map[string]bool {
	aliases := make(map[string]bool)
	for _, imported := range file.Imports {
		path, err := strconv.Unquote(imported.Path.Value)
		if err != nil || path != "encoding/json/v2" {
			continue
		}
		name := "json"
		if imported.Name != nil {
			name = imported.Name.Name
		}
		if name != "." && name != "_" {
			aliases[name] = true
		}
	}
	return aliases
}
