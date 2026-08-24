package vet

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

const loomHTTPImportPath = "github.com/CaliLuke/loom/http"

func analyzeModule(dir string) ([]Diagnostic, error) {
	root, err := findModuleRoot(dir)
	if err != nil {
		return nil, err
	}
	routes, err := analyzeRoutes(root)
	if err != nil {
		return nil, err
	}
	versions, err := analyzeGeneratedVersions(root)
	if err != nil {
		return nil, err
	}
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

func analyzeRoutes(root string) ([]Diagnostic, error) {
	var diagnostics []Diagnostic
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
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
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fileDiagnostics, err := analyzeRouteFile(root, path)
		if err != nil {
			return err
		}
		diagnostics = append(diagnostics, fileDiagnostics...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("analyze Loom mux routes: %w", err)
	}
	return diagnostics, nil
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

func analyzeRouteFile(root, path string) ([]Diagnostic, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if generatedByLoom(source) {
		return nil, nil
	}
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, source, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	loomAliases := loomHTTPAliases(file)
	if len(loomAliases) == 0 {
		return nil, nil
	}

	globalScope := newMuxScope(nil)
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		collectMuxValueSpecs(general.Specs, loomAliases, globalScope)
	}

	var diagnostics []Diagnostic
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		functionScope := newMuxScope(globalScope)
		collectMuxParameters(function.Type.Params, loomAliases, functionScope)
		visitor := &routeVisitor{
			root:        root,
			path:        path,
			fileSet:     fileSet,
			file:        file,
			aliases:     loomAliases,
			scope:       functionScope,
			diagnostics: &diagnostics,
		}
		ast.Walk(visitor, function.Body)
	}
	return diagnostics, nil
}

type muxScope struct {
	parent *muxScope
	values map[string]bool
}

func newMuxScope(parent *muxScope) *muxScope {
	return &muxScope{parent: parent, values: make(map[string]bool)}
}

func (scope *muxScope) declare(name string, mux bool) {
	scope.values[name] = mux
}

func (scope *muxScope) isMux(name string) bool {
	for current := scope; current != nil; current = current.parent {
		if mux, exists := current.values[name]; exists {
			return mux
		}
	}
	return false
}

type routeVisitor struct {
	root        string
	path        string
	fileSet     *token.FileSet
	file        *ast.File
	aliases     map[string]struct{}
	scope       *muxScope
	diagnostics *[]Diagnostic
}

func (visitor *routeVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	switch actual := node.(type) {
	case *ast.BlockStmt:
		return visitor.withScope(newMuxScope(visitor.scope))
	case *ast.FuncLit:
		child := newMuxScope(visitor.scope)
		collectMuxParameters(actual.Type.Params, visitor.aliases, child)
		return visitor.withScope(child)
	case *ast.IfStmt, *ast.ForStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
		return visitor.withScope(newMuxScope(visitor.scope))
	case *ast.RangeStmt:
		child := newMuxScope(visitor.scope)
		if actual.Tok == token.DEFINE {
			declareRangeNames(actual, child)
		}
		return visitor.withScope(child)
	case *ast.AssignStmt:
		collectMuxAssignments(actual, visitor.aliases, visitor.scope)
	case *ast.DeclStmt:
		if declaration, ok := actual.Decl.(*ast.GenDecl); ok {
			collectMuxValueSpecs(declaration.Specs, visitor.aliases, visitor.scope)
		}
	case *ast.CallExpr:
		visitor.reportMuxHandle(actual)
	}
	return visitor
}

func (visitor *routeVisitor) withScope(scope *muxScope) *routeVisitor {
	child := *visitor
	child.scope = scope
	return &child
}

func (visitor *routeVisitor) reportMuxHandle(call *ast.CallExpr) {
	if !isMuxHandleCall(call, visitor.aliases, visitor.scope) ||
		sourceSuppressed(visitor.fileSet, visitor.file, call, RuleRouteOutsideDesign) {
		return
	}
	position := visitor.fileSet.Position(call.Pos())
	relative, err := filepath.Rel(visitor.root, visitor.path)
	if err != nil {
		relative = visitor.path
	}
	*visitor.diagnostics = append(*visitor.diagnostics, Diagnostic{
		Rule:     RuleRouteOutsideDesign,
		Severity: SeverityError,
		Message:  unmanagedRouteMessage(call),
		Location: Location{Path: filepath.ToSlash(relative), Line: position.Line, Column: position.Column},
	})
}

func declareRangeNames(statement *ast.RangeStmt, scope *muxScope) {
	if identifier, ok := statement.Key.(*ast.Ident); ok {
		scope.declare(identifier.Name, false)
	}
	if identifier, ok := statement.Value.(*ast.Ident); ok {
		scope.declare(identifier.Name, false)
	}
}

