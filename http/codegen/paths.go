package codegen

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// PathFiles returns the service path files.
func PathFiles(data *ServicesData) []*codegen.File {
	fw := make([]*codegen.File, 2*len(data.Expressions.Services))
	for i := 0; i < len(data.Expressions.Services); i++ {
		fw[i*2] = serverPath(data.Expressions.Services[i], data)
		fw[i*2+1] = clientPath(data.Expressions.Services[i], data)
	}
	return fw
}

// ServerPathFiles returns the service path files used by HTTP servers.
func ServerPathFiles(data *ServicesData) []*codegen.File {
	files := make([]*codegen.File, len(data.Expressions.Services))
	for i, service := range data.Expressions.Services {
		files[i] = serverPath(service, data)
	}
	return files
}

// serverPath returns the server file containing the request path constructors
// for the given service.
func serverPath(svc *expr.HTTPServiceExpr, services *ServicesData) *codegen.File {
	sd := services.Get(svc.Name())
	path := filepath.Join(codegen.Gendir, "http", sd.Service.PathName, "server", "paths.go")
	return &codegen.File{Path: path, Sections: pathSections(svc, "server", services)}
}

// clientPath returns the client file containing the request path constructors
// for the given service.
func clientPath(svc *expr.HTTPServiceExpr, services *ServicesData) *codegen.File {
	sd := services.Get(svc.Name())
	path := filepath.Join(codegen.Gendir, "http", sd.Service.PathName, "client", "paths.go")
	return &codegen.File{Path: path, Sections: pathSections(svc, "client", services)}
}

// pathSections returns the sections of the file of the pkg package that
// contains the request path constructors for the given service.
func pathSections(svc *expr.HTTPServiceExpr, pkg string, services *ServicesData) []codegen.Section {
	title := fmt.Sprintf("HTTP request path constructors for the %s service.", svc.Name())
	sections := make([]codegen.Section, 0, 1+len(svc.HTTPEndpoints))
	sections = append(sections,
		codegen.Header(title, pkg, pathImports()),
	)
	sdata := services.Get(svc.Name())
	for _, e := range svc.HTTPEndpoints {
		sections = append(sections, pathSection(sdata.Endpoint(e.Name())))
	}

	return sections
}

// pathImports returns the imports of the path builder files.
func pathImports() []*codegen.ImportSpec {
	return []*codegen.ImportSpec{
		{Path: "fmt"},
		{Path: "strconv"},
		{Path: "strings"},
		codegen.LoomImport(""),
		codegen.LoomNamedImport("http", "loomhttp"),
	}
}

// renderPathInitCode returns the body of the path builder of the route path
// whose wildcards take the values of args, in the order of the wildcards. The
// builder returns an escaped URL path: the literal parts of the route are
// escaped when the builder is generated, and every value that may need it is
// escaped at run time, keeping the "/" separators of a catch-all value.
func renderPathInitCode(args []*InitArgData, path string) string {
	if len(args) == 0 {
		return "\treturn " + renderJen(jen.Lit(escapeRouteLiteral(path))) + "\n"
	}
	var (
		b      sourceBuilder
		format strings.Builder
		values = make([]string, len(args))
		last   int
	)
	for i, match := range expr.HTTPWildcardRegex.FindAllStringIndex(path, -1) {
		escape := "loomhttp.EscapePathSegment"
		if strings.HasPrefix(path[match[0]:], "/{*") {
			escape = "loomhttp.EscapePathRemainder"
		}
		format.WriteString(pathFormatLiteral(path[last:match[0]]))
		format.WriteString("/%v")
		last = match[1]
		values[i] = renderPathValue(&b, args[i], escape)
	}
	format.WriteString(pathFormatLiteral(path[last:]))
	b.Add("\treturn " + renderJen(jen.Qual("fmt", "Sprintf")) + "(" + renderJen(jen.Lit(format.String())))
	for _, value := range values {
		b.Add(", " + value)
	}
	b.Add(")\n")
	return b.String()
}

// escapeRouteLiteral returns the literal part of a route path escaped as URL
// path text, keeping its "/" separators.
func escapeRouteLiteral(literal string) string {
	return (&url.URL{Path: literal}).EscapedPath()
}

// pathFormatLiteral returns the literal part of a route path escaped as URL
// path text and quoted for a fmt format.
func pathFormatLiteral(literal string) string {
	return strings.ReplaceAll(escapeRouteLiteral(literal), "%", "%%")
}

// renderPathValue returns the expression that formats the value of the path
// parameter arg as escaped path text with the runtime function escape, and
// writes to b the statements that the expression needs. Numbers and booleans
// never need escaping and are formatted as they are.
func renderPathValue(b *sourceBuilder, arg *InitArgData, escape string) string {
	typ := arg.FieldType
	if expr.IsArray(typ) {
		b.Addf("\t%s := make([]string, len(%s))\n", arg.Locals.Slice, arg.VarName)
		b.Addf("\tfor i, v := range %s {\n", arg.VarName)
		b.Addf("\t\t%s[i] = %s\n", arg.Locals.Slice, renderPathSliceConversion(expr.AsArray(typ).ElemType.Type, escape))
		b.Add("\t}\n")
		return "strings.Join(" + arg.Locals.Slice + ", \",\")"
	}
	if expr.IsAny(typ) {
		return escape + "(loom.JSONValueString(" + arg.VarName + "))"
	}
	for {
		ut, ok := typ.(expr.UserType)
		if !ok {
			break
		}
		typ = ut.Attribute().Type
	}
	if _, ok := typ.(expr.Primitive); ok && arg.TypeRef == codegen.GoNativeTypeName(typ) {
		switch typ.Kind() {
		case expr.StringKind:
			return escape + "(" + arg.VarName + ")"
		case expr.BytesKind:
			return escape + "(string(" + arg.VarName + "))"
		default:
			return arg.VarName
		}
	}
	return escape + "(" + renderJen(jen.Qual("fmt", "Sprint").Call(jen.Id(arg.VarName))) + ")"
}

// renderPathSliceConversion returns the expression that formats the element v
// of an array path parameter of element type dt as escaped path text with the
// runtime function escape.
func renderPathSliceConversion(dt expr.DataType, escape string) string {
	switch dt.Name() {
	case "string":
		return escape + "(v)"
	case "bytes":
		return escape + "(string(v))"
	}
	converted := renderQuerySliceConversion(dt)
	if inner, ok := strings.CutPrefix(converted, "url.QueryEscape("); ok {
		return escape + "(" + inner
	}
	return converted
}
