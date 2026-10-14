package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

type (
	mergedGoSection struct {
		path     string
		sections []codegen.Section
	}
	declarationOwner struct {
		definitions map[string]string
		conflicts   map[string]struct{}
	}
)

func (s *mergedGoSection) SectionName() string {
	return "merged-go-declarations"
}

func (s *mergedGoSection) Write(w io.Writer) error {
	source, err := s.source()
	if err != nil {
		return err
	}
	merged, err := mergeGoDeclarations(s.path, source)
	if err != nil {
		return err
	}
	_, err = w.Write(merged)
	return err
}

// source retains every non-header section, independently of its category.
func (s *mergedGoSection) source() ([]byte, error) {
	var header, additionalImports, body bytes.Buffer
	packages := map[string]struct{}{}
	imports := map[string]struct{}{}
	bindings := map[string]string{}
	conflicts := map[string]struct{}{}
	for _, section := range s.sections {
		if section.SectionName() != "source-header" {
			if err := section.Write(&body); err != nil {
				return nil, err
			}
			body.WriteByte('\n')
			continue
		}
		var rendered bytes.Buffer
		if err := section.Write(&rendered); err != nil {
			return nil, err
		}
		f, err := parser.ParseFile(token.NewFileSet(), s.path, rendered.Bytes(), parser.ImportsOnly)
		if err != nil {
			return nil, err
		}
		packages[f.Name.Name] = struct{}{}
		first := header.Len() == 0
		if first {
			header.Write(rendered.Bytes())
		}
		for _, spec := range f.Imports {
			name := ""
			if spec.Name != nil {
				name = spec.Name.Name
			}
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return nil, err
			}
			if name != "" && name != "_" && name != "." {
				if prior, ok := bindings[name]; ok && prior != path {
					conflicts[name] = struct{}{}
				}
				bindings[name] = path
			}
			key := name + "\x00" + path
			if _, ok := imports[key]; ok {
				continue
			}
			imports[key] = struct{}{}
			if !first {
				fmt.Fprintf(&additionalImports, "import %s %s\n", name, strconv.Quote(path))
			}
		}
	}
	if len(packages) > 1 {
		return nil, fmt.Errorf("conflicting package declarations in %s: %s", s.path, sortedKeys(packages))
	}
	if len(conflicts) > 0 {
		return nil, fmt.Errorf("conflicting import bindings in %s: %s", s.path, sortedKeys(conflicts))
	}
	header.Write(additionalImports.Bytes())
	header.Write(body.Bytes())
	return header.Bytes(), nil
}

// mergeGoDeclarations applies declaration ownership only to coalesced artifacts.
// Const groups and multi-name value specs are indivisible because their meaning
// can depend on declaration order, implicit expressions, or tuple assignment.
func mergeGoDeclarations(path string, source []byte) ([]byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, source, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	plain, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
	if err != nil {
		return nil, err
	}
	comments := ast.NewCommentMap(fset, file, file.Comments)
	owner := &declarationOwner{definitions: map[string]string{}, conflicts: map[string]struct{}{}}
	var declarations []ast.Decl
	for i, decl := range file.Decls {
		keep, err := owner.filter(decl, plain.Decls[i], file.Comments)
		if err != nil {
			return nil, err
		}
		if keep {
			declarations = append(declarations, decl)
		}
	}
	if len(owner.conflicts) > 0 {
		return nil, fmt.Errorf("conflicting Go declarations in %s: %s", path, sortedKeys(owner.conflicts))
	}
	file.Decls = declarations
	file.Comments = comments.Filter(file).Comments()
	var out bytes.Buffer
	if err := format.Node(&out, fset, file); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func (o *declarationOwner) filter(decl, plain ast.Decl, comments []*ast.CommentGroup) (bool, error) {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		if d.Recv == nil && (d.Name.Name == "init" || d.Name.Name == "_") {
			return true, nil
		}
		key := d.Name.Name
		if d.Recv != nil {
			key = receiverName(d.Recv.List[0].Type) + "." + key
		}
		definition, err := declarationDefinition(plain, d, comments)
		if err != nil {
			return false, err
		}
		return o.admit([]string{key}, definition, false), nil
	case *ast.GenDecl:
		if d.Tok == token.IMPORT {
			return true, nil
		}
		p := plain.(*ast.GenDecl)
		if d.Tok == token.CONST {
			var keys []string
			for _, spec := range d.Specs {
				names, _ := specNames(spec)
				keys = append(keys, names...)
			}
			definition, err := declarationDefinition(p, d, comments)
			if err != nil {
				return false, err
			}
			return o.admit(keys, definition, false), nil
		}
		var specs []ast.Spec
		for i, spec := range d.Specs {
			keys, blank := specNames(spec)
			normalized := &ast.GenDecl{Tok: d.Tok, Specs: []ast.Spec{p.Specs[i]}}
			definition, err := declarationDefinition(normalized, spec, comments)
			if err != nil {
				return false, err
			}
			definition += directiveText(d.Doc)
			if o.admit(keys, definition, blank && d.Tok == token.VAR) {
				specs = append(specs, spec)
			}
		}
		d.Specs = specs
		return len(specs) > 0, nil
	default:
		return true, nil
	}
}

