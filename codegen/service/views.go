package service

import (
	"path/filepath"
	"sort"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type viewedType struct {
	// Name is the type name.
	Name string
	// Views is the view data for all views defined in the type.
	Views []*ViewData
}

// ViewsFile returns the views file for the given service which contains
// logic to render result types using the defined views.
func ViewsFile(_ string, service *expr.ServiceExpr, services *ServicesData) *codegen.File {
	svc := services.Get(service.Name)
	if len(svc.projectedTypes) == 0 {
		return nil
	}
	unions := collectViewsUnions(svc)
	sections := make([]codegen.Section, 0, 2+len(svc.viewedResultTypes)+len(svc.projectedTypes)+len(unions))
	sections = append(sections, viewsHeader(service.Name, unions))
	sections = append(sections, viewTypeSections(svc, unions)...)
	sections = append(sections, viewedTypeMapSection(collectViewedTypes(svc)))
	sections = append(sections, viewValidationSections(svc)...)
	path := filepath.Join(codegen.Gendir, svc.PathName, "views", "view.go")
	return &codegen.File{Path: path, Sections: sections}
}

func collectViewsUnions(svc *Data) []*UnionTypeData {
	unionByHash := make(map[string]*UnionTypeData)
	seenUnions := make(map[string]struct{})
	viewLoc := &codegen.Location{RelImportPath: "views"}
	for _, t := range svc.projectedTypes {
		collectViewUnionTypes(&expr.AttributeExpr{Type: t.Type}, svc.ViewScope, viewLoc, unionByHash, seenUnions)
	}
	unions := make([]*UnionTypeData, 0, len(unionByHash))
	for _, u := range unionByHash {
		unions = append(unions, u)
	}
	sort.Slice(unions, func(i, j int) bool {
		return unions[i].Name < unions[j].Name
	})
	return unions
}

func viewsHeader(serviceName string, unions []*UnionTypeData) codegen.Section {
	imports := []*codegen.ImportSpec{
		codegen.LoomImport(""),
		{Path: "unicode/utf8"},
	}
	if len(unions) > 0 {
		imports = append(imports,
			codegen.SimpleImport("encoding/json/jsontext"),
			codegen.NewImport("json", "encoding/json/v2"),
			codegen.SimpleImport("fmt"),
			codegen.SimpleImport("net/url"),
			codegen.LoomNamedImport("http", "loomhttp"),
		)
	}
	return codegen.Header(serviceName+" views", "views", imports)
}

func viewTypeSections(svc *Data, unions []*UnionTypeData) []codegen.Section {
	sections := make([]codegen.Section, 0, len(svc.viewedResultTypes)+len(svc.projectedTypes)+len(unions))
	for _, t := range svc.viewedResultTypes {
		sections = append(sections, userTypeSection("viewed-result-type", t.UserTypeData))
	}
	for _, t := range svc.projectedTypes {
		sections = append(sections, userTypeSection("projected-type", t.UserTypeData))
	}
	for _, u := range unions {
		sections = append(sections, unionTypeSection("projected-union-type", u))
	}
	return sections
}

func collectViewedTypes(svc *Data) []*viewedType {
	var (
		rtdata []*viewedType
		seen   = make(map[string]struct{})
	)
	appendViews := func(views []*ViewData) {
		if len(views) == 0 {
			return
		}
		name := views[0].TypeVarName
		if _, ok := seen[name]; ok {
			return
		}
		rtdata = append(rtdata, &viewedType{Name: name, Views: views})
		seen[name] = struct{}{}
	}
	for _, t := range svc.viewedResultTypes {
		appendViews(t.Views)
	}
	for _, t := range svc.projectedTypes {
		appendViews(t.Views)
	}
	return rtdata
}

func viewValidationSections(svc *Data) []codegen.Section {
	sections := make([]codegen.Section, 0, len(svc.viewedResultTypes)+len(svc.projectedTypes))
	for _, t := range svc.viewedResultTypes {
		sections = append(sections, validateSection("validate-viewed-result-type", t.Validate))
	}
	for _, t := range svc.projectedTypes {
		for _, v := range t.Validations {
			sections = append(sections, validateSection("validate-projected-type", v))
		}
	}
	return sections
}
