package service

import (
	"fmt"
	"sort"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type sharedTypeVisit struct {
	attribute *expr.AttributeExpr
	path      string
}

type sharedTypePromoter struct {
	locations map[string]*codegen.Location
	userTypes map[string][]expr.UserType
	seen      map[sharedTypeVisit]struct{}
}

func promoteSharedTypeLocations(root *expr.RootExpr) {
	if root == nil {
		return
	}
	promoter := newSharedTypePromoter(root)
	promoter.walkRoot(root)
	promoter.apply()
}

func newSharedTypePromoter(root *expr.RootExpr) *sharedTypePromoter {
	promoter := &sharedTypePromoter{
		locations: make(map[string]*codegen.Location),
		userTypes: make(map[string][]expr.UserType),
		seen:      make(map[sharedTypeVisit]struct{}),
	}
	for _, userType := range root.Types {
		promoter.userTypes[userType.ID()] = append(promoter.userTypes[userType.ID()], userType)
	}
	return promoter
}

func (promoter *sharedTypePromoter) walkRoot(root *expr.RootExpr) {
	for _, service := range root.Services {
		for _, serviceError := range service.Errors {
			promoter.walk(serviceError.AttributeExpr, nil)
		}
		for _, method := range service.Methods {
			promoter.walk(method.Payload, nil)
			promoter.walk(method.StreamingPayload, nil)
			promoter.walk(method.Result, nil)
			promoter.walk(method.StreamingResult, nil)
			for _, methodError := range method.Errors {
				promoter.walk(methodError.AttributeExpr, nil)
			}
		}
	}
	for _, userType := range root.Types {
		if _, force := userType.Attribute().Meta["type:generate:force"]; force {
			promoter.walk(&expr.AttributeExpr{Type: userType}, nil)
		}
	}
}

func (promoter *sharedTypePromoter) walk(att *expr.AttributeExpr, inherited *codegen.Location) {
	if att == nil || att.Type == expr.Empty {
		return
	}
	switch dataType := att.Type.(type) {
	case expr.UserType:
		promoter.walkUserType(dataType, inherited)
	case *expr.Object:
		for _, named := range *dataType {
			promoter.walk(named.Attribute, inherited)
		}
	case *expr.Array:
		promoter.walk(dataType.ElemType, inherited)
	case *expr.Map:
		promoter.walk(dataType.KeyType, inherited)
		promoter.walk(dataType.ElemType, inherited)
	case *expr.Union:
		for _, named := range dataType.Values {
			promoter.walk(named.Attribute, inherited)
		}
	}
}

func (promoter *sharedTypePromoter) walkUserType(userType expr.UserType, inherited *codegen.Location) {
	attribute := userType.Attribute()
	promoter.userTypes[userType.ID()] = append(promoter.userTypes[userType.ID()], userType)
	location := codegen.UserTypeLocation(userType)
	if location == nil {
		location = inherited
	}
	path := ""
	if location != nil {
		path = location.RelImportPath
		promoter.recordLocation(userType, location)
	}
	visit := sharedTypeVisit{attribute: attribute, path: path}
	if _, ok := promoter.seen[visit]; ok {
		return
	}
	promoter.seen[visit] = struct{}{}
	promoter.walk(attribute, location)
}

func (promoter *sharedTypePromoter) recordLocation(userType expr.UserType, location *codegen.Location) {
	existing := promoter.locations[userType.ID()]
	if existing != nil && existing.RelImportPath != location.RelImportPath {
		paths := []string{existing.RelImportPath, location.RelImportPath}
		sort.Strings(paths)
		panic(fmt.Sprintf(
			"user type %q is transitively required by shared packages %q and %q; set struct:pkg:path metadata on the type to select its package",
			userType.Name(), paths[0], paths[1],
		))
	}
	promoter.locations[userType.ID()] = location
}

func (promoter *sharedTypePromoter) apply() {
	for id, location := range promoter.locations {
		for _, userType := range promoter.userTypes[id] {
			if codegen.UserTypeLocation(userType) != nil {
				continue
			}
			if userType.Attribute().Meta == nil {
				userType.Attribute().Meta = make(expr.MetaExpr)
			}
			userType.Attribute().Meta["struct:pkg:path"] = []string{location.RelImportPath}
		}
	}
}
