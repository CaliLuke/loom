package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

// ValidationFunction binds a named validator's declaration and calls to one
// physical parameter type. Diagnostic paths are runtime arguments and are never
// part of its declaration identity.
type ValidationFunction struct {
	// Name is the allocated function name, optionally qualified at call sites.
	Name string
	// Argument is the Go parameter name.
	Argument string
	// Root is the established diagnostic root, or Argument when empty.
	Root string
	// Ref is the physical parameter type allocated by the owning type scope.
	Ref string
	// mode retains the resolved declaration layout independently from a caller.
	mode *AttributeContext
}

// NamedValidationFunction resolves a complete named validator from the physical
// declaration attribute. Occurrence wrappers must not replace this attribute:
// a nullable named declaration accepts its wrapper, while an ordinary object
// declaration accepts a pointer even when a field wraps that object.
func NamedValidationFunction(att *expr.AttributeExpr, ctx *AttributeContext, argument string) ValidationFunction {
	mode := presenceValueContext(ctx, att)
	mode.validationValue = false
	prefix := ctx.ValidationPrefix
	if prefix == "" {
		prefix = "Validate"
	}
	return ValidationFunction{
		Name:     prefix + ctx.Scope.Name(concretePresenceAttribute(att), "", mode.Pointer, mode.UseDefault),
		Argument: argument,
		Ref:      ctx.Scope.Ref(att, ctx.DefaultPkg),
		mode:     mode,
	}
}

// Context derives the physical attribute context used to generate this function's
// predicates. It preserves representation facts and binds diagnostic expressions
// to the function's runtime path independently from its Go value parameter.
func (f ValidationFunction) Context(ctx *AttributeContext) *AttributeContext {
	result := ctx.Dup()
	if f.mode != nil {
		result = f.mode.Dup()
	}
	result.validationRoot = f.Root
	if result.validationRoot == "" {
		result.validationRoot = f.Argument
	}
	result.validationPath, _, _ = f.locals()
	return result
}

// Call renders an invocation. An empty path expression selects the declaration's
// established root; nested occurrences pass their complete diagnostic expression.
func (f ValidationFunction) Call(target, path string) string {
	if path == "" {
		return f.Name + "(" + target + ")"
	}
	return f.Name + "(" + target + ", " + path + ")"
}

// Path renders the function's diagnostic context followed by an authored suffix.
// Suffixes describe occurrences (for example, ".child" or "[*]") and do not
// affect the named declaration or its physical parameter type.
func (f ValidationFunction) Path(suffix string) string {
	path, _, _ := f.locals()
	if suffix == "" {
		return path
	}
	return path + " + " + quoteString(suffix)
}

// Definition renders one complete validator. Optional path components form the
// caller's diagnostic path; calls without components retain the established root.
// This allows entry points and recursive occurrences to share one implementation.
func (f ValidationFunction) Definition(body string) string {
	path, components, part := f.locals()
	root := f.Root
	if root == "" {
		root = f.Argument
	}
	var b strings.Builder
	fmt.Fprintf(&b, "func %s(%s %s, %s ...string) (err error) {\n", f.Name, f.Argument, f.Ref, components)
	// Empty declarations need no path binding. Nonempty declarations share the
	// same binding regardless of which predicates or nested calls consume it.
	if strings.TrimSpace(body) != "" && strings.TrimSpace(body) != "// no validations" {
		fmt.Fprintf(&b, "%s := %q\nif len(%s) > 0 {\n%s = \"\"\nfor _, %s := range %s {\n%s += %s\n}\n}\n", path, root, components, path, part, components, path, part)
	}
	if body != "" {
		b.WriteString(body)
		b.WriteByte('\n')
	}
	b.WriteString("return\n}\n")
	return b.String()
}

// valueTarget returns the physical argument for an unwrapped value. The caller
// supplies the original carrier when the declaration itself owns presence.
func (f ValidationFunction) valueTarget(value, carrier string, unwrapped bool) string {
	if carrier != "" {
		return carrier
	}
	if unwrapped && strings.HasPrefix(f.Ref, "*") {
		return "&" + value
	}
	return value
}

func (f ValidationFunction) locals() (path, components, part string) {
	scope := NewNameScope()
	scope.Unique(f.Argument)
	scope.Unique("err")
	return scope.Unique("validationPath"), scope.Unique("validationContext"), scope.Unique("validationPart")
}

func validationContextExpr(ctx *AttributeContext, context string) string {
	if ctx.validationPath == "" {
		return quoteString(context)
	}
	if context == ctx.validationRoot {
		return ctx.validationPath
	}
	if strings.HasPrefix(context, ctx.validationRoot+".") || strings.HasPrefix(context, ctx.validationRoot+"[") {
		return ctx.validationPath + " + " + quoteString(strings.TrimPrefix(context, ctx.validationRoot))
	}
	return quoteString(context)
}
