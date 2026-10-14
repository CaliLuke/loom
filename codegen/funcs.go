package codegen

import (
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/naming"
	"github.com/CaliLuke/loom/internal/representation"
)

type (
	// InitArgData contains the data needed to render code to initialize struct
	// fields with the given arguments.
	InitArgData struct {
		// Name is the argument name.
		Name string
		// Pointer if true indicates that the argument is a pointer.
		Pointer bool
		// Type is the argument type.
		Type expr.DataType
		// Attribute preserves the argument occurrence metadata when provided.
		Attribute *expr.AttributeExpr
		// FieldName is the name of the field in the struct initialized by the
		// argument.
		FieldName string
		// FieldPointer if true indicates that the field in the struct is a
		// pointer.
		FieldPointer bool
		// FieldType is the type of the field in the struct.
		FieldType expr.DataType
		// FieldAttribute preserves the destination occurrence metadata when provided.
		FieldAttribute *expr.AttributeExpr
	}
)

// TemplateFuncs lists common template helper functions.
func TemplateFuncs() map[string]any {
	return map[string]any{
		"commandLine": CommandLine,
		"comment":     Comment,
	}
}

// CommandLine return the command used to run this process.
func CommandLine() string {
	cmdl := "loom"
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "--cmd=") {
			cmdl = arg[6:]
			break
		}
	}
	return cmdl
}

// Comment produces line comments by concatenating the given strings and
// producing 80 characters long lines starting with "//". Carriage returns
// break lines, and invalid UTF-8, NUL bytes, and byte order marks, which Go
// source cannot contain, are replaced by U+FFFD.
func Comment(elems ...string) string {
	sanitized := make([]string, len(elems))
	lineCount := 0
	for i, e := range elems {
		sanitized[i] = sanitizeCommentText(e)
		lineCount += strings.Count(sanitized[i], "\n") + 1
	}
	lines := make([]string, 0, lineCount)
	for _, e := range sanitized {
		lines = append(lines, strings.Split(e, "\n")...)
	}
	var trimmed = make([]string, len(lines))
	for i, l := range lines {
		trimmed[i] = strings.TrimLeft(l, " \t")
	}
	t := strings.Join(trimmed, "\n")

	return Indent(WrapText(t, 77), "// ")
}

// Indent inserts prefix at the beginning of each non-empty line of s. The
// end-of-line marker is NL.
func Indent(s, prefix string) string {
	var (
		b   = []byte(s)
		p   = []byte(prefix)
		res = make([]byte, 0, len(b)+len(b)/4*len(p)) // preallocate with estimated size
		bol = true
	)
	for _, c := range b {
		if bol && c != '\n' {
			res = append(res, p...)
		}
		res = append(res, c)
		bol = c == '\n'
	}
	return string(res)
}

// CamelCase produces the CamelCase version of the given string. It removes any
// non letter and non digit character.
//
// If firstUpper is true the first letter of the string is capitalized else
// the first letter is in lowercase.
//
// If acronym is true and a part of the string is a common acronym
// then it keeps the part capitalized (firstUpper = true)
// (e.g. APIVersion) or lowercase (firstUpper = false) (e.g. apiVersion).
func CamelCase(name string, firstUpper, acronym bool) string {
	return naming.CamelCase(name, firstUpper, acronym)
}

// SnakeCase produces the snake_case version of the given CamelCase string.
// News    => news
// OldNews => old_news
// CNNNews => cnn_news
// ÉtéFoo  => été_foo
//
// SnakeCase processes runes, not bytes. Letters without case, such as CJK
// ideographs, continue the current word like lower case letters do. Upper and
// title case letters that have a lower case form start a new word. Combining marks stay attached to the
// rune they follow. Invalid UTF-8 bytes become the Unicode replacement
// character, so the result is always valid UTF-8.
func SnakeCase(name string) string {
	return naming.SnakeCase(name)
}

// KebabCase produces the kebab-case version of the given CamelCase string.
func KebabCase(name string) string {
	return naming.KebabCase(name)
}

// WrapText produces lines with text capped at maxChars
// it will keep words intact and respects newlines.
func WrapText(text string, maxChars int) string {
	res := ""
	lines := strings.Split(text, "\n")
	for _, v := range lines {
		runes := []rune(strings.TrimSpace(v))
		for l := len(runes); l >= 0; l = len(runes) {
			if maxChars >= l {
				res = res + string(runes) + "\n"
				break
			}

			i := runeSpacePosRev(runes[:maxChars])
			if i == 0 {
				i = runeSpacePos(runes)
			}

			res = res + string(runes[:i]) + "\n"
			if l == i {
				break
			}
			runes = runes[i+1:]
		}
	}
	return res[:len(res)-1]
}

