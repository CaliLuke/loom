package service

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dave/jennifer/jen"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/expr"
)

// Files returns the generated files for the given service as well as a map
// indexing user type names by custom path as defined by the "struct:pkg:path"
// metadata. The map is built over each invocation of Files to avoid duplicate
// type definitions.
func Files(genpkg string, service *expr.ServiceExpr, services *ServicesData, userTypePkgs map[string][]string) []*codegen.File {
	svc := services.Get(service.Name)
	svcName := svc.PathName
	svcPath := filepath.Join(codegen.Gendir, svcName, "service.go")
	seen := make(map[string]struct{})
	typeDefSections := make(map[string]map[string]codegen.Section)
	typesByPath := make(map[string][]string)
	svcSections := make([]codegen.Section, 0, 10)

	addTypeDefSection := func(path, name string, section codegen.Section) {
		if typeDefSections[path] == nil {
			typeDefSections[path] = make(map[string]codegen.Section)
		}
		typeDefSections[path][name] = section
		typesByPath[path] = append(typesByPath[path], name)
		seen[name] = struct{}{}
	}

	for _, m := range svc.Methods {
		payloadPath := pathWithDefault(m.PayloadLoc, svcPath)
		resultPath := pathWithDefault(m.ResultLoc, svcPath)
		if m.PayloadDef != "" {
			if _, ok := seen[m.Payload]; !ok {
				addTypeDefSection(payloadPath, m.Payload, payloadSection(m))
			}
		}
		if m.StreamingPayloadDef != "" {
			if _, ok := seen[m.StreamingPayload]; !ok {
				addTypeDefSection(payloadPath, m.StreamingPayload, streamingPayloadSection(m))
			}
		}
		if m.ResultDef != "" {
			if _, ok := seen[m.Result]; !ok {
				addTypeDefSection(resultPath, m.Result, resultSection("service-result", m.Result, m.ResultDesc, m.ResultDef))
			}
		}
		// Generate streaming result type if different from result
		if m.StreamingResultDef != "" && m.StreamingResult != m.Result {
			if _, ok := seen[m.StreamingResult]; !ok {
				addTypeDefSection(resultPath, m.StreamingResult, resultSection("service-streaming-result", m.StreamingResult, m.StreamingResultDesc, m.StreamingResultDef))
			}
		}
	}
	for _, ut := range svc.userTypes {
		if _, ok := seen[ut.VarName]; !ok {
			addTypeDefSection(pathWithDefault(ut.Loc, svcPath), ut.VarName, userTypeSection("service-user-type", ut))
		}
	}
	for _, u := range svc.unions {
		addTypeDefSection(pathWithDefault(u.Loc, svcPath), "~union:"+u.Name, unionTypeSection("service-union-type", u))
	}

	var errorTypes []*UserTypeData
	seenErrs := make(map[string]struct{})
	for _, et := range svc.errorTypes {
		if et.Type == expr.ErrorResult {
			continue
		}
		if _, ok := seenErrs[et.Name]; !ok {
			seenErrs[et.Name] = struct{}{}
			if _, ok := seen[et.Name]; !ok {
				addTypeDefSection(pathWithDefault(et.Loc, svcPath), et.Name, userTypeSection("error-user-type", et))
			}
			errorTypes = append(errorTypes, et)
		}
	}

	for _, et := range errorTypes {
		// Don't override the section created for the error type
		// declaration, make sure the key does not clash with existing
		// type names, make it generated last.
		key := "|" + et.Name
		addTypeDefSection(pathWithDefault(et.Loc, svcPath), key, errorSection(et))
	}
	for _, er := range svc.errorInits {
		svcSections = append(svcSections, errorInitSection(er))
	}

	// transform result type functions
	for _, t := range svc.viewedResultTypes {
		svcSections = append(svcSections,
			typeInitSection("viewed-result-type-to-service-result-type", t.ResultInit),
			typeInitSection("service-result-type-to-viewed-result-type", t.Init))
	}
	var projh []*codegen.TransformFunctionData
	for _, t := range svc.projectedTypes {
		for _, i := range t.TypeInits {
			projh = codegen.AppendHelpers(projh, i.Helpers)
			svcSections = append(svcSections, typeInitSection("projected-type-to-service-type", i))
		}
		for _, i := range t.Projections {
			projh = codegen.AppendHelpers(projh, i.Helpers)
			svcSections = append(svcSections, typeInitSection("service-type-to-projected-type", i))
		}
	}

	for _, h := range projh {
		svcSections = append(svcSections, transformHelperSection("transform-helpers", h))
	}

	imports := []*codegen.ImportSpec{
		codegen.SimpleImport("context"),
		codegen.SimpleImport("io"),
		codegen.GoaImport(""),
		codegen.GoaImport("security"),
		codegen.NewImport(svc.ViewsPkg, genpkg+"/"+svcName+"/views"),
	}
	if len(svc.unions) > 0 {
		imports = append(imports,
			codegen.SimpleImport("encoding/json"),
			codegen.SimpleImport("fmt"),
			codegen.SimpleImport("net/url"),
			codegen.GoaNamedImport("http", "goahttp"),
		)
	}
	header := codegen.Header(service.Name+" service", svc.PkgName, imports)
	def := serviceDefinitionSection(svc)

	// service.go
	var sections []codegen.Section
	{
		names := make([]string, len(typeDefSections[svcPath]))
		i := 0
		for n := range typeDefSections[svcPath] {
			names[i] = n
			i++
		}
		sections = make([]codegen.Section, 0, 2+len(names)+len(svcSections))
		sections = append(sections, header, def)
		sort.Strings(names)
		for _, n := range names {
			sections = append(sections, typeDefSections[svcPath][n])
		}
		sections = append(sections, svcSections...)
	}
	files := []*codegen.File{{Path: svcPath, Sections: sections}}

	// service and client interceptors
	files = append(files, InterceptorsFiles(genpkg, service, services)...)

	// user types
	paths := make([]string, len(typeDefSections))
	i := 0
	for p := range typesByPath {
		paths[i] = p
		i++
	}
	sort.Strings(paths)
	for _, p := range paths {
		if p == svcPath {
			continue
		}
		var secs []codegen.Section
		hasUnion := false
		ts := typesByPath[p]
		sort.Strings(ts)
		for _, name := range ts {
			if strings.HasPrefix(name, "~union:") {
				hasUnion = true
			}
			hasName := false
			for _, n := range userTypePkgs[p] {
				if hasName = n == name; hasName {
					break
				}
			}
			if hasName {
				continue
			}
			userTypePkgs[p] = append(userTypePkgs[p], name)
			secs = append(secs, typeDefSections[p][name])
		}
		if len(secs) == 0 {
			continue
		}
		fullRelPath := filepath.Join(codegen.Gendir, p)
		dir, _ := filepath.Split(fullRelPath)
		imports := []*codegen.ImportSpec{
			codegen.SimpleImport("fmt"),
			codegen.GoaImport(""),
		}
		if hasUnion {
			imports = append(imports,
				codegen.SimpleImport("encoding/json"),
				codegen.SimpleImport("net/url"),
				codegen.GoaNamedImport("http", "goahttp"),
			)
		}
		h := codegen.Header("User types", codegen.Goify(filepath.Base(dir), false), imports)
		sections := append([]codegen.Section{h}, secs...)
		files = append(files, &codegen.File{Path: fullRelPath, Sections: sections})
	}

	return files
}