func (o *declarationOwner) admit(keys []string, definition string, blankEffect bool) bool {
	if len(keys) == 0 {
		return true
	}
	duplicates := 0
	local := map[string]struct{}{}
	for _, key := range keys {
		if _, ok := local[key]; ok {
			o.conflicts[key] = struct{}{}
		}
		local[key] = struct{}{}
		if prior, ok := o.definitions[key]; ok {
			duplicates++
			if prior != definition || blankEffect {
				o.conflicts[key] = struct{}{}
			}
		}
	}
	if duplicates > 0 && duplicates < len(keys) {
		for _, key := range keys {
			if _, ok := o.definitions[key]; ok {
				o.conflicts[key] = struct{}{}
			}
		}
	}
	// Retain every encountered identity even after rejecting an atomic overlap,
	// so a later conflict reports the same symbols in every contribution order.
	for _, key := range keys {
		if _, ok := o.definitions[key]; !ok {
			o.definitions[key] = definition
		}
	}
	return duplicates == 0
}

func specNames(spec ast.Spec) ([]string, bool) {
	switch s := spec.(type) {
	case *ast.TypeSpec:
		if s.Name.Name == "_" {
			return nil, false
		}
		return []string{s.Name.Name}, false
	case *ast.ValueSpec:
		var names []string
		blank := false
		for _, name := range s.Names {
			if name.Name == "_" {
				blank = true
				continue
			}
			names = append(names, name.Name)
		}
		return names, blank
	}
	return nil, false
}

func receiverName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return receiverName(e.X)
	case *ast.IndexExpr:
		return receiverName(e.X)
	case *ast.IndexListExpr:
		return receiverName(e.X)
	case *ast.ParenExpr:
		return receiverName(e.X)
	}
	return "<invalid receiver>"
}

// Ordinary documentation does not change a declaration's definition. Compiler
// directives do, so retain their exact text when comparing otherwise equal ASTs.
func declarationDefinition(plain, original ast.Node, comments []*ast.CommentGroup) (string, error) {
	var b bytes.Buffer
	if err := format.Node(&b, token.NewFileSet(), plain); err != nil {
		return "", err
	}
	start := original.Pos()
	switch n := original.(type) {
	case *ast.FuncDecl:
		if n.Doc != nil {
			start = n.Doc.Pos()
		}
	case *ast.GenDecl:
		if n.Doc != nil {
			start = n.Doc.Pos()
		}
	case *ast.TypeSpec:
		if n.Doc != nil {
			start = n.Doc.Pos()
		}
	case *ast.ValueSpec:
		if n.Doc != nil {
			start = n.Doc.Pos()
		}
	}
	for _, group := range comments {
		if group.Pos() < start || group.End() > original.End() {
			continue
		}
		b.WriteString(directiveText(group))
	}
	return b.String(), nil
}

func directiveText(group *ast.CommentGroup) string {
	if group == nil {
		return ""
	}
	var b strings.Builder
	for _, comment := range group.List {
		if strings.HasPrefix(comment.Text, "//go:") || strings.HasPrefix(comment.Text, "//line ") || strings.HasPrefix(comment.Text, "//export ") {
			b.WriteString("\n" + comment.Text)
		}
	}
	return b.String()
}

func sortedKeys(values map[string]struct{}) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return strings.Join(keys, ", ")
}