// InitStructFields produces Go code to initialize a struct and its fields from
// the given init arguments. scope retains the allocated nominal identities and
// struct:pkg:path import names of the field types. A nil scope selects ordinary
// type and package names. Constructor locals do not reserve package identities.
func InitStructFields(args []*InitArgData, targetVar, sourcePkg, targetPkg string, scope *NameScope) (TransformPlan, error) {
	if scope == nil {
		scope = NewNameScope()
	}

	var (
		code       string
		errorAware bool
		helpers    []*TransformFunctionData
	)
	for _, arg := range args {
		source, target := arg.Attribute, arg.FieldAttribute
		if source == nil {
			source = &expr.AttributeExpr{Type: arg.Type}
		}
		if target == nil {
			target = &expr.AttributeExpr{Type: arg.FieldType}
		}
		sourceContext := NewAttributeContext(arg.Pointer, false, true, sourcePkg, scope)
		targetContext := NewAttributeContext(arg.FieldPointer, false, true, targetPkg, scope)
		attributes := &TransformAttrs{SourceCtx: sourceContext, TargetCtx: targetContext}
		if source.Type != nil && target.Type != nil && stringRepresentationCrossing(source, target, attributes) {
			plan, err := initStringRepresentationField(arg, args, targetVar, source, target, attributes)
			if err != nil {
				return TransformPlan{}, err
			}
			code += plan.Code
			errorAware = errorAware || plan.ErrorAware
			helpers = AppendHelpers(helpers, plan.Helpers)
			continue
		}
		switch {
		case arg.FieldName == "" && arg.FieldType == nil:
		// do nothing
		case canCopyInitField(arg, source, target, attributes):
			// arg type and struct field type are the same. No need to call transform
			// to initialize the field
			deref := ""
			if !arg.Pointer && arg.FieldPointer && expr.IsPrimitive(arg.FieldType) {
				deref = "&"
			}
			code += fmt.Sprintf("%s.%s = %s%s\n", targetVar, arg.FieldName, deref, arg.Name) // nolint: namescope -- target.field assignment, not a type ref
		case expr.IsPrimitive(arg.FieldType):
			code += initAliasedPrimitiveField(arg, targetVar, targetPkg, scope)
		default:
			srcctx := NewAttributeContext(arg.Pointer, false, true, sourcePkg, scope)
			tgtctx := NewAttributeContext(arg.FieldPointer, false, true, targetPkg, scope)
			plan, err := GoTransform(
				source, target,
				arg.Name, fmt.Sprintf("%s.%s", targetVar, arg.FieldName), srcctx, tgtctx, "", false) // nolint: namescope -- target.field lvalue, not a type ref
			if err != nil {
				return TransformPlan{}, err
			}
			code += plan.Code + "\n"
			errorAware = errorAware || plan.ErrorAware
			helpers = AppendHelpers(helpers, plan.Helpers)
		}
	}
	return TransformPlan{Code: code, Helpers: helpers, ErrorAware: errorAware}, nil
}

// canCopyInitField keeps logical equality from hiding custom storage differences.
func canCopyInitField(arg *InitArgData, source, target *expr.AttributeExpr, ta *TransformAttrs) bool {
	if !expr.Equal(unalias(arg.Type), arg.FieldType) {
		return false
	}
	if !representation.ContainsCustomString(source) && !representation.ContainsCustomString(target) {
		return true
	}
	return ta.SourceCtx.Scope.Ref(source, ta.SourceCtx.Pkg(source)) == ta.TargetCtx.Scope.Ref(target, ta.TargetCtx.Pkg(target))
}

func initAliasedPrimitiveField(arg *InitArgData, targetVar, targetPkg string, scope *NameScope) string {
	var code string
	// aliased primitive type
	pkg := targetPkg
	if loc := UserTypeLocation(arg.FieldType); loc != nil {
		pkg = scope.PackageName(loc)
	}
	t := scope.GoFullTypeRef(&expr.AttributeExpr{Type: arg.FieldType}, pkg)
	cast := fmt.Sprintf("%s(%s)", t, arg.Name)
	if arg.Pointer {
		code += "if " + arg.Name + " != nil {\n"
		cast = fmt.Sprintf("%s(*%s)", t, arg.Name)
	}
	switch {
	case arg.FieldPointer:
		code += fmt.Sprintf("tmp%s := %s\n%s.%s = &tmp%s\n", arg.Name, cast, targetVar, arg.FieldName, arg.Name)
	case arg.FieldName != "":
		code += fmt.Sprintf("%s.%s = %s\n", targetVar, arg.FieldName, cast) // nolint: namescope -- target.field assignment, not a type ref
	default:
		code += fmt.Sprintf("%s := %s\n", targetVar, cast)
	}
	if arg.Pointer {
		code += "}\n"
	}
	return code
}

func initStringRepresentationField(arg *InitArgData, args []*InitArgData, targetVar string, source, target *expr.AttributeExpr, ta *TransformAttrs) (TransformPlan, error) {
	var code string
	value := arg.Name
	if arg.Pointer {
		code += "if " + value + " != nil {\n"
		value = "*" + value
	}
	locals := representationLocals(ta.TargetCtx.Scope.Ref(target, ta.TargetCtx.Pkg(target)))
	locals.Unique(targetVar)
	for _, parameter := range args {
		locals.Unique(parameter.Name)
	}
	converted := locals.Unique(Goify(arg.Name, false) + "Converted")
	plan, err := GoTransform(source, target, value, converted, ta.SourceCtx, ta.TargetCtx, "", true)
	if err != nil {
		return TransformPlan{}, err
	}
	code += plan.Code + "\n"
	address := ""
	if arg.FieldPointer {
		address = "&"
	}
	code += targetVar + "." + arg.FieldName + " = " + address + converted + "\n"
	if arg.Pointer {
		code += "}\n"
	}
	plan.Code = code
	return plan, nil
}

// Get the underlying primitive type of a aliased type or return the type itself
// if not aliased.
func unalias(dt expr.DataType) expr.DataType {
	if ut, ok := dt.(expr.UserType); ok {
		if _, ok := ut.Attribute().Type.(expr.Primitive); ok {
			return ut.Attribute().Type
		}
		return unalias(ut.Attribute().Type)
	}
	return dt
}

func runeSpacePosRev(r []rune) int {
	for i := len(r) - 1; i > 0; i-- {
		if unicode.IsSpace(r[i]) {
			return i
		}
	}
	return 0
}

func runeSpacePos(r []rune) int {
	for i := range r {
		if unicode.IsSpace(r[i]) {
			return i
		}
	}
	return len(r)
}
