//nolint:errcheck // Source builders write only to in-memory strings.
package service

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func authorizationFiles(genpkg string, services *ServicesData, svc *Data) []*codegen.File {
	if svc.Authorization == nil {
		return nil
	}
	typeImports := userTypeImports(genpkg, svc)
	imports := make([]*codegen.ImportSpec, 0, 5+len(typeImports)+len(svc.metaTypeImports))
	imports = append(imports,
		codegen.SimpleImport("context"), codegen.SimpleImport("unicode/utf8"),
		codegen.SimpleImport("fmt"), codegen.LoomImport(""), codegen.LoomImport("security"),
	)
	imports = append(imports, typeImports...)
	imports = append(imports, svc.metaTypeImports...)
	fileSvc, imports := services.fileData(svc.Name, imports)
	var b strings.Builder
	writeAuthorizerInterface(&b, fileSvc.Authorization)
	for _, m := range endpointData(fileSvc).Methods {
		if m.Authorization != nil {
			writeAuthorizationCheck(&b, m)
		}
	}
	writeAuthorizationValidators(&b, fileSvc)
	return []*codegen.File{{
		Path:     filepath.Join(codegen.Gendir, svc.PathName, "authorization.go"),
		Sections: []codegen.Section{codegen.Header(svc.Name+" authorization", svc.PkgName, imports), codegen.NewRawSection("authorization", b.String())},
	}, authorizationManifest(svc)}
}

// writeAuthorizerInterface writes the interface the application implements
// to evaluate the service access requirements. Requirement names are design
// text, so their comments go through codegen.LineComment.
func writeAuthorizerInterface(b *strings.Builder, a *authorizationServiceData) {
	if len(a.requirements) == 0 {
		return
	}
	fmt.Fprintf(b, "// %s evaluates application access requirements. Implementations must be read-only\n// and safe to repeat. Nil errors allow execution; all errors prevent it.\ntype %s interface {\n", a.interfaceName, a.interfaceName)
	for _, r := range a.requirements {
		b.WriteString(codegen.LineComment(fmt.Sprintf("%s evaluates %s using current application access facts.", r.methodName, r.expr.Name)))
		fmt.Fprintf(b, "\n%s(context.Context", r.methodName)
		if r.inputRef != "" {
			fmt.Fprintf(b, ", %s", r.inputRef)
		}
		b.WriteString(") error\n")
	}
	b.WriteString("}\n")
}

func writeAuthorizationCheck(b *strings.Builder, m *EndpointMethodData) {
	a := m.Authorization
	params := authorizationCheckParams(m)
	fmt.Fprintf(b, "func %s(%s) security.EndpointCheck {\n", a.checkName, params)
	if a.hasEvaluator() {
		b.WriteString("security.RequireAuthorizer(access)\n")
	}
	for _, scheme := range m.Schemes.DedupeByType() {
		fmt.Fprintf(b, "security.RequireAuthorizer(auth%sFn)\n", scheme.Type)
	}
	b.WriteString("return func(ctx context.Context, req any) (context.Context, error) {\n")
	fmt.Fprintf(b, "ctx = context.WithValue(ctx, loom.ServiceKey, %q)\nctx = context.WithValue(ctx, loom.MethodKey, %q)\n", m.ServiceName, m.Name)
	payload := writeAuthorizationPayload(b, m)
	if payload != "" {
		validation := authorizationValidationCode(a.expr.Payload, a.scope, payload)
		if validation != "" {
			fmt.Fprintf(b, "{\nvar err error\n%s\nif err != nil {\nreturn nil, err\n}\n}\n", validation)
		}
	}
	if len(m.Requirements) > 0 {
		b.WriteString("requestCtx := ctx\n")
		stmt := jen.Empty()
		stmt.BlockFunc(func(g *jen.Group) {
			buildEndpointAuth(g, m, payload)
		})
		fmt.Fprintf(b, "%#v\n", stmt)
		b.WriteString("if err := requestCtx.Err(); err != nil {\nreturn nil, err\n}\nif ctx == nil {\nreturn nil, loom.Fault(\"authentication returned a nil context\")\n}\nif err := ctx.Err(); err != nil {\nreturn nil, err\n}\n")
	}
	writeAuthorizationClassification(b, m, a.expr.Authorization, payload, "", "")
	b.WriteString("return ctx, nil\n}\n}\n")
}

