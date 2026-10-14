package codegen

import (
	"fmt"
	"go/scanner"
	"go/token"
	"strconv"

	"github.com/CaliLuke/loom/expr"
)

func renderCustomStringDefault(attribute *expr.AttributeExpr, text string, context *AttributeContext, path string) string {
	ref := context.Scope.Ref(attribute, context.Pkg(attribute))
	scope := representationLocals(ref)
	value := scope.Unique("defaultValue")
	failure := scope.Unique("defaultError")
	return fmt.Sprintf("func() %s {\nvar %s %s\nif *transformErr != nil {\nreturn %s\n}\n%s, %s := loom.ParseStringValue[%s](%s, %s)\nif %s != nil {\n*transformErr = %s\n}\nreturn %s\n}()", ref, value, ref, value, value, failure, ref, strconv.Quote(text), strconv.Quote(path), failure, failure, value)
}

// representationLocals reserves referenced identifiers before allocating an
// expression's lexical locals, including custom type package aliases.
func representationLocals(source string) *NameScope {
	scope := NewNameScope()
	files := token.NewFileSet()
	file := files.AddFile("representation", files.Base(), len(source))
	var scan scanner.Scanner
	scan.Init(file, []byte(source), nil, 0)
	for {
		_, kind, literal := scan.Scan()
		if kind == token.EOF {
			break
		}
		if kind == token.IDENT {
			scope.Unique(literal)
		}
	}
	scope.Unique("loom")
	scope.Unique("transformErr")
	return scope
}
