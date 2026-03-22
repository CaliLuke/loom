package service

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
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

	addTypeDefSection := newTypeSectionCollector(typeDefSections, typesByPath, seen)
	collectMethodTypeSections(svc, svcPath, addTypeDefSection, seen)
	collectUserTypeSections(svc, svcPath, addTypeDefSection, seen)
	errorTypes := collectErrorTypeSections(svc, svcPath, addTypeDefSection, seen)
	svcSections = append(svcSections, buildErrorSections(errorTypes, svc, svcPath, addTypeDefSection)...)
	svcSections = append(svcSections, buildTypeInitSections(svc)...)

	imports := []*codegen.ImportSpec{
		codegen.SimpleImport("context"),
		codegen.SimpleImport("io"),
		codegen.LoomImport(""),
		codegen.LoomImport("security"),
		codegen.NewImport(svc.ViewsPkg, genpkg+"/"+svcName+"/views"),
	}
	if len(svc.unions) > 0 {
		imports = append(imports,
			codegen.SimpleImport("encoding/json"),
			codegen.SimpleImport("fmt"),
			codegen.SimpleImport("net/url"),
			codegen.LoomNamedImport("http", "loomhttp"),
		)
	}
	header := codegen.Header(service.Name+" service", svc.PkgName, imports)
	def := serviceDefinitionSection(svc)

	files := make([]*codegen.File, 0, 1+len(svc.ServerInterceptors)+len(svc.ClientInterceptors))
	files = append(files, &codegen.File{Path: svcPath, Sections: buildServiceFileSections(typeDefSections[svcPath], header, def, svcSections)})

	files = append(files, InterceptorsFiles(genpkg, service, services)...)
	return appendUserTypeFiles(files, svcPath, typeDefSections, typesByPath, userTypePkgs)
}

func newTypeSectionCollector(typeDefSections map[string]map[string]codegen.Section, typesByPath map[string][]string, seen map[string]struct{}) func(string, string, codegen.Section) {
	return func(path, name string, section codegen.Section) {
		if typeDefSections[path] == nil {
			typeDefSections[path] = make(map[string]codegen.Section)
		}
		typeDefSections[path][name] = section
		typesByPath[path] = append(typesByPath[path], name)
		seen[name] = struct{}{}
	}
}

func collectMethodTypeSections(svc *Data, svcPath string, addTypeDefSection func(string, string, codegen.Section), seen map[string]struct{}) {
	for _, method := range svc.Methods {
		payloadPath := pathWithDefault(method.PayloadLoc, svcPath)
		resultPath := pathWithDefault(method.ResultLoc, svcPath)
		maybeAddNamedTypeSection(method.PayloadDef != "", method.Payload, payloadPath, payloadSection(method), addTypeDefSection, seen)
		maybeAddNamedTypeSection(method.StreamingPayloadDef != "", method.StreamingPayload, payloadPath, streamingPayloadSection(method), addTypeDefSection, seen)
		maybeAddNamedTypeSection(method.ResultDef != "", method.Result, resultPath, resultSection("service-result", method.Result, method.ResultDesc, method.ResultDef), addTypeDefSection, seen)
		maybeAddNamedTypeSection(method.StreamingResultDef != "" && method.StreamingResult != method.Result, method.StreamingResult, resultPath, resultSection("service-streaming-result", method.StreamingResult, method.StreamingResultDesc, method.StreamingResultDef), addTypeDefSection, seen)
	}
}

func collectUserTypeSections(svc *Data, svcPath string, addTypeDefSection func(string, string, codegen.Section), seen map[string]struct{}) {
	for _, userType := range svc.userTypes {
		maybeAddNamedTypeSection(true, userType.VarName, pathWithDefault(userType.Loc, svcPath), userTypeSection("service-user-type", userType), addTypeDefSection, seen)
	}
	for _, union := range svc.unions {
		addTypeDefSection(pathWithDefault(union.Loc, svcPath), "~union:"+union.Name, unionTypeSection("service-union-type", union))
	}
}

func collectErrorTypeSections(svc *Data, svcPath string, addTypeDefSection func(string, string, codegen.Section), seen map[string]struct{}) []*UserTypeData {
	var errorTypes []*UserTypeData
	seenErrs := make(map[string]struct{})
	for _, errorType := range svc.errorTypes {
		if errorType.Type == expr.ErrorResult {
			continue
		}
		if _, ok := seenErrs[errorType.Name]; ok {
			continue
		}
		seenErrs[errorType.Name] = struct{}{}
		maybeAddNamedTypeSection(true, errorType.Name, pathWithDefault(errorType.Loc, svcPath), userTypeSection("error-user-type", errorType), addTypeDefSection, seen)
		errorTypes = append(errorTypes, errorType)
	}
	return errorTypes
}

