package vet

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/tools/go/packages"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/designfingerprint"
)

type generatedManifest struct {
	Version      string `json:"loom_version"`
	DesignDigest string `json:"design_digest"`
}

const loomHTTPImportPath = "github.com/CaliLuke/loom/http"

func analyzeModule(dir string) ([]Diagnostic, error) {
	return analyzeModuleWithDesign(dir, nil)
}

func analyzeModuleWithDesign(dir string, design *expr.RootExpr) ([]Diagnostic, error) {
	root, err := findModuleRoot(dir)
	if err != nil {
		return nil, err
	}
	routes, err := analyzeRoutesWithDesign(root, design)
	if err != nil {
		return nil, err
	}
	mounts, err := analyzeConfiguredHTTPMounts(root, design)
	if err != nil {
		return nil, err
	}
	versions, err := analyzeGeneratedManifests(root, design)
	if err != nil {
		return nil, err
	}
	routes = append(routes, mounts...)
	return append(routes, versions...), nil
}

func findModuleRoot(dir string) (string, error) {
	current, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve module directory: %w", err)
	}
	for {
		info, statErr := os.Stat(filepath.Join(current, "go.mod"))
		switch {
		case statErr == nil && info.Mode().IsRegular():
			return current, nil
		case statErr == nil:
			return "", fmt.Errorf("module file %s is not a regular file", filepath.Join(current, "go.mod"))
		case !os.IsNotExist(statErr):
			return "", fmt.Errorf("inspect module directory: %w", statErr)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("no go.mod found from %s", dir)
		}
		current = parent
	}
}

func analyzeRoutesWithDesign(root string, design *expr.RootExpr) ([]Diagnostic, error) {
	loaded, err := packages.Load(&packages.Config{
		Dir: root,
		Mode: packages.NeedName |
			packages.NeedCompiledGoFiles |
			packages.NeedModule |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo,
		ParseFile: func(fileSet *token.FileSet, filename string, source []byte) (*ast.File, error) {
			return parser.ParseFile(fileSet, filename, source, parser.ParseComments)
		},
	}, "./...")
	if err != nil {
		return nil, fmt.Errorf("load Go module for Loom route analysis: %w", err)
	}

	var diagnostics []Diagnostic
	var manualRoutes []manualRoute
	var firstPackageError error
	analyzablePackages := 0
	for _, pkg := range loaded {
		packageDiagnostics, packageRoutes, analyzeErr := analyzePackageRoutes(root, pkg)
		if analyzeErr != nil {
			return nil, analyzeErr
		}
		diagnostics = append(diagnostics, packageDiagnostics...)
		manualRoutes = append(manualRoutes, packageRoutes...)
		if packageSyntaxUsable(pkg) {
			analyzablePackages++
		}
		if len(pkg.Errors) > 0 && firstPackageError == nil {
			firstPackageError = pkg.Errors[0]
		}
	}
	if analyzablePackages == 0 && firstPackageError != nil && len(diagnostics) == 0 {
		return nil, fmt.Errorf("load Go module for Loom route analysis: %w", firstPackageError)
	}
	diagnostics = append(diagnostics, duplicateManualRouteDiagnostics(manualRoutes)...)
	diagnostics = append(diagnostics, designRouteConflictDiagnostics(manualRoutes, design)...)
	return diagnostics, nil
}

func analyzePackageRoutes(root string, pkg *packages.Package) ([]Diagnostic, []manualRoute, error) {
	var diagnostics []Diagnostic
	var routes []manualRoute
	if incomplete := incompleteAnalysisDiagnostic(root, pkg); incomplete != nil {
		diagnostics = append(diagnostics, *incomplete)
	}
	for index, file := range pkg.Syntax {
		if index >= len(pkg.CompiledGoFiles) {
			continue
		}
		path := pkg.CompiledGoFiles[index]
		source, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, fmt.Errorf("read %s: %w", path, err)
		}
		if generatedByLoom(source) {
			continue
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, registration, registered := typedLoomMuxRegistration(pkg, node)
			if !registered {
				return true
			}
			position := pkg.Fset.Position(call.Pos())
			relative, err := filepath.Rel(root, path)
			if err != nil {
				relative = path
			}
			location := Location{Path: filepath.ToSlash(relative), Line: position.Line, Column: position.Column}
			if !sourceSuppressed(pkg.Fset, file, call, RuleRouteOutsideDesign) {
				diagnostics = append(diagnostics, Diagnostic{
					Rule:     RuleRouteOutsideDesign,
					Severity: SeverityError,
					Message:  unmanagedRouteMessage(pkg, call, registration),
					Location: location,
				})
			}
			method := stringConstantArgument(pkg, call, registration.methodArg)
			path := stringConstantArgument(pkg, call, registration.pathArg)
			if method != "" && path != "" {
				routes = append(routes, manualRoute{
					method:              method,
					path:                path,
					location:            location,
					duplicateSuppressed: sourceSuppressed(pkg.Fset, file, call, RuleDuplicateRouteRegistration),
					conflictSuppressed:  sourceSuppressed(pkg.Fset, file, call, RuleRouteConflictWithDesign),
				})
			}
			return true
		})
	}
	return diagnostics, routes, nil
}

