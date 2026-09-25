package service

import (
	"sort"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type (
	authorizationServiceData struct {
		interfaceName string
		requirements  []*authorizationRequirementData
		methods       map[string]*authorizationMethodData
	}

	authorizationRequirementData struct {
		expr       *expr.AuthorizationExpr
		methodName string
		inputRef   string
		inputName  string
	}

	authorizationMethodData struct {
		expr      *expr.MethodExpr
		service   *authorizationServiceData
		scope     *codegen.NameScope
		checkName string
		fieldName string
	}
)

func collectAuthorizationTypes(service *expr.ServiceExpr, scope *codegen.NameScope, seen map[string]struct {
}) []*UserTypeData {
	var types []*UserTypeData
	for _, method := range service.Methods {
		for _, requirement := range authorizationRequirements(method.Authorization) {
			types = append(types, collectTypes(requirement.Input, scope, seen, nil)...)
		}
	}
	return types
}

func authorizationRequirements(a *expr.MethodAuthorizationExpr) []*expr.AuthorizationExpr {
	if a == nil {
		return nil
	}
	var requirements []*expr.AuthorizationExpr
	for _, use := range a.Requirements {
		requirements = append(requirements, use.Requirement)
	}
	for _, c := range a.Cases {
		requirements = append(requirements, authorizationRequirements(c.Authorization)...)
	}
	return requirements
}

func buildAuthorizationData(service *expr.ServiceExpr, data *Data) *authorizationServiceData {
	a := &authorizationServiceData{methods: make(map[string]*authorizationMethodData)}
	seen := make(map[string]bool)
	for _, method := range service.Methods {
		if method.Authorization == nil {
			continue
		}
		md := data.Method(method.Name)
		a.methods[method.Name] = &authorizationMethodData{
			expr: method, service: a, scope: data.Scope,
			checkName: data.Scope.Unique("new" + md.VarName + "AccessCheck"),
			fieldName: "check" + md.VarName,
		}
		for _, requirement := range authorizationRequirements(method.Authorization) {
			if seen[requirement.Name] {
				continue
			}
			seen[requirement.Name] = true
			a.requirements = append(a.requirements, &authorizationRequirementData{expr: requirement})
		}
	}
	if len(a.methods) == 0 {
		return nil
	}
	sort.Slice(a.requirements, func(i, j int) bool {
		return a.requirements[i].expr.Name < a.requirements[j].expr.Name
	})
	a.interfaceName = data.Scope.Unique("AccessAuthorizer")
	methodScope := codegen.NewNameScope()
	for _, requirement := range a.requirements {
		requirement.methodName = methodScope.Unique("Authorize" + codegen.Goify(requirement.expr.Name, true))
		if requirement.expr.Input.Type != expr.Empty {
			pkg := data.Scope.PackageName(codegen.UserTypeLocation(requirement.expr.Input.Type))
			requirement.inputRef = data.Scope.GoFullTypeRef(requirement.expr.Input, pkg)
			requirement.inputName = data.Scope.GoFullTypeName(requirement.expr.Input, pkg)
		}
	}
	return a
}

func methodAuthorizationData(data *Data, name string) *authorizationMethodData {
	if data.Authorization == nil {
		return nil
	}
	return data.Authorization.methods[name]
}

func (a *authorizationMethodData) hasEvaluator() bool {
	return len(authorizationRequirements(a.expr.Authorization)) > 0
}

func (a *authorizationServiceData) requirement(name string) *authorizationRequirementData {
	for _, requirement := range a.requirements {
		if requirement.expr.Name == name {
			return requirement
		}
	}
	panic("validated authorization requirement missing from service data")
}