// dedupeByResult returns a slice of methods where only a single representative
// per unique ResultRef is kept (first occurrence wins). Methods without a
// ResultRef are ignored.
func dedupeByResult(ms []*MethodData) []*MethodData {
	seen := make(map[string]struct{})
	out := make([]*MethodData, 0, len(ms))
	for _, m := range ms {
		key := m.Result
		if key == "" {
			key = m.StreamingResult
		}
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, m)
	}
	return out
}

// AddServiceDataMetaTypeImports Adds all imports defined by struct:field:type from the service expr and the service data
func AddServiceDataMetaTypeImports(header *codegen.SectionTemplate, svcExpr *expr.ServiceExpr, svcData *Data) {
	codegen.AddServiceMetaTypeImports(header, svcExpr)
	for _, ut := range svcData.userTypes {
		codegen.AddImport(header, codegen.GetMetaTypeImports(ut.Type.Attribute())...)
	}
	for _, et := range svcData.errorTypes {
		codegen.AddImport(header, codegen.GetMetaTypeImports(et.Type.Attribute())...)
	}
	for _, t := range svcData.viewedResultTypes {
		codegen.AddImport(header, codegen.GetMetaTypeImports(t.Type.Attribute())...)
	}
	for _, t := range svcData.projectedTypes {
		codegen.AddImport(header, codegen.GetMetaTypeImports(t.Type.Attribute())...)
	}
}