func authorizationCheckParams(m *EndpointMethodData) string {
	var params []string
	if m.Authorization.hasEvaluator() {
		params = append(params, "access "+m.Authorization.service.interfaceName)
	}
	for _, scheme := range m.Schemes.DedupeByType() {
		params = append(params, "auth"+scheme.Type+"Fn security.Auth"+scheme.Type+"Func")
	}
	return strings.Join(params, ", ")
}

func authorizationCheckArgs(m *EndpointMethodData, aggregate bool) string {
	var args []string
	if m.Authorization.hasEvaluator() {
		args = append(args, "access")
	}
	for _, scheme := range m.Schemes.DedupeByType() {
		name := "auth" + scheme.Type + "Fn"
		if aggregate {
			name = "a." + scheme.Type + "Auth"
		}
		args = append(args, name)
	}
	return strings.Join(args, ", ")
}

func writeAuthorizationPayload(b *strings.Builder, m *EndpointMethodData) string {
	payload := payloadVar(m)
	ref := m.PayloadRef
	variable := "p"
	if m.ServerStream != nil && m.ServerStream.EndpointStruct != "" {
		ref = "*" + m.ServerStream.EndpointStruct
		variable = "ep"
	} else if m.SkipRequestBodyEncodeDecode {
		ref = "*" + m.RequestStruct
		variable = "ep"
	}
	if ref == "" {
		return ""
	}
	fmt.Fprintf(b, "%s, ok := req.(%s)\nif !ok {\nreturn nil, loom.InvalidFieldTypeError(\"payload\", req, %q)\n}\n", variable, ref, ref)
	if strings.HasPrefix(ref, "*") {
		writeAuthorizationNilCheck(b, variable, "payload")
	}
	if m.PayloadRef == "" {
		fmt.Fprintf(b, "_ = %s\n", variable)
		return ""
	}
	if variable == "ep" && strings.HasPrefix(m.PayloadRef, "*") {
		writeAuthorizationNilCheck(b, payload, "payload")
	}
	return payload
}

func writeAuthorizationClassification(b *strings.Builder, m *EndpointMethodData, a *expr.MethodAuthorizationExpr, payload, selector, branch string) {
	if len(a.Cases) > 0 {
		writeAuthorizationCases(b, m, a, payload)
		return
	}
	for _, use := range a.Requirements {
		writeAuthorizationRequirement(b, m, use, payload, selector, branch)
	}
}

func writeAuthorizationCases(b *strings.Builder, m *EndpointMethodData, a *expr.MethodAuthorizationExpr, payload string) {
	target, att, pointer := authorizationAccess(b, m, payload, a.Selector, "", "")
	union := expr.AsUnion(att.Type)
	variantTarget := target
	if union != nil {
		target += ".Kind()"
	} else if pointer {
		writeAuthorizationNilCheck(b, target, a.Selector)
		target = "*" + target
	}
	fmt.Fprintf(b, "switch string(%s) {\n", target)
	for _, c := range a.Cases {
		value := c.Value
		if union != nil {
			for _, v := range union.Values {
				if v.Name == c.Value {
					value = expr.UnionVariantTag(v)
				}
			}
		}
		fmt.Fprintf(b, "case %q:\n", value)
		if union != nil {
			for _, v := range union.Values {
				if v.Name == c.Value && expr.IsObject(v.Attribute.Type) {
					writeAuthorizationNilCheck(b, variantTarget+"."+codegen.Goify(v.Name, true), a.Selector)
				}
			}
		}
		writeAuthorizationClassification(b, m, c.Authorization, payload, a.Selector, c.Value)
	}
	b.WriteString("default:\nreturn nil, loom.PermanentError(\"invalid_authorization_variant\", \"Unknown authorization variant\")\n}\n")
}

func writeAuthorizationRequirement(b *strings.Builder, m *EndpointMethodData, use *expr.AuthorizationUseExpr, payload, selector, branch string) {
	r := m.Authorization.service.requirement(use.Requirement.Name)
	b.WriteString("{\nif err := ctx.Err(); err != nil {\nreturn nil, err\n}\n")
	if r.inputRef != "" {
		fmt.Fprintf(b, "input := &%s{}\n", r.inputName)
		for _, binding := range use.Bindings {
			target, _, pointer := authorizationAccess(b, m, payload, binding.Payload, selector, branch)
			input := r.expr.Input.Find(binding.Input)
			inputPointer := r.expr.Input.IsPrimitivePointer(binding.Input, true) || expr.IsObject(input.Type) || (expr.IsUnion(input.Type) && !r.expr.Input.IsRequired(binding.Input))
			if pointer && !inputPointer {
				writeAuthorizationNilCheck(b, target, binding.Payload)
				target = "*" + target
			} else if !pointer && inputPointer {
				target = "&" + target
			}
			fmt.Fprintf(b, "input.%s = %s\n", codegen.GoifyAtt(input, binding.Input, true), target)
		}
		validation := authorizationValidationCode(r.expr.Input, m.Authorization.scope, "input")
		if validation != "" {
			fmt.Fprintf(b, "var err error\n%s\nif err != nil {\nreturn nil, err\n}\n", validation)
		}
	}
	fmt.Fprintf(b, "if err := access.%s(ctx", r.methodName)
	if r.inputRef != "" {
		b.WriteString(", input")
	}
	b.WriteString("); err != nil {\nreturn nil, err\n}\n}\n")
}

