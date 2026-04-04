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
	var (
		types      []*UserTypeData
		errTypes   []*UserTypeData
		errorInits []*ErrorInitData
		projTypes  []*ProjectedTypeData
		viewedRTs  []*ViewedResultTypeData
	)
	scope := codegen.NewNameScope()
	scope.Unique("Use")       // Reserve "Use" for Endpoints struct Use method.
	scope.Unique("websocket") // Reserve "websocket" to avoid collision with gorilla/websocket
	viewScope := codegen.NewNameScope()
	pkgName := scope.HashedUnique(service, strings.ToLower(codegen.Goify(service.Name, false)), "svc")
	viewspkg := pkgName + "views"
	seen := make(map[string]struct{})
	seenErrors := make(map[string]struct{})
	seenProj := make(map[string]*ProjectedTypeData)
	seenViewed := make(map[string]*ViewedResultTypeData)
	errTypes, errorInits = d.collectServiceErrorData(service.Errors, scope, seen, seenErrors, errTypes, errorInits)
	types, projTypes, errTypes, errorInits = d.collectMethodTypeData(service, scope, viewScope, viewspkg, seen, seenProj, seenErrors, types, projTypes, errTypes, errorInits)
	wrapRawObjectMethods(service, scope, seen)

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

	var (
		methods []*MethodData
		schemes SchemesData
	)
	methods, schemes, viewedRTs = d.buildServiceMethods(service, scope, viewScope, viewspkg, seenProj, seenViewed, viewedRTs)

	for _, m := range methods {
		m.EndpointField = scope.Unique(m.VarName+"Endpoint", "")
		if m.HasMixedResults {
			m.StreamEndpointField = scope.Unique(m.VarName+"StreamEndpoint", "")
		}
	}

	unions := collectServiceUnions(service, types, errTypes, scope)

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

	d.Services[service.Name] = data

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
	if errExpr.Type == expr.ErrorResult {
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
			method.ViewedResult = vrt
			continue
		}
		projected := seenProj[rt.ID()]
		projAtt := &expr.AttributeExpr{Type: projected.Type}
		vrt := buildViewedResultType(methodExpr.Result, projAtt, viewspkg, scope, viewScope)
		if !containsViewedResultType(viewedRTs, vrt) {
			viewedRTs = append(viewedRTs, vrt)
		}
		method.ViewedResult = vrt
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
