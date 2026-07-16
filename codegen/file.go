package codegen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/scanner"
	"go/token"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/imports"
)

// Gendir is the name of the subdirectory of the output directory that contains
// the generated files. This directory is wiped and re-written each time Loom is
// run.
const Gendir = "gen"

type (
	// A Section renders one file fragment.
	Section interface {
		// SectionName returns the stable section identifier used by tests and
		// merge logic.
		SectionName() string
		// Write writes the rendered section content to the given writer.
		Write(io.Writer) error
	}

	// A File contains the logic to generate a complete file.
	File struct {
		// Sections is the list of file sections in order of rendering.
		Sections []Section
		// SectionTemplates is the template-backed section list used by external
		// generator extensions. Framework-owned generators use Sections.
		SectionTemplates []*SectionTemplate
		// Path returns the file path relative to the output directory.
		Path string
		// SkipExist indicates whether the file should be skipped if one
		// already exists at the given path.
		SkipExist bool
		// FinalizeFunc is called after the file has been generated. It
		// is given the absolute path to the file as argument.
		FinalizeFunc func(string) error
	}

	// A SectionTemplate is a template and accompanying render data. The
	// template format is described in the (stdlib) text/template package.
	SectionTemplate struct {
		// Name is the name reported when parsing the source fails.
		Name string
		// Source is used to create the text/template.Template that
		// renders the section text.
		Source string
		// FuncMap lists the functions used to render the templates.
		FuncMap map[string]any
		// Data used as input of template.
		Data any
	}
)

// Section returns the sections with the given name or nil if not found.
func (f *File) Section(name string) []Section {
	var sts []Section
	for _, s := range f.AllSections() {
		if s.SectionName() == name {
			sts = append(sts, s)
		}
	}
	return sts
}

// AllSections returns all file sections using the generic section abstraction.
func (f *File) AllSections() []Section {
	if len(f.Sections) > 0 {
		return f.Sections
	}
	sections := make([]Section, len(f.SectionTemplates))
	for i, section := range f.SectionTemplates {
		sections[i] = section
	}
	return sections
}

// SetSections replaces the file sections with the given generic section list.
func (f *File) SetSections(sections []Section) {
	f.Sections = sections
	f.SectionTemplates = nil
}

// HeaderTemplate returns the first section when it is a template-backed header.
func (f *File) HeaderTemplate() *SectionTemplate {
	header := f.HeaderSection()
	if header == nil {
		return nil
	}
	template, _ := header.(*SectionTemplate)
	return template
}

// HeaderSection returns the first section when it is a generated source header.
func (f *File) HeaderSection() Section {
	sections := f.AllSections()
	if len(sections) == 0 || sections[0].SectionName() != "source-header" {
		return nil
	}
	return sections[0]
}

