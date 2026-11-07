package service

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type (
	nullableErrorData struct {
		error *expr.ErrorExpr
		owner *expr.AttributeExpr
	}
	nullableErrorConstructorKey string
)

func nullableErrorAttribute(scope *codegen.NameScope, name string, payload *expr.AttributeExpr) *expr.AttributeExpr {
	location := ""
	if loc := codegen.UserTypeLocation(payload.Type); loc != nil {
		location = loc.RelImportPath
	}
	// The internal nominal identity is separate from the emitted spelling and
	// from authored types. NameScope owns the public name and all collisions.
	identity := "loom-nullable-error:" + name + ":" + location + ":" + payload.Type.Hash()
	owner := &expr.AttributeExpr{Type: &expr.UserTypeExpr{
		TypeName:      identity,
		UID:           identity,
		AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}},
	}}
	scope.HashedUnique(owner.Type, codegen.Goify(name, true)+"Error")
	return owner
}

func nullableErrorConstructor(scope *codegen.NameScope, owner *expr.AttributeExpr) string {
	name := scope.GoTypeName(owner)
	return scope.HashedUnique(nullableErrorConstructorKey(owner.Type.Hash()), "New"+name)
}

func (k nullableErrorConstructorKey) Hash() string {
	return "nullable-error-constructor:" + string(k)
}

func (d *ServicesData) nullableErrorDeclarations(service *expr.ServiceExpr) []*expr.ErrorExpr {
	errors := append([]*expr.ErrorExpr(nil), service.Errors...)
	errors = append(errors, mappedServiceErrors(d.Root, service)...)
	for _, method := range service.Methods {
		errors = append(errors, method.Errors...)
	}
	return errors
}

func (d *ServicesData) allocateNullableErrorSymbols(service *expr.ServiceExpr, scope *codegen.NameScope) {
	for _, failure := range d.nullableErrorDeclarations(service) {
		if expr.IsNullable(failure.AttributeExpr) {
			nullableErrorConstructor(scope, nullableErrorAttribute(scope, failure.Name, failure.AttributeExpr))
		}
	}
}

func (d *ServicesData) collectNullableErrors(service *expr.ServiceExpr, scope *codegen.NameScope) []*nullableErrorData {
	var result []*nullableErrorData
	seen := make(map[string]bool)
	for _, failure := range d.nullableErrorDeclarations(service) {
		if !expr.IsNullable(failure.AttributeExpr) {
			continue
		}
		owner := nullableErrorAttribute(scope, failure.Name, failure.AttributeExpr)
		key := owner.Type.Hash()
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, &nullableErrorData{error: failure, owner: owner})
	}
	return result
}

func nullableErrorSection(data *Data, owner *nullableErrorData) codegen.Section {
	return codegen.NewJenniferSection("service-nullable-error", func(stmt *jen.Statement) {
		stmt.Add(codegen.Expr(renderNullableError(data, owner)))
		stmt.Line()
	})
}

func renderNullableError(data *Data, owner *nullableErrorData) string {
	failure := owner.error
	if expr.IsDefaultErrorResult(failure.Type) {
		return renderNullableDefaultError(data, owner)
	}
	name := data.Scope.GoTypeName(owner.owner)
	ref := data.Scope.GoTypeRef(owner.owner)
	constructor := nullableErrorConstructor(data.Scope, owner.owner)
	pkg := data.attributePackageName(failure.AttributeExpr)
	if pkg == data.PkgName {
		pkg = ""
	}
	payload := data.Scope.GoFullTypeRef(failure.AttributeExpr, pkg)
	var b strings.Builder
	fmt.Fprintf(&b, "// %s owns the %q error independently of its nullable payload.\ntype %s struct {\n payload %s\n}\n", name, failure.Name, name, payload)
	fmt.Fprintf(&b, "// %s constructs the %q error, rejecting a conflicting concrete ErrorName.\nfunc %s(payload %s) (%s, error) {\n e := &%s{payload: payload}\n if _, err := e.Payload(); err != nil {\n return nil, err\n }\n return e, nil\n}\n", constructor, failure.Name, constructor, payload, ref, name)
	fmt.Fprintf(&b, "// Payload returns the authored value after checking its declared error identity.\n// Absent and explicit null remain distinct. A nil owner is invalid.\nfunc (e %s) Payload() (%s, error) {\n if e == nil {\n var zero %s\n return zero, fmt.Errorf(%q)\n }\n", ref, payload, payload, "nil "+name+" error owner")
	if selector := expr.TaggedAttribute(failure.AttributeExpr, "struct:error:name"); selector != "" {
		_, field := failure.FindAttribute(selector)
		access := "value." + codegen.GoifyAtt(field, selector, true)
		condition := "string(" + access + ") != " + fmt.Sprintf("%q", failure.Name)
		if failure.IsPrimitivePointer(selector, true) {
			condition = access + " == nil || string(*" + access + ") != " + fmt.Sprintf("%q", failure.Name)
		}
		fmt.Fprintf(&b, "if value, ok := e.payload.Value(); ok {\n if %s {\n var zero %s\n return zero, fmt.Errorf(%q)\n }\n }\n", condition, payload, "concrete ErrorName does not match declared error "+failure.Name)
	}
	b.WriteString("return e.payload, nil\n}\n")
	fmt.Fprintf(&b, "// LoomErrorName returns the declared identity, including for null payloads.\nfunc (e %s) LoomErrorName() string {\n return %q\n}\n", ref, failure.Name)
	description := failure.Description
	if description == "" {
		description = failure.Name
	}
	fmt.Fprintf(&b, "// Error returns the concrete error message or the declared description.\nfunc (e %s) Error() string {\n if e != nil {\n if value, ok := e.payload.Value(); ok {\n if err, ok := any(&value).(error); ok {\n return err.Error()\n }\n if err, ok := any(value).(error); ok {\n return err.Error()\n }\n }\n }\n return %q\n}\n", ref, description)
	if failure.Remedy != nil {
		fmt.Fprintf(&b, "// LoomErrorRemedy returns the declared remediation guidance.\nfunc (e %s) LoomErrorRemedy() *loom.ErrorRemedy {\n return &loom.ErrorRemedy{Code: %q, SafeMessage: %q, RetryHint: %q}\n}\n", ref, errorRemedyCode(failure), errorSafeMessage(failure), errorRetryHint(failure))
	}
	return b.String()
}
