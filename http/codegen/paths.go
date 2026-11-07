package codegen

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/representation"
)

// PathFiles returns the service path files.
func PathFiles(genpkg string, data *ServicesData) []*codegen.File {
	services := data.servicesForGeneration()
	fw := make([]*codegen.File, 2*len(services))
	for i, service := range services {
		fw[i*2] = serverPath(genpkg, service, data)
		fw[i*2+1] = clientPath(genpkg, service, data)
	}
	return fw
}

// ServerPathFiles returns the service path files used by HTTP servers.
func ServerPathFiles(genpkg string, data *ServicesData) []*codegen.File {
	services := data.servicesForGeneration()
	files := make([]*codegen.File, len(services))
	for i, service := range services {
		files[i] = serverPath(genpkg, service, data)
	}
	return files
}

// serverPath returns the server file containing the request path constructors
// for the given service.
func serverPath(genpkg string, svc *expr.HTTPServiceExpr, services *ServicesData) *codegen.File {
	sd := services.Get(svc.Name())
	path := filepath.Join(codegen.Gendir, "http", sd.Service.PathName, "server", "paths.go")
	return &codegen.File{Path: path, Sections: pathSections(genpkg, svc, "server", services)}
}

// clientPath returns the client file containing the request path constructors
// for the given service.
func clientPath(genpkg string, svc *expr.HTTPServiceExpr, services *ServicesData) *codegen.File {
	sd := services.Get(svc.Name())
	path := filepath.Join(codegen.Gendir, "http", sd.Service.PathName, "client", "paths.go")
	return &codegen.File{Path: path, Sections: pathSections(genpkg, svc, "client", services)}
}

// pathSections returns the sections of the file of the pkg package that
// contains the request path constructors for the given service.
func pathSections(genpkg string, svc *expr.HTTPServiceExpr, pkg string, services *ServicesData) []codegen.Section {
	title := fmt.Sprintf("HTTP request path constructors for the %s service.", svc.Name())
	sections := make([]codegen.Section, 0, 1+len(svc.HTTPEndpoints))
	sdata := services.Get(svc.Name())
	imports := append(pathImports(), &codegen.ImportSpec{Path: genpkg + "/" + sdata.Service.PathName, Name: sdata.Service.PkgName})
	imports = append(imports, sdata.Service.UserTypeImports...)
	sdata, imports = services.FileData(svc.Name(), imports)
	sections = append(sections, codegen.Header(title, pkg, imports))
	for _, e := range svc.HTTPEndpoints {
		sections = append(sections, pathSection(sdata.Endpoint(e.Name())))
	}

	return sections
}

// pathImports returns the imports of the path builder files.
func pathImports() []*codegen.ImportSpec {
	return []*codegen.ImportSpec{
		{Path: "encoding/json/jsontext"},
		{Path: "fmt"},
		{Path: "sort"},
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
		b      strings.Builder
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
	b.WriteString("\treturn " + renderJen(jen.Qual("fmt", "Sprintf")) + "(" + renderJen(jen.Lit(format.String())))
	for _, value := range values {
		b.WriteString(", " + value)
	}
	b.WriteString(")\n")
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
func renderPathValue(b *strings.Builder, arg *InitArgData, escape string) string {
	typ := arg.FieldType
	if expr.IsArray(typ) {
		fmt.Fprintf(b, "\t%s := make([]string, len(%s))\n", arg.Locals.Slice, arg.VarName)
		fmt.Fprintf(b, "\tfor i, v := range %s {\n", arg.VarName)
		element := expr.AsArray(typ).ElemType
		conversion := renderPathSliceConversion(element.Type, escape)
		if physical := pathPhysicalElement(arg); representation.IsIntegerKind(element.Type) && !nativeIntegerStorage(physical) {
			// Public path builders do not validate; they print the stored value
			// faithfully. BuildRequest preflight rejects unrepresentable values.
			conversion = renderIntegerDecimal(b, physical, arg.fieldContext.Scope.Ref(physical, arg.fieldContext.Pkg(physical)), "v", arg.Locals.Text)
		}
		fmt.Fprintf(b, "\t\t%s[i] = %s\n", arg.Locals.Slice, conversion)
		b.WriteString("\t}\n")
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
	if representation.IsIntegerKind(typ) {
		// Integer storage overrides are transparent: print the decimal value
		// without validating the wire range here.
		return renderIntegerDecimal(b, arg.AttributeData.fieldAttribute, arg.TypeRef, arg.VarName, arg.Locals.Text)
	}
	return escape + "(" + renderJen(jen.Qual("fmt", "Sprint").Call(jen.Id(arg.VarName))) + ")"
}

// pathNeedsIntegerText reports an ordinary path argument whose faithful
// integer text needs a declared local: opaque scalar storage or opaque array
// element storage. It mirrors the integer branches of renderPathValue.
func pathNeedsIntegerText(arg *InitArgData) bool {
	if arg.parameter != nil && arg.parameter.Styled || arg.AttributeData.fieldAttribute == nil {
		return false
	}
	if array := expr.AsArray(arg.FieldType); array != nil {
		physical := pathPhysicalElement(arg)
		return representation.IsIntegerKind(array.ElemType.Type) && !nativeIntegerStorage(physical) && representation.IntegerStorage(physical) == nil
	}
	typ, _ := transportUnderlyingType(arg.FieldType)
	return representation.IsIntegerKind(typ) && arg.TypeRef != codegen.GoNativeTypeName(typ) && representation.IntegerStorage(arg.AttributeData.fieldAttribute) == nil
}

// pathPhysicalElement returns the actual element storage of an array path
// argument. InitArgData.fieldAttribute names the constructor destination, so
// the retained transport storage is read from the embedded AttributeData.
func pathPhysicalElement(arg *InitArgData) *expr.AttributeExpr {
	return expr.AsArray(arg.AttributeData.fieldAttribute.Type).ElemType
}

// nativeIntegerStorage reports an unnamed integer whose storage is the wire's
// own Go carrier.
func nativeIntegerStorage(attribute *expr.AttributeExpr) bool {
	if _, named := attribute.Type.(expr.UserType); named {
		return false
	}
	storage := representation.IntegerStorage(attribute)
	return storage != nil && storage.Name() == codegen.GoNativeTypeName(attribute.Type)
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