func buildErrorSections(errorTypes []*UserTypeData, svc *Data, svcPath string, addTypeDefSection func(string, string, codegen.Section)) []codegen.Section {
	sections := make([]codegen.Section, 0, len(errorTypes)+len(svc.errorInits))
	for _, errorType := range errorTypes {
		addTypeDefSection(pathWithDefault(errorType.Loc, svcPath), "|"+errorType.Name, errorSection(errorType))
	}
	for _, errorInit := range svc.errorInits {
		sections = append(sections, errorInitSection(errorInit))
	}
	return sections
}

func buildTypeInitSections(svc *Data) []codegen.Section {
	var sections []codegen.Section
	for _, viewed := range svc.viewedResultTypes {
		sections = append(sections,
			typeInitSection("viewed-result-type-to-service-result-type", viewed.ResultInit),
			typeInitSection("service-result-type-to-viewed-result-type", viewed.Init),
		)
	}
	var helpers []*codegen.TransformFunctionData
	for _, projected := range svc.projectedTypes {
		for _, initData := range projected.TypeInits {
			helpers = codegen.AppendHelpers(helpers, initData.Helpers)
			sections = append(sections, typeInitSection("projected-type-to-service-type", initData))
		}
		for _, projection := range projected.Projections {
			helpers = codegen.AppendHelpers(helpers, projection.Helpers)
			sections = append(sections, typeInitSection("service-type-to-projected-type", projection))
		}
	}
	for _, helper := range helpers {
		sections = append(sections, transformHelperSection("transform-helpers", helper))
	}
	return sections
}

func buildServiceFileSections(typeSections map[string]codegen.Section, header, def codegen.Section, svcSections []codegen.Section) []codegen.Section {
	names := make([]string, 0, len(typeSections))
	for name := range typeSections {
		names = append(names, name)
	}
	sort.Strings(names)

	sections := make([]codegen.Section, 0, 2+len(names)+len(svcSections))
	sections = append(sections, header, def)
	for _, name := range names {
		sections = append(sections, typeSections[name])
	}
	return append(sections, svcSections...)
}

func appendUserTypeFiles(files []*codegen.File, svcPath string, typeDefSections map[string]map[string]codegen.Section, typesByPath map[string][]string, userTypePkgs map[string][]string) []*codegen.File {
	paths := sortedTypePaths(typesByPath)
	for _, path := range paths {
		if path == svcPath {
			continue
		}
		sections, hasUnion := collectUserTypeFileSections(path, typesByPath[path], typeDefSections[path], userTypePkgs)
		if len(sections) == 0 {
			continue
		}
		files = append(files, newUserTypeFile(path, sections, hasUnion))
	}
	return files
}

func sortedTypePaths(typesByPath map[string][]string) []string {
	paths := make([]string, 0, len(typesByPath))
	for path := range typesByPath {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func collectUserTypeFileSections(path string, typeNames []string, typeSections map[string]codegen.Section, userTypePkgs map[string][]string) ([]codegen.Section, bool) {
	names := append([]string(nil), typeNames...)
	sort.Strings(names)
	sections := make([]codegen.Section, 0, len(names))
	hasUnion := false
	for _, name := range names {
		if strings.HasPrefix(name, "~union:") {
			hasUnion = true
		}
		if containsString(userTypePkgs[path], name) {
			continue
		}
		userTypePkgs[path] = append(userTypePkgs[path], name)
		sections = append(sections, typeSections[name])
	}
	return sections, hasUnion
}

func newUserTypeFile(path string, sections []codegen.Section, hasUnion bool) *codegen.File {
	fullRelPath := filepath.Join(codegen.Gendir, path)
	dir, _ := filepath.Split(fullRelPath)
	imports := []*codegen.ImportSpec{
		codegen.SimpleImport("fmt"),
		codegen.LoomImport(""),
	}
	if hasUnion {
		imports = append(imports,
			codegen.SimpleImport("encoding/json"),
			codegen.SimpleImport("net/url"),
			codegen.LoomNamedImport("http", "loomhttp"),
		)
	}
	header := codegen.Header("User types", codegen.Goify(filepath.Base(dir), false), imports)
	return &codegen.File{Path: fullRelPath, Sections: append([]codegen.Section{header}, sections...)}
}

func maybeAddNamedTypeSection(enabled bool, name, path string, section codegen.Section, addTypeDefSection func(string, string, codegen.Section), seen map[string]struct{}) {
	if !enabled {
		return
	}
	if _, ok := seen[name]; ok {
		return
	}
	addTypeDefSection(path, name, section)
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
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
				group.Id("serr").Op(":=").Add(codegen.Expr("loom.NewServiceError")).Call(
					jen.Id("err"),
					jen.Lit(data.ErrName),
					jen.Lit(data.Timeout),
					jen.Lit(data.Temporary),
					jen.Lit(data.Fault),
				)
				group.Add(codegen.Expr(fmt.Sprintf(`loom.WithErrorRemedy(serr, &loom.ErrorRemedy{
	Code:        %q,
	SafeMessage: %q,
	RetryHint:   %q,
})`, data.RemedyCode, data.SafeMessage, data.RetryHint)))
				group.Return(jen.Id("serr"))
				return
			}
			group.Return(
				codegen.Expr("loom.NewServiceError").Call(
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
