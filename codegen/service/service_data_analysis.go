package service

import (
	"fmt"
	"slices"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// analyze creates the data necessary to render the code of the given service.
// It records the user types needed by the service definition in userTypes.
func (d *ServicesData) analyze(service *expr.ServiceExpr) *Data {
	scope, viewScope, pkgName, viewspkg := newServiceScopes(service)
	state := d.collectServiceAnalysisData(service, scope, viewScope, viewspkg)
	seen := analysisSeenTypes(state.types, state.errTypes)
	wrapRawObjectMethods(service, scope, seen)
	state.types = append(state.types, d.collectForcedServiceTypes(service, scope, seen)...)
	methods, schemes, viewedRTs := d.buildServiceMethods(service, scope, viewScope, viewspkg, state.seenProj, state.seenViewed, state.viewedRTs)
	assignEndpointFields(methods, scope)
	unions := collectServiceUnions(service, state.types, state.errTypes, scope)
	data := newServiceData(d, service, scope, viewScope, pkgName, viewspkg, methods, schemes, state.types, state.projTypes, state.errTypes, state.errorInits, viewedRTs, unions)
	d.Services[service.Name] = data
	return data
}

func newServiceScopes(service *expr.ServiceExpr) (*codegen.NameScope, *codegen.NameScope, string, string) {
	scope := codegen.NewNameScope()
	scope.Unique("Use")
	scope.Unique("websocket")
	viewScope := codegen.NewNameScope()
	pkgName := scope.HashedUnique(service, strings.ToLower(codegen.Goify(service.Name, false)), "svc")
	return scope, viewScope, pkgName, pkgName + "views"
}

type serviceAnalysisState struct {
	types      []*UserTypeData
	errTypes   []*UserTypeData
	errorInits []*ErrorInitData
	projTypes  []*ProjectedTypeData
	viewedRTs  []*ViewedResultTypeData
	seenProj   map[string]*ProjectedTypeData
	seenViewed map[string]*ViewedResultTypeData
}

func (d *ServicesData) collectServiceAnalysisData(service *expr.ServiceExpr, scope, viewScope *codegen.NameScope, viewspkg string) *serviceAnalysisState {
	seen := make(map[string]struct{})
	seenErrors := make(map[string]struct{})
	state := &serviceAnalysisState{
		seenProj:   make(map[string]*ProjectedTypeData),
		seenViewed: make(map[string]*ViewedResultTypeData),
	}
	state.errTypes, state.errorInits = d.collectServiceErrorData(service.Errors, scope, seen, seenErrors, state.errTypes, state.errorInits)
	state.types, state.projTypes, state.errTypes, state.errorInits = d.collectMethodTypeData(service, scope, viewScope, viewspkg, seen, state.seenProj, seenErrors, state.types, state.projTypes, state.errTypes, state.errorInits)
	return state
}

func analysisSeenTypes(types, errTypes []*UserTypeData) map[string]struct{} {
	seen := make(map[string]struct{}, len(types)+len(errTypes))
	for _, t := range types {
		seen[t.Type.ID()] = struct{}{}
	}
	for _, t := range errTypes {
		seen[t.Type.ID()] = struct{}{}
	}
	return seen
}

func (d *ServicesData) collectForcedServiceTypes(service *expr.ServiceExpr, scope *codegen.NameScope, seen map[string]struct{}) []*UserTypeData {
	var types []*UserTypeData
	for _, t := range d.Root.Types {
		svcs, ok := t.Attribute().Meta["type:generate:force"]
		if !ok {
			continue
		}
		att := &expr.AttributeExpr{Type: t}
		if len(svcs) > 0 {
			if slices.Contains(svcs, service.Name) {
				types = append(types, collectTypes(att, scope, seen, nil)...)
			}
			continue
		}
		types = append(types, collectTypes(att, scope, seen, nil)...)
	}
	return types
}

func assignEndpointFields(methods []*MethodData, scope *codegen.NameScope) {
	for _, m := range methods {
		m.AssignEndpointFields(scope)
	}
}

func newServiceData(
	d *ServicesData,
	service *expr.ServiceExpr,
	scope, viewScope *codegen.NameScope,
	pkgName, viewspkg string,
	methods []*MethodData,
	schemes SchemesData,
	types []*UserTypeData,
	projTypes []*ProjectedTypeData,
	errTypes []*UserTypeData,
	errorInits []*ErrorInitData,
	viewedRTs []*ViewedResultTypeData,
	unions []*UnionTypeData,
) *Data {
	desc := service.Description
	if desc == "" {
		desc = fmt.Sprintf("Service is the %s service interface.", service.Name)
	}

	varName := codegen.Goify(service.Name, false)
	data := &Data{
		Name:               service.Name,
		Description:        desc,
		APIName:            d.Root.API.Name,
		APIVersion:         d.Root.API.Version,
		VarName:            varName,
		PathName:           codegen.SnakeCase(varName),
		StructName:         codegen.Goify(service.Name, true),
		PkgName:            pkgName,
		ViewsPkg:           viewspkg,
		Methods:            methods,
		Schemes:            schemes,
		ServerInterceptors: d.collectInterceptors(service, methods, scope, true),
		ClientInterceptors: d.collectInterceptors(service, methods, scope, false),
		Scope:              scope,
		ViewScope:          viewScope,
		errorTypes:         errTypes,
		errorInits:         errorInits,
		userTypes:          types,
		projectedTypes:     projTypes,
		viewedResultTypes:  viewedRTs,
		unions:             unions,
	}
	return data
}

func (d *ServicesData) collectServiceErrorData(
	errors []*expr.ErrorExpr,
	scope *codegen.NameScope,
	seen map[string]struct{},
	seenErrors map[string]struct{},
	errTypes []*UserTypeData,
	errorInits []*ErrorInitData,
) ([]*UserTypeData, []*ErrorInitData) {
	for _, errExpr := range errors {
		errTypes, errorInits = recordServiceError(errExpr, scope, seen, seenErrors, errTypes, errorInits)
	}
	return errTypes, errorInits
}

func (d *ServicesData) collectMethodTypeData(
	service *expr.ServiceExpr,
	scope, viewScope *codegen.NameScope,
	viewspkg string,
	seen map[string]struct{},
	seenProj map[string]*ProjectedTypeData,
	seenErrors map[string]struct{},
	types []*UserTypeData,
	projTypes []*ProjectedTypeData,
	errTypes []*UserTypeData,
	errorInits []*ErrorInitData,
) ([]*UserTypeData, []*ProjectedTypeData, []*UserTypeData, []*ErrorInitData) {
	for _, method := range service.Methods {
		types = append(types, collectMethodUserTypes(method, scope, seen)...)
		if hasResultType(method.Result) {
			projTypes = append(projTypes, collectProjectedTypes(expr.DupAtt(method.Result), method.Result, viewspkg, scope, viewScope, seenProj)...)
		}
		for _, errExpr := range method.Errors {
			errTypes, errorInits = recordServiceError(errExpr, scope, seen, seenErrors, errTypes, errorInits)
		}
	}
	return types, projTypes, errTypes, errorInits
}

func collectMethodUserTypes(method *expr.MethodExpr, scope *codegen.NameScope, seen map[string]struct{}) []*UserTypeData {
	var types []*UserTypeData
	appendTypes := func(att *expr.AttributeExpr) {
		if att == nil {
			return
		}
		var loc *codegen.Location
		if ut, ok := att.Type.(expr.UserType); ok {
			loc = codegen.UserTypeLocation(ut)
			att = ut.Attribute()
		}
		types = append(types, collectTypes(att, scope, seen, loc)...)
	}
	appendTypes(method.Payload)
	appendTypes(method.StreamingPayload)
	appendTypes(method.Result)
	if method.HasMixedResults() {
		appendTypes(method.StreamingResult)
	}
	return types
}

func recordServiceError(
	errExpr *expr.ErrorExpr,
	scope *codegen.NameScope,
	seen map[string]struct{},
	seenErrors map[string]struct{},
	errTypes []*UserTypeData,
	errorInits []*ErrorInitData,
) ([]*UserTypeData, []*ErrorInitData) {
	collected := collectTypes(errExpr.AttributeExpr, scope, seen, nil)
	errTypes = append(errTypes, collected...)
	if ut, ok := errExpr.Type.(expr.UserType); ok {
		for _, t := range collected {
			if t.Type.ID() != ut.ID() {
				continue
			}
			t.RemedyCode = errorRemedyCode(errExpr)
			t.SafeMessage = errorSafeMessage(errExpr)
			t.RetryHint = errorRetryHint(errExpr)
			break
		}
	}
	if expr.IsDefaultErrorResult(errExpr.Type) {
		if _, ok := seenErrors[errExpr.Name]; !ok {
			seenErrors[errExpr.Name] = struct{}{}
			errorInits = append(errorInits, buildErrorInitData(errExpr, scope))
		}
	}
	return errTypes, errorInits
}

func wrapRawObjectMethods(service *expr.ServiceExpr, scope *codegen.NameScope, seen map[string]struct{}) {
	for _, method := range service.Methods {
		name := codegen.Goify(method.Name, true)
		wrapRawObject(method.Payload, name+"Payload", service.Name+"#"+name+"Payload", scope, seen)
		wrapRawObject(method.StreamingPayload, name+"StreamingPayload", service.Name+"#"+name+"StreamingPayload", scope, seen)
		wrapRawObject(method.Result, name+"Result", service.Name+"#"+name+"Result", scope, seen)
		if method.HasMixedResults() {
			wrapRawObject(method.StreamingResult, name+"StreamingResult", service.Name+"#"+name+"StreamingResult", scope, seen)
		}
	}
}

func wrapRawObject(att *expr.AttributeExpr, name, id string, scope *codegen.NameScope, seen map[string]struct{}) {
	if att == nil {
		return
	}
	if _, ok := att.Type.(*expr.Object); ok {
		att.Type = &expr.UserTypeExpr{
			AttributeExpr: expr.DupAtt(att),
			TypeName:      scope.PeekUnique(name),
			UID:           id,
		}
	}
	if ut, ok := att.Type.(expr.UserType); ok {
		seen[ut.ID()] = struct{}{}
	}
}

func (d *ServicesData) buildServiceMethods(
	service *expr.ServiceExpr,
	scope, viewScope *codegen.NameScope,
	viewspkg string,
	seenProj map[string]*ProjectedTypeData,
	seenViewed map[string]*ViewedResultTypeData,
	viewedRTs []*ViewedResultTypeData,
) ([]*MethodData, SchemesData, []*ViewedResultTypeData) {
	methods := make([]*MethodData, len(service.Methods))
	var schemes SchemesData
	for i, methodExpr := range service.Methods {
		method := d.buildMethodData(methodExpr, scope)
		methods[i] = method
		for _, scheme := range method.Schemes {
			schemes = schemes.Append(scheme)
		}
		rt, ok := methodExpr.Result.Type.(*expr.ResultTypeExpr)
		if !ok {
			continue
		}
		view := ""
		if v, ok := methodExpr.Result.Meta.Last(expr.ViewMetaKey); ok {
			view = v
		}
		if vrt, ok := seenViewed[method.Result+"::"+view]; ok {
			method.SetViewedResult(vrt)
			continue
		}
		projected := seenProj[rt.ID()]
		projAtt := &expr.AttributeExpr{Type: projected.Type}
		vrt := buildViewedResultType(methodExpr.Result, projAtt, viewspkg, scope, viewScope)
		if !containsViewedResultType(viewedRTs, vrt) {
			viewedRTs = append(viewedRTs, vrt)
		}
		method.SetViewedResult(vrt)
		seenViewed[vrt.Name+"::"+view] = vrt
	}
	return methods, schemes, viewedRTs
}

func containsViewedResultType(viewed []*ViewedResultTypeData, target *ViewedResultTypeData) bool {
	for _, existing := range viewed {
		if existing.Type.ID() == target.Type.ID() {
			return true
		}
	}
	return false
}