func packageSyntaxUsable(pkg *packages.Package) bool {
	return len(pkg.Syntax) > 0 && pkg.TypesInfo != nil
}

func incompleteAnalysisDiagnostic(root string, pkg *packages.Package) *Diagnostic {
	if !packageBelongsToModule(root, pkg) {
		return nil
	}
	loadError, incomplete := incompletePackageError(pkg)
	if !incomplete {
		return nil
	}
	location := packageErrorLocation(root, pkg, loadError)
	return &Diagnostic{
		Rule:     RuleVetAnalysisIncomplete,
		Severity: SeverityError,
		Message:  fmt.Sprintf("package %q could not be fully analyzed: %s", pkg.PkgPath, loadError.Msg),
		Location: location,
	}
}

func packageBelongsToModule(root string, pkg *packages.Package) bool {
	if pkg.Module == nil || pkg.Module.Dir == "" {
		return false
	}
	moduleDir, err := filepath.Abs(pkg.Module.Dir)
	if err != nil {
		return false
	}
	return filepath.Clean(moduleDir) == filepath.Clean(root)
}

func incompletePackageError(pkg *packages.Package) (packages.Error, bool) {
	for _, loadError := range pkg.Errors {
		if loadError.Kind == packages.ParseError {
			return loadError, true
		}
	}
	if len(pkg.CompiledGoFiles) == len(pkg.Syntax) && pkg.Types != nil && pkg.TypesInfo != nil {
		return packages.Error{}, false
	}
	if len(pkg.Errors) > 0 {
		return pkg.Errors[0], true
	}
	return packages.Error{}, false
}

func packageErrorLocation(root string, pkg *packages.Package, loadError packages.Error) Location {
	path, line, column := splitPackageErrorPosition(loadError.Pos)
	if path == "" && len(pkg.CompiledGoFiles) > 0 {
		path = pkg.CompiledGoFiles[0]
	}
	if path == "" {
		return Location{Path: pkg.PkgPath}
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	if relative, err := filepath.Rel(root, path); err == nil {
		path = relative
	}
	return Location{Path: filepath.ToSlash(path), Line: line, Column: column}
}

func splitPackageErrorPosition(position string) (string, int, int) {
	if position == "" || position == "-" {
		return "", 0, 0
	}
	path, column := position, 0
	lastColon := strings.LastIndexByte(path, ':')
	if lastColon < 0 {
		return path, 0, 0
	}
	value, err := strconv.Atoi(path[lastColon+1:])
	if err != nil {
		return path, 0, 0
	}
	path = path[:lastColon]
	column = value
	lastColon = strings.LastIndexByte(path, ':')
	if lastColon < 0 {
		return path, column, 0
	}
	value, err = strconv.Atoi(path[lastColon+1:])
	if err != nil {
		return path, column, 0
	}
	return path[:lastColon], value, column
}

func skipDirectory(name string) bool {
	switch name {
	case ".git", ".hg", ".svn", "node_modules", "vendor":
		return true
	default:
		return strings.HasPrefix(name, ".")
	}
}

func skipModuleDirectory(root, path, name string) (bool, error) {
	if path == root {
		return false, nil
	}
	if skipDirectory(name) {
		return true, nil
	}
	info, err := os.Stat(filepath.Join(path, "go.mod"))
	switch {
	case err == nil:
		return info.Mode().IsRegular(), nil
	case os.IsNotExist(err):
		return false, nil
	default:
		return false, fmt.Errorf("inspect nested module %s: %w", path, err)
	}
}

func generatedByLoom(source []byte) bool {
	prefix := source
	if len(prefix) > 512 {
		prefix = prefix[:512]
	}
	return strings.Contains(string(prefix), "Code generated by Loom, DO NOT EDIT.")
}

type routeRegistration struct {
	methodArg int
	pathArg   int
}

func typedLoomMuxRegistration(pkg *packages.Package, node ast.Node) (*ast.CallExpr, routeRegistration, bool) {
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return nil, routeRegistration{}, false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, routeRegistration{}, false
	}
	switch selector.Sel.Name {
	case "Handle":
		selection := pkg.TypesInfo.Selections[selector]
		if selection == nil {
			return nil, routeRegistration{}, false
		}
		method := selection.Obj()
		if method.Pkg() == nil || method.Pkg().Path() != loomHTTPImportPath {
			return nil, routeRegistration{}, false
		}
		return call, routeRegistration{methodArg: 0, pathArg: 1}, true
	case "MountHandler":
		function, ok := pkg.TypesInfo.Uses[selector.Sel].(*types.Func)
		if !ok || function.Pkg() == nil || function.Pkg().Path() != loomHTTPImportPath {
			return nil, routeRegistration{}, false
		}
		return call, routeRegistration{methodArg: 1, pathArg: 2}, true
	default:
		return nil, routeRegistration{}, false
	}
}