func generatedByLoom(source []byte) bool {
	prefix := source
	if len(prefix) > 512 {
		prefix = prefix[:512]
	}
	return strings.Contains(string(prefix), "Code generated by Loom, DO NOT EDIT.")
}

func loomHTTPAliases(file *ast.File) map[string]struct{} {
	aliases := make(map[string]struct{})
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil || path != loomHTTPImportPath {
			continue
		}
		alias := "http"
		if spec.Name != nil {
			alias = spec.Name.Name
		}
		if alias != "." && alias != "_" {
			aliases[alias] = struct{}{}
		}
	}
	return aliases
}

func collectMuxParameters(fields *ast.FieldList, aliases map[string]struct{}, scope *muxScope) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		mux := isMuxType(field.Type, aliases)
		for _, name := range field.Names {
			scope.declare(name.Name, mux)
		}
	}
}

func isMuxType(node ast.Expr, aliases map[string]struct{}) bool {
	selector, ok := node.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	identifier, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	if _, ok := aliases[identifier.Name]; !ok {
		return false
	}
	switch selector.Sel.Name {
	case "Muxer", "MiddlewareMuxer", "ResolverMuxer":
		return true
	default:
		return false
	}
}

func collectMuxAssignments(assignment *ast.AssignStmt, aliases map[string]struct{}, scope *muxScope) {
	if assignment.Tok != token.DEFINE {
		return
	}
	for index, left := range assignment.Lhs {
		identifier, ok := left.(*ast.Ident)
		if !ok || identifier.Name == "_" {
			continue
		}
		if _, exists := scope.values[identifier.Name]; exists {
			continue
		}
		mux := false
		if len(assignment.Rhs) == len(assignment.Lhs) {
			mux = isMuxValue(assignment.Rhs[index], aliases, scope)
		}
		scope.declare(identifier.Name, mux)
	}
}

func collectMuxValueSpecs(specs []ast.Spec, aliases map[string]struct{}, scope *muxScope) {
	for _, rawSpec := range specs {
		spec, ok := rawSpec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		typedMux := isMuxType(spec.Type, aliases)
		for index, name := range spec.Names {
			mux := typedMux
			if !mux && len(spec.Values) == len(spec.Names) {
				mux = isMuxValue(spec.Values[index], aliases, scope)
			}
			scope.declare(name.Name, mux)
		}
	}
}

func isMuxValue(node ast.Expr, aliases map[string]struct{}, scope *muxScope) bool {
	switch actual := node.(type) {
	case *ast.Ident:
		return scope.isMux(actual.Name)
	case *ast.CallExpr:
		selector, ok := actual.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "NewMuxer" {
			return false
		}
		identifier, ok := selector.X.(*ast.Ident)
		if !ok {
			return false
		}
		_, ok = aliases[identifier.Name]
		return ok
	default:
		return false
	}
}

func isMuxHandleCall(call *ast.CallExpr, aliases map[string]struct{}, scope *muxScope) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Handle" {
		return false
	}
	return isMuxValue(selector.X, aliases, scope)
}

func unmanagedRouteMessage(call *ast.CallExpr) string {
	method := stringLiteralArgument(call, 0)
	path := stringLiteralArgument(call, 1)
	switch {
	case method != "" && path != "":
		return fmt.Sprintf("route %s %s is registered directly on a Loom mux; declare and mount it through the design", method, path)
	case path != "":
		return fmt.Sprintf("route %s is registered directly on a Loom mux; declare and mount it through the design", path)
	default:
		return "route is registered directly on a Loom mux; declare and mount it through the design"
	}
}

func stringLiteralArgument(call *ast.CallExpr, index int) string {
	if index >= len(call.Args) {
		return ""
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
	moduleData, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil, fmt.Errorf("read go.mod: %w", err)
	}
	module, err := modfile.Parse("go.mod", moduleData, nil)
	if err != nil {
		return nil, fmt.Errorf("parse go.mod: %w", err)
	}
	requiredVersion, comparable := effectiveLoomVersion(module)
	if !comparable {
		return nil, nil
	}

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
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		manifest := struct {
			Version string `json:"loom_version"`
		}{}
		if unmarshalErr := json.Unmarshal(data, &manifest); unmarshalErr != nil {
			return fmt.Errorf("parse %s: %w", path, unmarshalErr)
		}
		if manifest.Version == "" || manifest.Version == requiredVersion {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			relative = path
		}
		diagnostics = append(diagnostics, Diagnostic{
			Rule:     RuleGeneratedVersionSkew,
			Severity: SeverityError,
			Message:  fmt.Sprintf("generated with Loom %s but go.mod requires %s; run loom gen", manifest.Version, requiredVersion),
			Location: Location{Path: filepath.ToSlash(relative)},
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("inspect generated Loom versions: %w", err)
	}
	return diagnostics, nil
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