// Render executes the file section templates and writes the resulting bytes to
// an output file. The path of the output file is computed by appending the file
// path to dir. If SkipExist is true and a file already exists with the computed
// path, Render skips writing and returns an empty path. Otherwise, Render
// overwrites the computed path and returns it.
func (f *File) Render(dir string) (string, error) {
	base, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	path := filepath.Join(base, f.Path)
	if f.SkipExist {
		if _, err = os.Stat(path); err == nil {
			return "", nil
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return "", err
	}

	// Render all sections to a buffer instead of directly to file
	var buf bytes.Buffer
	for _, s := range f.AllSections() {
		if err := s.Write(&buf); err != nil {
			return "", err
		}
	}

	// For Go files, process everything in memory
	content := buf.Bytes()
	if filepath.Ext(path) == ".go" {
		content, err = finalizeGoSource(path, content)
		if err != nil {
			return "", err
		}
	}

	// Write the final content exactly once
	if err := os.WriteFile(path, content, 0644); err != nil {
		return "", err
	}

	// Run finalizer if any
	if f.FinalizeFunc != nil {
		if err := f.FinalizeFunc(path); err != nil {
			return "", err
		}
	}

	return path, nil
}

// SectionName returns the stable identifier of the section.
func (s *SectionTemplate) SectionName() string {
	return s.Name
}

// Write writes the section to the given writer.
func (s *SectionTemplate) Write(w io.Writer) error {
	if s.Name == "source-header" {
		return renderHeaderSection(w, HeaderSectionData(s))
	}
	funcs := TemplateFuncs()
	maps.Copy(funcs, s.FuncMap)
	tmpl := template.Must(template.New(s.Name).Funcs(funcs).Parse(s.Source))
	return tmpl.Execute(w, s.Data)
}

// finalizeGoSource processes Go source entirely in memory without file I/O
func finalizeGoSource(path string, content []byte) ([]byte, error) {
	// Parse the content
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, content, parser.ParseComments)
	if err != nil {
		var buf bytes.Buffer
		scanner.PrintError(&buf, err)
		return nil, fmt.Errorf("%s\n========\nContent:\n%s", buf.String(), content)
	}

	compilePatternValidations(fset, file, path)

	// Clean unused imports using optimized single-pass detection
	impMap := buildImportMap(file)
	detectUsedImports(file, impMap)
	removeUnusedImports(fset, file, impMap)
	ast.SortImports(fset, file)

	// Format the AST back to bytes
	var formatted bytes.Buffer
	if err := format.Node(&formatted, fset, file); err != nil {
		return nil, err
	}

	// Apply goimports formatting
	opt := imports.Options{
		Comments:   true,
		FormatOnly: true,
	}
	result, err := imports.Process(path, formatted.Bytes(), &opt)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func compilePatternValidations(fset *token.FileSet, file *ast.File, path string) {
	prefix := patternVarPrefix(path)
	usedNames := topLevelNames(file)
	patternVars := make(map[string]string)
	var patternOrder []string
	changed := false

	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || !isLoomValidatePatternCall(call) || len(call.Args) != 3 {
			return true
		}
		pattern, ok := call.Args[2].(*ast.BasicLit)
		if !ok || pattern.Kind != token.STRING {
			return true
		}
		if _, err := strconv.Unquote(pattern.Value); err != nil {
			return true
		}
		name, ok := patternVars[pattern.Value]
		if !ok {
			name = uniquePatternVarName(prefix, len(patternVars), usedNames)
			patternVars[pattern.Value] = name
			patternOrder = append(patternOrder, pattern.Value)
		}
		call.Fun.(*ast.SelectorExpr).Sel.Name = "ValidatePatternCompiled"
		call.Args[2] = ast.NewIdent(name)
		changed = true
		return true
	})

	if !changed {
		return
	}

	astutil.AddImport(fset, file, "regexp")
	specs := make([]ast.Spec, 0, len(patternOrder))
	for _, pattern := range patternOrder {
		specs = append(specs, &ast.ValueSpec{
			Names: []*ast.Ident{ast.NewIdent(patternVars[pattern])},
			Values: []ast.Expr{&ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent("regexp"),
					Sel: ast.NewIdent("MustCompile"),
				},
				Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: pattern}},
			}},
		})
	}
	decl := &ast.GenDecl{Tok: token.VAR, Specs: specs}
	file.Decls = append(file.Decls, decl)
}

func isLoomValidatePatternCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "ValidatePattern" {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	return ok && ident.Name == "loom"
}

// patternVarPrefix derives a per-file identifier prefix so pattern variables
// hoisted in different files of the same package cannot collide.
func patternVarPrefix(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), ".go")
	var b strings.Builder
	b.WriteString("loomPattern")
	upper := true
	for _, r := range base {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			upper = true
			continue
		}
		if upper {
			r = unicode.ToUpper(r)
			upper = false
		}
		b.WriteRune(r)
	}
	return b.String()
}

func uniquePatternVarName(prefix string, index int, used map[string]struct{}) string {
	for {
		name := fmt.Sprintf("%s%d", prefix, index)
		if _, ok := used[name]; !ok {
			used[name] = struct{}{}
			return name
		}
		index++
	}
}

func topLevelNames(file *ast.File) map[string]struct{} {
	used := make(map[string]struct{})
	for _, decl := range file.Decls {
		switch actual := decl.(type) {
		case *ast.FuncDecl:
			if actual.Name != nil {
				used[actual.Name.Name] = struct{}{}
			}
		case *ast.GenDecl:
			for _, spec := range actual.Specs {
				switch typed := spec.(type) {
				case *ast.TypeSpec:
					used[typed.Name.Name] = struct{}{}
				case *ast.ValueSpec:
					for _, name := range typed.Names {
						used[name.Name] = struct{}{}
					}
				}
			}
		}
	}
	return used
}