func unmanagedRouteMessage(pkg *packages.Package, call *ast.CallExpr, registration routeRegistration) string {
	method := stringConstantArgument(pkg, call, registration.methodArg)
	path := stringConstantArgument(pkg, call, registration.pathArg)
	switch {
	case method != "" && path != "":
		return fmt.Sprintf("route %s %s is registered directly on a Loom mux; declare and mount it through the design", method, path)
	case path != "":
		return fmt.Sprintf("route %s is registered directly on a Loom mux; declare and mount it through the design", path)
	default:
		return "route is registered directly on a Loom mux; declare and mount it through the design"
	}
}

func stringConstantArgument(pkg *packages.Package, call *ast.CallExpr, index int) string {
	if index >= len(call.Args) {
		return ""
	}
	if value, ok := pkg.TypesInfo.Types[call.Args[index]]; ok && value.Value != nil && value.Value.Kind() == constant.String {
		return constant.StringVal(value.Value)
	}
	literal, ok := call.Args[index].(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return ""
	}
	value, err := strconv.Unquote(literal.Value)
	if err != nil {
		return ""
	}
	return value
}

func sourceSuppressed(fileSet *token.FileSet, file *ast.File, node ast.Node, rule string) bool {
	line := fileSet.Position(node.Pos()).Line
	for _, comments := range file.Comments {
		endLine := fileSet.Position(comments.End()).Line
		if endLine != line && endLine != line-1 {
			continue
		}
		for _, comment := range comments.List {
			text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
			if strings.Contains(text, "loom:vet ignore "+rule) || strings.Contains(text, "loom:vet ignore all") {
				return true
			}
		}
	}
	return false
}

func analyzeGeneratedVersions(root string) ([]Diagnostic, error) {
	return analyzeGeneratedManifests(root, nil)
}

func analyzeGeneratedManifests(root string, design *expr.RootExpr) ([]Diagnostic, error) {
	moduleData, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil, fmt.Errorf("read go.mod: %w", err)
	}
	module, err := modfile.Parse("go.mod", moduleData, nil)
	if err != nil {
		return nil, fmt.Errorf("parse go.mod: %w", err)
	}
	requiredVersion, comparable := effectiveLoomVersion(module)

	var diagnostics []Diagnostic
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			skip, err := skipModuleDirectory(root, path, entry.Name())
			if err != nil {
				return err
			}
			if skip {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() != "loom.json" {
			return nil
		}
		manifestDiagnostics, inspectErr := inspectGeneratedManifest(
			module,
			root,
			path,
			requiredVersion,
			comparable,
			design,
		)
		if inspectErr != nil {
			return inspectErr
		}
		diagnostics = append(diagnostics, manifestDiagnostics...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("inspect generated Loom manifests: %w", err)
	}
	return diagnostics, nil
}