func authorizationAccess(b *strings.Builder, m *EndpointMethodData, payload, path, selector, branch string) (string, *expr.AttributeExpr, bool) {
	steps, err := expr.AuthorizationPath(m.Authorization.expr.Payload, path, selector, branch)
	if err != nil {
		panic(err)
	}
	target := payload
	att := m.Authorization.expr.Payload
	pointer := strings.HasPrefix(m.PayloadRef, "*")
	for _, step := range steps {
		if pointer {
			writeAuthorizationNilCheck(b, target, path)
		}
		if step.Union {
			target += "." + codegen.Goify(step.Name, true)
		} else {
			target += "." + codegen.GoifyAtt(step.Attribute, step.Name, true)
		}
		att = step.Attribute
		pointer = expr.IsObject(att.Type) || (!step.Union && (step.Parent.IsPrimitivePointer(step.Name, true) || (expr.IsUnion(att.Type) && !step.Parent.IsRequired(step.Name))))
	}
	if expr.IsUnion(att.Type) && pointer {
		writeAuthorizationNilCheck(b, target, path)
	}
	return target, att, pointer
}

func writeAuthorizationNilCheck(b *strings.Builder, target, path string) {
	fmt.Fprintf(b, "if %s == nil {\nreturn nil, loom.MissingFieldError(%q, \"authorization\")\n}\n", target, path)
}

func authorizationValidationCode(att *expr.AttributeExpr, scope *codegen.NameScope, target string) string {
	ctx := typeContext(scope)
	ctx.ValidationPrefix = "validateAccess"
	if ut, ok := att.Type.(expr.UserType); ok {
		if expr.IsAlias(ut) {
			return codegen.ValidationCode(att, ut, ctx, true, true, false, target)
		}
		return codegen.ValidationCode(ut.Attribute(), ut, ctx, true, false, false, target)
	}
	return codegen.ValidationCode(att, nil, ctx, true, false, false, target)
}

func writeAuthorizationValidators(b *strings.Builder, svc *Data) {
	needed := authorizationValidatorTypes(svc)
	for _, ut := range svc.userTypes {
		name := "validateAccess" + ut.VarName
		if !needed[ut.Type.ID()] || (!expr.IsObject(ut.Type) && !expr.IsArray(ut.Type) && !expr.IsMap(ut.Type)) {
			continue
		}
		validation := authorizationValidationCode(&expr.AttributeExpr{Type: ut.Type}, svc.Scope, "value")
		if validation == "" {
			continue
		}
		fmt.Fprintf(b, "// %s checks the authored constraints before authorization.\nfunc %s(value %s) error {\n", name, name, svc.Scope.GoFullTypeRef(&expr.AttributeExpr{Type: ut.Type}, svc.Scope.PackageName(ut.Loc)))
		if expr.IsObject(ut.Type) {
			b.WriteString("if value == nil {\nreturn loom.MissingFieldError(\"value\", \"authorization\")\n}\n")
		}
		fmt.Fprintf(b, "var err error\n%s\nreturn err\n}\n", validation)
	}
}

// Only request and decision inputs need authorization validators. Result-only
// types and unclassified methods retain their existing generation behavior.
func authorizationValidatorTypes(svc *Data) map[string]bool {
	types := make(map[string]bool)
	roots := make([]*expr.AttributeExpr, 0, len(svc.Authorization.methods)+len(svc.Authorization.requirements))
	for _, method := range svc.Authorization.methods {
		roots = append(roots, method.expr.Payload)
	}
	for _, requirement := range svc.Authorization.requirements {
		roots = append(roots, requirement.expr.Input)
	}
	for _, root := range roots {
		if err := codegen.Walk(root, func(att *expr.AttributeExpr) error {
			if ut, ok := att.Type.(expr.UserType); ok {
				types[ut.ID()] = true
			}
			return nil
		}); err != nil {
			panic(err)
		}
	}
	return types
}