// AddUserTypeImports sets the import paths for the user types defined in the
// service.  User types may be declared in multiple packages when defined with
// the Meta key "struct:pkg:path".
func AddUserTypeImports(genpkg string, header *codegen.SectionTemplate, d *Data) {
	importsByPath := make(map[string]*codegen.ImportSpec)

	initLoc := func(loc *codegen.Location) {
		if loc == nil {
			return
		}
		importsByPath[loc.FilePath] = &codegen.ImportSpec{Name: loc.PackageName(), Path: genpkg + "/" + loc.RelImportPath}
	}

	// Process method-specific locations
	for _, m := range d.Methods {
		initLoc(m.PayloadLoc)
		initLoc(m.ResultLoc)
		for _, l := range m.ErrorLocs {
			initLoc(l)
		}
	}

	// Process service-level types once (not per method)
	for _, ut := range d.userTypes {
		initLoc(ut.Loc)
	}
	for _, et := range d.errorTypes {
		initLoc(et.Loc)
	}

	for _, imp := range importsByPath { // Order does not matter, imports are sorted during formatting.
		codegen.AddImport(header, imp)
		d.UserTypeImports = append(d.UserTypeImports, imp)
	}
}

func typeInitSection(name string, data *InitData) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		codegen.Doc(stmt, data.Description)
		stmt.Add(codegen.Expr("func " + data.Name + "(" + initArgsString(data.Args) + ") " + data.ReturnTypeRef + " {\n" + data.Code + "\n}"))
		stmt.Line()
	})
}

func errorInitSection(data *ErrorInitData) codegen.Section {
	return codegen.NewJenniferSection("error-init-func", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s builds a %s from an error.", data.Name, data.TypeName))
		stmt.Func().Id(data.Name).Params(jen.Id("err").Error()).Add(codegen.TypeRef(data.TypeRef)).BlockFunc(func(group *jen.Group) {
			if data.RemedyCode != "" || data.SafeMessage != "" || data.RetryHint != "" {
				group.Id("serr").Op(":=").Add(codegen.Expr("goa.NewServiceError")).Call(
					jen.Id("err"),
					jen.Lit(data.ErrName),
					jen.Lit(data.Timeout),
					jen.Lit(data.Temporary),
					jen.Lit(data.Fault),
				)
				group.Add(codegen.Expr(fmt.Sprintf(`goa.WithErrorRemedy(serr, &goa.ErrorRemedy{
	Code:        %q,
	SafeMessage: %q,
	RetryHint:   %q,
})`, data.RemedyCode, data.SafeMessage, data.RetryHint)))
				group.Return(jen.Id("serr"))
				return
			}
			group.Return(
				codegen.Expr("goa.NewServiceError").Call(
					jen.Id("err"),
					jen.Lit(data.ErrName),
					jen.Lit(data.Timeout),
					jen.Lit(data.Temporary),
					jen.Lit(data.Fault),
				),
			)
		})
		stmt.Line()
	})
}

func initArgsString(args []*InitArgData) string {
	if len(args) == 0 {
		return ""
	}
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		parts = append(parts, arg.Name+" "+arg.Ref)
	}
	return strings.Join(parts, ", ")
}

func errorName(et *UserTypeData) string {
	obj := expr.AsObject(et.Type)
	if obj != nil {
		for _, att := range *obj {
			if _, ok := att.Attribute.Meta["struct:error:name"]; ok {
				return fmt.Sprintf("e.%s", codegen.GoifyAtt(att.Attribute, att.Name, true))
			}
		}
	}
	// if error type is a custom user type and used by at most one error, then
	// error Finalize should have added "struct:error:name" to the user type
	// attribute's meta.
	if v, ok := et.Type.Attribute().Meta["struct:error:name"]; ok {
		return fmt.Sprintf("%q", v[0])
	}
	return fmt.Sprintf("%q", et.Name)
}

// isJSONRPCSSE returns true if the service uses SSE for JSON-RPC streaming.
// This requires checking the HTTP endpoints in the root expression.
func isJSONRPCSSE(sd *ServicesData, svc *expr.ServiceExpr) bool {
	// Check if service has JSON-RPC
	httpSvc := sd.Root.API.JSONRPC.HTTPExpr.Service(svc.Name)
	if httpSvc == nil {
		return false
	}

	// Check if any JSON-RPC streaming endpoint uses SSE
	for _, e := range httpSvc.HTTPEndpoints {
		if e.MethodExpr.IsStreaming() && e.IsJSONRPC() && e.SSE != nil {
			return true
		}
	}

	return false
}

func pathWithDefault(loc *codegen.Location, def string) string {
	if loc == nil {
		return def
	}
	return loc.FilePath
}