func inspectGeneratedManifest(
	moduleFile *modfile.File,
	moduleRoot,
	manifestPath,
	requiredVersion string,
	comparable bool,
	design *expr.RootExpr,
) ([]Diagnostic, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	var manifest generatedManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse %s: %w", manifestPath, err)
	}
	location := generatedManifestLocation(moduleRoot, manifestPath)
	var diagnostics []Diagnostic
	if comparable && manifest.Version != "" && manifest.Version != requiredVersion {
		diagnostics = append(diagnostics, Diagnostic{
			Rule:     RuleGeneratedVersionSkew,
			Severity: SeverityError,
			Message:  fmt.Sprintf("generated with Loom %s but go.mod requires %s; run loom gen", manifest.Version, requiredVersion),
			Location: location,
		})
	}
	designDiagnostic, err := generatedDesignDiagnostic(moduleFile, moduleRoot, manifestPath, manifest, design, location)
	if err != nil {
		return nil, err
	}
	if designDiagnostic != nil {
		diagnostics = append(diagnostics, *designDiagnostic)
	}
	return diagnostics, nil
}

func generatedManifestLocation(moduleRoot, manifestPath string) Location {
	relative, err := filepath.Rel(moduleRoot, manifestPath)
	if err != nil {
		relative = manifestPath
	}
	return Location{Path: filepath.ToSlash(relative)}
}

func generatedDesignDiagnostic(
	moduleFile *modfile.File,
	moduleRoot,
	manifestPath string,
	manifest generatedManifest,
	design *expr.RootExpr,
	location Location,
) (*Diagnostic, error) {
	if design == nil || design.API == nil || suppressed(design.API.Meta, RuleGeneratedDesignSkew) {
		return nil, nil
	}
	genpkg, err := generatedPackagePath(moduleFile, moduleRoot, manifestPath)
	if err != nil {
		return nil, err
	}
	currentDigest, err := designfingerprint.Digest(design, "gen", genpkg, codegen.DesignVersion)
	if err != nil {
		return nil, err
	}
	message := ""
	if manifest.DesignDigest == "" {
		message = "generated manifest does not record a design digest; run loom gen"
	} else if manifest.DesignDigest != currentDigest {
		message = "generated output does not match the evaluated design; run loom gen"
	}
	if message == "" {
		return nil, nil
	}
	return &Diagnostic{
		Rule:     RuleGeneratedDesignSkew,
		Severity: SeverityError,
		Message:  message,
		Location: location,
	}, nil
}

func generatedPackagePath(moduleFile *modfile.File, moduleRoot, manifestPath string) (string, error) {
	if moduleFile.Module == nil || moduleFile.Module.Mod.Path == "" {
		return "", errors.New("go.mod has no module path")
	}
	outputDir := filepath.Dir(filepath.Dir(manifestPath))
	relativeOutput, err := filepath.Rel(moduleRoot, outputDir)
	if err != nil {
		return "", fmt.Errorf("resolve generated output package: %w", err)
	}
	genpkg := moduleFile.Module.Mod.Path
	if relativeOutput != "." {
		genpkg = path.Join(genpkg, filepath.ToSlash(relativeOutput))
	}
	return path.Join(genpkg, codegen.Gendir), nil
}

func effectiveLoomVersion(moduleFile *modfile.File) (string, bool) {
	var requiredVersion string
	for _, requirement := range moduleFile.Require {
		if requirement.Mod.Path == "github.com/CaliLuke/loom" {
			requiredVersion = requirement.Mod.Version
			break
		}
	}
	if !comparableModuleVersion(requiredVersion) {
		return "", false
	}

	var wildcard *modfile.Replace
	for _, replacement := range moduleFile.Replace {
		if replacement.Old.Path != "github.com/CaliLuke/loom" {
			continue
		}
		if replacement.Old.Version == requiredVersion {
			return comparableReplacementVersion(replacement.New.Version)
		}
		if replacement.Old.Version == "" {
			wildcard = replacement
		}
	}
	if wildcard != nil {
		return comparableReplacementVersion(wildcard.New.Version)
	}
	return requiredVersion, true
}

func comparableReplacementVersion(version string) (string, bool) {
	if !comparableModuleVersion(version) {
		return "", false
	}
	return version, true
}

func comparableModuleVersion(version string) bool {
	return version != "" && !module.IsPseudoVersion(version)
}
