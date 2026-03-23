package service

import (
	"fmt"
	"go/build"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// convertData contains the info needed to render convert and create functions.
type convertData struct {
	// Name is the name of the function.
	Name string
	// ReceiverTypeRef is a reference to the receiver type.
	ReceiverTypeRef string
	// TypeRef is a reference to the external type.
	TypeRef string
	// TypeName is the name of the external type.
	TypeName string
	//  Code is the function code.
	Code string
}

// ConvertFiles returns multiple files containing conversion and creation functions,
// grouped by target package as specified by struct:pkg:path metadata.
func ConvertFiles(root *expr.RootExpr, service *expr.ServiceExpr, services *ServicesData) ([]*codegen.File, error) {
	svc := services.Get(service.Name)
	conversions, creations := relevantTypeMaps(root, service, svc)

	if len(conversions) == 0 && len(creations) == 0 {
		return nil, nil
	}

	conversionsByPath, creationsByPath, allPaths := groupTypeMapsByPath(conversions, creations, service)

	// Generate a file for each path
	var files []*codegen.File
	for path := range allPaths {
		file, err := generateConvertFileForPath(
			path,
			conversionsByPath[path],
			creationsByPath[path],
			service,
			svc,
		)
		if err != nil {
			return nil, err
		}
		if file != nil {
			files = append(files, file)
		}
	}

	return files, nil
}

// generateConvertFileForPath generates a single convert.go file for the given path
// containing the specified conversions and creations
func generateConvertFileForPath(
	convertPath string,
	conversions []*expr.TypeMap,
	creations []*expr.TypeMap,
	service *expr.ServiceExpr,
	svc *Data,
) (*codegen.File, error) {
	if len(conversions) == 0 && len(creations) == 0 {
		return nil, nil
	}

	convertPkgName := convertPackageName(conversions, creations, svc.PkgName)
	pkgs, err := convertImports(conversions, creations)
	if err != nil {
		return nil, err
	}
	sections := []codegen.Section{
		codegen.Header(service.Name+" service type conversion functions", convertPkgName, pkgs),
	}

	var (
		names      = map[string]struct{}{}
		transFuncs []*codegen.TransformFunctionData
	)

	for _, c := range conversions {
		data, tf, err := buildConvertSectionData(c, svc, convertPkgName, names)
		if err != nil {
			return nil, err
		}
		transFuncs = codegen.AppendHelpers(transFuncs, tf)
		sections = append(sections, convertSection("convert-to", data))
	}

	for _, c := range creations {
		data, tf, err := buildCreateSectionData(c, svc, convertPkgName, names)
		if err != nil {
			return nil, err
		}
		transFuncs = codegen.AppendHelpers(transFuncs, tf)
		sections = append(sections, createSection("create-from", data))
	}

	// Build transformation helper functions section if any.
	seen := make(map[string]struct{})
	for _, tf := range transFuncs {
		if _, ok := seen[tf.Name]; ok {
			continue
		}
		seen[tf.Name] = struct{}{}
		sections = append(sections, transformHelperSection("convert-create-helper", tf))
	}

	return &codegen.File{Path: convertPath, Sections: sections}, nil
}

func commonPath(sep byte, paths ...string) string {
	// Handle special cases.
	switch len(paths) {
	case 0:
		return ""
	case 1:
		return path.Clean(paths[0])
	}

	// Note, we treat string as []byte, not []rune as is often
	// done in Go. (And sep as byte, not rune). This is because
	// most/all supported OS' treat paths as string of non-zero
	// bytes. A filename may be displayed as a sequence of Unicode
	// runes (typically encoded as UTF-8) but paths are
	// not required to be valid UTF-8 or in any normalized form
	// (e.g. "é" (U+00C9) and "é" (U+0065,U+0301) are different
	// file names.
	c := []byte(path.Clean(paths[0]))

	// We add a trailing sep to handle the case where the
	// common prefix directory is included in the path list
	// (e.g. /home/user1, /home/user1/foo, /home/user1/bar).
	// path.Clean will have cleaned off trailing / separators with
	// the exception of the root directory, "/" (in which case we
	// make it "//", but this will get fixed up to "/" bellow).
	c = append(c, sep)

	// Ignore the first path since it's already in c
	for _, v := range paths[1:] {
		// Clean up each path before testing it
		v = path.Clean(v) + string(sep)

		// Find the first non-common byte and truncate c
		if len(v) < len(c) {
			c = c[:len(v)]
		}
		for i := 0; i < len(c); i++ {
			if v[i] != c[i] {
				c = c[:i]
				break
			}
		}
	}

	// Remove trailing non-separator characters and the final separator
	for i := len(c) - 1; i >= 0; i-- {
		if c[i] == sep {
			c = c[:i]
			break
		}
	}

	return string(c)
}

// getPkgImport returns the correct import path of a package.
// It's needed because the "reflect" package provides the binary import path
// ("github.com/CaliLuke/loom/vendor/some/package") for vendored packages
// instead the source import path ("some/package")
func getPkgImport(pkg, cwd string) string {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	gosrc := path.Join(filepath.ToSlash(gopath), "src")
	cwd = filepath.ToSlash(cwd)

	// check for go modules
	if !strings.HasPrefix(cwd, gosrc) {
		return pkg
	}

	pkgpath := path.Join(gosrc, pkg)
	parentpath := commonPath(os.PathSeparator, cwd, pkgpath)

	// check for external packages
	if parentpath == gosrc {
		return pkg
	}

	rootpkg := parentpath[len(gosrc)+1:]

	// check for vendored packages
	vendorPrefix := path.Join(rootpkg, "vendor")
	if strings.HasPrefix(pkg, vendorPrefix) {
		return pkg[len(vendorPrefix)+1:]
	}

	return pkg
}

func getExternalTypeInfo(external any) (string, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}
	pkg := reflect.TypeOf(external)
	pkgImport := getPkgImport(pkg.PkgPath(), cwd)
	alias := strings.Split(pkg.String(), ".")[0]
	return pkgImport, alias, nil
}

// ConvertFile returns the file containing the conversion and creation functions
// if any.
func ConvertFile(root *expr.RootExpr, service *expr.ServiceExpr, services *ServicesData) (*codegen.File, error) {
	svc := services.Get(service.Name)
	conversions, creations := relevantTypeMaps(root, service, svc)
	if len(conversions) == 0 && len(creations) == 0 {
		return nil, nil
	}

	pkgs, err := convertImports(conversions, creations)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(codegen.Gendir, codegen.SnakeCase(service.Name), "convert.go")
	sections := []codegen.Section{
		codegen.Header(service.Name+" service type conversion functions", svc.PkgName, pkgs),
	}

	var (
		names = map[string]struct{}{}

		transFuncs []*codegen.TransformFunctionData
	)

	for _, c := range conversions {
		data, tf, err := buildDefaultConvertSectionData(c, svc, names)
		if err != nil {
			return nil, err
		}
		transFuncs = codegen.AppendHelpers(transFuncs, tf)
		sections = append(sections, convertSection("convert-to", data))
	}

	for _, c := range creations {
		data, tf, err := buildDefaultCreateSectionData(c, svc, names)
		if err != nil {
			return nil, err
		}
		transFuncs = codegen.AppendHelpers(transFuncs, tf)
		sections = append(sections, createSection("create-from", data))
	}

	// Build transformation helper functions section if any.
	seen := make(map[string]struct{})
	for _, tf := range transFuncs {
		if _, ok := seen[tf.Name]; ok {
			continue
		}
		seen[tf.Name] = struct{}{}
		sections = append(sections, transformHelperSection("convert-create-helper", tf))
	}

	return &codegen.File{Path: path, Sections: sections}, nil
}

func convertSection(name string, data convertData) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s creates an instance of %s initialized from t.", data.Name, data.TypeName))
		stmt.Add(codegen.Expr("func (t " + data.ReceiverTypeRef + ") " + data.Name + "() " + data.TypeRef + " {\n" + data.Code + "\nreturn v\n}"))
	})
}

func createSection(name string, data convertData) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s initializes t from the fields of v", data.Name))
		stmt.Add(codegen.Expr("func (t " + data.ReceiverTypeRef + ") " + data.Name + "(v " + data.TypeRef + ") {\n" + data.Code + "\n*t = *temp\n}"))
	})
}

func transformHelperSection(name string, data *codegen.TransformFunctionData) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s builds a value of type %s from a value of type %s.", data.Name, data.ResultTypeRef, data.ParamTypeRef))
		stmt.Add(codegen.Expr("func " + data.Name + "(v " + data.ParamTypeRef + ") " + data.ResultTypeRef + " {\n" + data.Code + "\nreturn res\n}"))
		stmt.Line()
	})
}

// uniquify checks if base is a key of taken and if not returns it. Otherwise
// uniquify appends integers to base starting at 2 and incremented by 1 each
// time a key already exists for the value. uniquify returns the unique value
// and updates taken with it.
func uniquify(base string, taken map[string]struct{}) string {
	name := base
	idx := 2
	_, ok := taken[name]
	for ok {
		name = base + strconv.Itoa(idx)
		idx++
		_, ok = taken[name]
	}
	taken[name] = struct{}{}
	return name
}

type dtRec struct {
	path string
	seen map[string]expr.DataType
}

func appendPath(r dtRec, p string) dtRec {
	r.path += p
	return r
}

// buildDesignType builds a user type that represents the given external type.
// ref is the user type the data type being built is converted to or created
// from. It's used to compute the non-generated type field names and can be nil
// if no matching attribute exists.
func buildDesignType(dt *expr.DataType, t reflect.Type, ref expr.DataType, recs ...dtRec) error {
	// check compatibility
	if ref != nil {
		if err := compatible(ref, t); err != nil {
			return fmt.Errorf("%q: %w", t.Name(), err)
		}
	}

	// handle recursive data structures
	var rec dtRec
	if recs != nil {
		rec = recs[0]
		if s, ok := rec.seen[t.Name()]; ok {
			*dt = s
			return nil
		}
	} else {
		rec.path = "<value>"
		rec.seen = make(map[string]expr.DataType)
	}

	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64, reflect.String:
		*dt = primitiveDesignType(t.Kind())
	case reflect.Slice:
		return buildSliceDesignType(dt, t, ref, rec)
	case reflect.Map:
		return buildMapDesignType(dt, t, ref, rec)
	case reflect.Struct:
		return buildStructDesignType(dt, t, ref, rec)
	case reflect.Ptr:
		rec.path = "*(" + rec.path + ")"
		if err := buildDesignType(dt, t.Elem(), ref, rec); err != nil {
			return err
		}
		if !expr.IsObject(*dt) {
			return fmt.Errorf("%s: only pointer to struct can be converted", rec.path)
		}
	default:
		*dt = expr.Any
	}
	return nil
}

func relevantTypeMaps(root *expr.RootExpr, service *expr.ServiceExpr, svc *Data) ([]*expr.TypeMap, []*expr.TypeMap) {
	return collectRelevantTypeMaps(root.Conversions, service, svc), collectRelevantTypeMaps(root.Creations, service, svc)
}

func collectRelevantTypeMaps(maps []*expr.TypeMap, service *expr.ServiceExpr, svc *Data) []*expr.TypeMap {
	filtered := make([]*expr.TypeMap, 0, len(maps)*3)
	for _, c := range maps {
		filtered = append(filtered, collectMethodTypeMapMatches(c, service.Methods, true)...)
		filtered = append(filtered, collectMethodTypeMapMatches(c, service.Methods, false)...)
		filtered = append(filtered, collectUserTypeMapMatches(c, svc.userTypes)...)
	}
	return filtered
}

func collectMethodTypeMapMatches(tm *expr.TypeMap, methods []*expr.MethodExpr, payload bool) []*expr.TypeMap {
	var matches []*expr.TypeMap
	for _, m := range methods {
		var dt expr.DataType
		if payload {
			dt = m.Payload.Type
		} else {
			dt = m.Result.Type
		}
		if userTypeMatches(tm.User, dt) {
			matches = append(matches, tm)
			break
		}
	}
	return matches
}

func collectUserTypeMapMatches(tm *expr.TypeMap, userTypes []*UserTypeData) []*expr.TypeMap {
	for _, t := range userTypes {
		if tm.User.Name() == t.Name {
			return []*expr.TypeMap{tm}
		}
	}
	return nil
}

func userTypeMatches(expected expr.UserType, dt expr.DataType) bool {
	ut, ok := dt.(expr.UserType)
	return ok && ut.Name() == expected.Name()
}

func groupTypeMapsByPath(conversions, creations []*expr.TypeMap, service *expr.ServiceExpr) (map[string][]*expr.TypeMap, map[string][]*expr.TypeMap, map[string]struct{}) {
	conversionsByPath := make(map[string][]*expr.TypeMap)
	creationsByPath := make(map[string][]*expr.TypeMap)
	allPaths := make(map[string]struct{})
	for _, c := range conversions {
		path := convertFilePath(c.User, service.Name)
		conversionsByPath[path] = append(conversionsByPath[path], c)
		allPaths[path] = struct{}{}
	}
	for _, c := range creations {
		path := convertFilePath(c.User, service.Name)
		creationsByPath[path] = append(creationsByPath[path], c)
		allPaths[path] = struct{}{}
	}
	return conversionsByPath, creationsByPath, allPaths
}

func convertFilePath(user expr.UserType, serviceName string) string {
	if loc := codegen.UserTypeLocation(user); loc != nil {
		return filepath.Join(codegen.Gendir, filepath.Dir(loc.FilePath), "convert.go")
	}
	return filepath.Join(codegen.Gendir, codegen.SnakeCase(serviceName), "convert.go")
}

func convertPackageName(conversions, creations []*expr.TypeMap, fallback string) string {
	first := firstTypeMap(conversions, creations)
	if first == nil {
		return fallback
	}
	if loc := codegen.UserTypeLocation(first.User); loc != nil {
		return loc.PackageName()
	}
	return fallback
}

func firstTypeMap(groups ...[]*expr.TypeMap) *expr.TypeMap {
	for _, group := range groups {
		if len(group) > 0 {
			return group[0]
		}
	}
	return nil
}

func convertImports(conversions, creations []*expr.TypeMap) ([]*codegen.ImportSpec, error) {
	ppm := make(map[string]string)
	for _, c := range append(append([]*expr.TypeMap{}, conversions...), creations...) {
		pkgImport, alias, err := getExternalTypeInfo(c.External)
		if err != nil {
			return nil, err
		}
		ppm[pkgImport] = alias
	}
	pkgs := make([]*codegen.ImportSpec, 0, len(ppm)+2)
	for pp, alias := range ppm {
		pkgs = append(pkgs, &codegen.ImportSpec{Name: alias, Path: pp})
	}
	pkgs = append(pkgs, &codegen.ImportSpec{Path: "context"}, codegen.LoomImport(""))
	return pkgs, nil
}

func buildConvertSectionData(c *expr.TypeMap, svc *Data, convertPkgName string, names map[string]struct{}) (convertData, []*codegen.TransformFunctionData, error) {
	dt, t, err := designTypeFromExternal(c)
	if err != nil {
		return convertData{}, nil, err
	}
	srcCtx := convertSourceContext(c.User, svc.Scope, convertPkgName)
	tgtCtx := externalAttributeContext(t)
	srcAtt := &expr.AttributeExpr{Type: c.User}
	tgtAtt := &expr.AttributeExpr{Type: dt}
	tgtAtt.AddMeta("struct:type:name", dt.Name())
	code, tf, err := codegen.GoTransform(srcAtt, tgtAtt, "t", "v", srcCtx, tgtCtx, "transform", true)
	if err != nil {
		return convertData{}, nil, err
	}
	return convertData{
		Name:            uniquify("ConvertTo"+t.Name(), names),
		ReceiverTypeRef: srcCtx.Scope.Ref(srcAtt, ""),
		TypeName:        t.Name(),
		TypeRef:         externalTypeRef(t, c.User),
		Code:            code,
	}, tf, nil
}

func buildCreateSectionData(c *expr.TypeMap, svc *Data, convertPkgName string, names map[string]struct{}) (convertData, []*codegen.TransformFunctionData, error) {
	dt, t, err := designTypeFromExternal(c)
	if err != nil {
		return convertData{}, nil, err
	}
	srcCtx := externalAttributeContext(t)
	tgtCtx := convertSourceContext(c.User, svc.Scope, convertPkgName)
	tgtAtt := &expr.AttributeExpr{Type: c.User}
	code, tf, err := codegen.GoTransform(&expr.AttributeExpr{Type: dt}, tgtAtt, "v", "temp", srcCtx, tgtCtx, "transform", true)
	if err != nil {
		return convertData{}, nil, err
	}
	return convertData{
		Name:            uniquify("CreateFrom"+t.Name(), names),
		ReceiverTypeRef: tgtCtx.Scope.Ref(tgtAtt, ""),
		TypeRef:         externalTypeRef(t, c.User),
		Code:            code,
	}, tf, nil
}

func buildDefaultConvertSectionData(c *expr.TypeMap, svc *Data, names map[string]struct{}) (convertData, []*codegen.TransformFunctionData, error) {
	dt, t, err := designTypeFromExternal(c)
	if err != nil {
		return convertData{}, nil, err
	}
	srcCtx := typeContext(svc.Scope)
	tgtCtx := externalAttributeContext(t)
	srcAtt := &expr.AttributeExpr{Type: c.User}
	tgtAtt := &expr.AttributeExpr{Type: dt}
	tgtAtt.AddMeta("struct:type:name", dt.Name())
	code, tf, err := codegen.GoTransform(srcAtt, tgtAtt, "t", "v", srcCtx, tgtCtx, "transform", true)
	if err != nil {
		return convertData{}, nil, err
	}
	return convertData{
		Name:            uniquify("ConvertTo"+t.Name(), names),
		ReceiverTypeRef: svc.Scope.GoTypeRef(srcAtt),
		TypeName:        t.Name(),
		TypeRef:         externalTypeRef(t, c.User),
		Code:            code,
	}, tf, nil
}

func buildDefaultCreateSectionData(c *expr.TypeMap, svc *Data, names map[string]struct{}) (convertData, []*codegen.TransformFunctionData, error) {
	dt, t, err := designTypeFromExternal(c)
	if err != nil {
		return convertData{}, nil, err
	}
	srcCtx := externalAttributeContext(t)
	tgtCtx := typeContext(svc.Scope)
	tgtAtt := &expr.AttributeExpr{Type: c.User}
	code, tf, err := codegen.GoTransform(&expr.AttributeExpr{Type: dt}, tgtAtt, "v", "temp", srcCtx, tgtCtx, "transform", true)
	if err != nil {
		return convertData{}, nil, err
	}
	return convertData{
		Name:            uniquify("CreateFrom"+t.Name(), names),
		ReceiverTypeRef: codegen.NewNameScope().GoTypeRef(tgtAtt),
		TypeRef:         externalTypeRef(t, c.User),
		Code:            code,
	}, tf, nil
}

func designTypeFromExternal(c *expr.TypeMap) (expr.DataType, reflect.Type, error) {
	var dt expr.DataType
	t := reflect.TypeOf(c.External)
	if err := buildDesignType(&dt, t, c.User); err != nil {
		return nil, nil, err
	}
	return dt, t, nil
}

func externalAttributeContext(t reflect.Type) *codegen.AttributeContext {
	pkg := t.String()
	if idx := strings.Index(pkg, "."); idx != -1 {
		pkg = pkg[:idx]
	}
	return codegen.NewAttributeContext(false, false, false, pkg, codegen.NewNameScope())
}

func convertSourceContext(user expr.UserType, scope *codegen.NameScope, convertPkgName string) *codegen.AttributeContext {
	if loc := codegen.UserTypeLocation(user); loc != nil {
		srcScope := codegen.NewNameScope()
		srcScope.GoTypeName(&expr.AttributeExpr{Type: user})
		return codegen.NewAttributeContextForConversion(false, false, true, convertPkgName, srcScope)
	}
	return typeContext(scope)
}

func externalTypeRef(t reflect.Type, user expr.UserType) string {
	ref := t.String()
	if expr.IsObject(user) {
		return "*" + ref
	}
	return ref
}

func primitiveDesignType(kind reflect.Kind) expr.DataType {
	switch kind {
	case reflect.Bool:
		return expr.Boolean
	case reflect.Int:
		return expr.Int
	case reflect.Int32:
		return expr.Int32
	case reflect.Int64:
		return expr.Int64
	case reflect.Uint:
		return expr.UInt
	case reflect.Uint32:
		return expr.UInt32
	case reflect.Uint64:
		return expr.UInt64
	case reflect.Float32:
		return expr.Float32
	case reflect.Float64:
		return expr.Float64
	case reflect.String:
		return expr.String
	default:
		return expr.Any
	}
}

func buildSliceDesignType(dt *expr.DataType, t reflect.Type, ref expr.DataType, rec dtRec) error {
	e := t.Elem()
	if e.Kind() == reflect.Uint8 {
		*dt = expr.Bytes
		return nil
	}
	var eref expr.DataType
	if ref != nil {
		eref = expr.AsArray(ref).ElemType.Type
	}
	var elem expr.DataType
	if err := buildDesignType(&elem, e, eref, appendPath(rec, "[0]")); err != nil {
		return fmt.Errorf("%w", err)
	}
	*dt = &expr.Array{ElemType: &expr.AttributeExpr{Type: elem}}
	return nil
}

func buildMapDesignType(dt *expr.DataType, t reflect.Type, ref expr.DataType, rec dtRec) error {
	var kref, vref expr.DataType
	if ref != nil {
		m := expr.AsMap(ref)
		kref = m.KeyType.Type
		vref = m.ElemType.Type
	}
	var kt expr.DataType
	if err := buildDesignType(&kt, t.Key(), kref, appendPath(rec, ".key")); err != nil {
		return fmt.Errorf("%w", err)
	}
	var vt expr.DataType
	if err := buildDesignType(&vt, t.Elem(), vref, appendPath(rec, ".value")); err != nil {
		return fmt.Errorf("%w", err)
	}
	*dt = &expr.Map{KeyType: &expr.AttributeExpr{Type: kt}, ElemType: &expr.AttributeExpr{Type: vt}}
	return nil
}

func buildStructDesignType(dt *expr.DataType, t reflect.Type, ref expr.DataType, rec dtRec) error {
	var oref *expr.Object
	if ref != nil {
		oref = expr.AsObject(ref)
	}
	fields := externalStructFields(t, oref)
	obj := expr.Object(make([]*expr.NamedAttributeExpr, len(fields)))
	ut := &expr.UserTypeExpr{
		AttributeExpr: &expr.AttributeExpr{Type: &obj},
		TypeName:      t.Name(),
		UID:           t.PkgPath() + "#" + t.Name(),
	}
	*dt = ut
	rec.seen[t.Name()] = ut
	required, err := populateStructDesignFields(&obj, fields, t, oref, rec)
	if err != nil {
		return err
	}
	if len(required) > 0 {
		ut.Validation = &expr.ValidationExpr{Required: required}
	}
	return nil
}

func externalStructFields(t reflect.Type, oref *expr.Object) []reflect.StructField {
	var fields []reflect.StructField
	for i := 0; i < t.NumField(); i++ {
		f := t.FieldByIndex([]int{i})
		atn, _ := attributeName(oref, f.Name)
		if shouldIgnoreExternalField(oref, atn) {
			continue
		}
		fields = append(fields, f)
	}
	return fields
}

func shouldIgnoreExternalField(oref *expr.Object, attributeName string) bool {
	if oref == nil {
		return false
	}
	at := oref.Attribute(attributeName)
	if at == nil {
		return false
	}
	for _, key := range []string{"struct:field:external", "struct.field.external"} {
		if m := at.Meta[key]; len(m) > 0 && m[0] == "-" {
			return true
		}
	}
	return false
}

func populateStructDesignFields(obj *expr.Object, fields []reflect.StructField, t reflect.Type, oref *expr.Object, rec dtRec) ([]string, error) {
	var required []string
	for i, f := range fields {
		nat, isRequired, err := buildStructNamedAttribute(f, t.Name(), oref, rec)
		if err != nil {
			return nil, err
		}
		if isRequired {
			required = append(required, nat.Name)
		}
		(*obj)[i] = nat
	}
	return required, nil
}

func buildStructNamedAttribute(f reflect.StructField, typeName string, oref *expr.Object, rec dtRec) (*expr.NamedAttributeExpr, bool, error) {
	recf := appendPath(rec, "."+f.Name)
	atn, fn := attributeName(oref, f.Name)
	aref := matchingAttributeRef(oref, atn)
	fdt, required, err := buildStructFieldDesignType(f, atn, aref, rec, recf)
	if err != nil {
		if strings.HasPrefix(err.Error(), recf.path+":") {
			return nil, false, err
		}
		return nil, false, fmt.Errorf("%q.%s: %w", typeName, f.Name, err)
	}
	name := atn
	if fn != "" {
		name += ":" + fn
	}
	return &expr.NamedAttributeExpr{
		Name:      name,
		Attribute: &expr.AttributeExpr{Type: fdt},
	}, required, nil
}

func matchingAttributeRef(oref *expr.Object, atn string) expr.DataType {
	if oref == nil {
		return nil
	}
	if at := oref.Attribute(atn); at != nil {
		return at.Type
	}
	return nil
}

func buildStructFieldDesignType(f reflect.StructField, attributeName string, aref expr.DataType, rec, recf dtRec) (expr.DataType, bool, error) {
	var (
		fdt      expr.DataType
		required bool
	)
	switch f.Type.Kind() {
	case reflect.Ptr:
		if err := buildDesignType(&fdt, f.Type.Elem(), aref, recf); err != nil {
			return nil, false, err
		}
		if expr.IsArray(fdt) {
			return nil, false, fmt.Errorf("%s: field of type pointer to slice are not supported, use slice instead", rec.path)
		}
		if expr.IsMap(fdt) {
			return nil, false, fmt.Errorf("%s: field of type pointer to map are not supported, use map instead", rec.path)
		}
	case reflect.Struct:
		return nil, false, fmt.Errorf("%s: fields of type struct must use pointers", recf.path)
	default:
		required = isPrimitive(f.Type)
		if err := buildDesignType(&fdt, f.Type, aref, recf); err != nil {
			return nil, false, err
		}
	}
	_ = attributeName
	return fdt, required, nil
}

// attributeName computes the name of the attribute for the given field name and
// object that must contain the matching attribute.
func attributeName(obj *expr.Object, name string) (string, string) {
	if obj == nil {
		return name, ""
	}
	// first look for a "struct:field:external" meta
	for _, nat := range *obj {
		if m := nat.Attribute.Meta["struct:field:external"]; len(m) > 0 {
			if m[0] == name {
				return nat.Name, name
			}
		}
	}
	for _, nat := range *obj { // Deprecated syntax. Only present for backward compatibility.
		if m := nat.Attribute.Meta["struct.field.external"]; len(m) > 0 {
			if m[0] == name {
				return nat.Name, name
			}
		}
	}
	// next look for an exact match
	for _, nat := range *obj {
		if nat.Name == name {
			return name, ""
		}
	}
	// next try to lower case first letter
	ln := strings.ToLower(name[0:1]) + name[1:]
	for _, nat := range *obj {
		if nat.Name == ln {
			return ln, name
		}
	}
	// next look for a lower camel case without acronym
	lcn := codegen.CamelCase(name, false, false)
	for _, nat := range *obj {
		if nat.Name == lcn {
			return lcn, name
		}
	}
	// finally look for a snake case representation
	sn := codegen.SnakeCase(name)
	for _, nat := range *obj {
		if nat.Name == sn {
			return sn, name
		}
	}
	// no match, return field name
	return name, ""
}

// isPrimitive is true if the given kind matches a Loom primitive type.
func isPrimitive(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Bool:
		fallthrough
	case reflect.Int:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		fallthrough
	case reflect.Uint:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		fallthrough
	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		fallthrough
	case reflect.Interface:
		fallthrough
	case reflect.String:
		return true
	case reflect.Slice:
		e := t.Elem()
		if e.Kind() == reflect.Uint8 {
			return true
		}
		return false
	default:
		return false
	}
}

type compRec struct {
	path string
	seen map[string]struct{}
}

func appendCompPath(r compRec, p string) compRec {
	r.path += p
	return r
}

// compatible checks the user and external type definitions map recursively . It
// returns nil if they do, an error otherwise.
func compatible(from expr.DataType, to reflect.Type, recs ...compRec) error {
	// deference if needed
	if to.Kind() == reflect.Ptr {
		return compatible(from, to.Elem(), recs...)
	}
	toName := compatibleTypeName(to)
	rec := compatibleRec(from, toName, recs)
	if rec.seen == nil {
		return nil
	}
	if expr.IsArray(from) {
		return compatibleArray(from, to, rec)
	}
	if expr.IsMap(from) {
		return compatibleMap(from, to, rec)
	}
	if expr.IsObject(from) {
		return compatibleObject(from, to, toName, rec)
	}
	ok, err := compatiblePrimitive(from, to)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return fmt.Errorf("types don't match: type of %s is %s but type of corresponding attribute is %s", rec.path, toName, from.Name())
}

func compatibleTypeName(to reflect.Type) string {
	if name := to.Name(); name != "" {
		return name
	}
	return to.Kind().String()
}

func compatibleRec(from expr.DataType, toName string, recs []compRec) compRec {
	if recs == nil {
		rec := compRec{path: "<value>", seen: make(map[string]struct{})}
		rec.seen[from.Hash()+"-"+toName] = struct{}{}
		return rec
	}
	rec := recs[0]
	if _, ok := rec.seen[from.Hash()+"-"+toName]; ok {
		return compRec{}
	}
	rec.seen[from.Hash()+"-"+toName] = struct{}{}
	return rec
}

func compatibleArray(from expr.DataType, to reflect.Type, rec compRec) error {
	if to.Kind() != reflect.Slice {
		return fmt.Errorf("types don't match: %s must be a slice", rec.path)
	}
	return compatible(expr.AsArray(from).ElemType.Type, to.Elem(), appendCompPath(rec, "[0]"))
}

func compatibleMap(from expr.DataType, to reflect.Type, rec compRec) error {
	if to.Kind() != reflect.Map {
		return fmt.Errorf("types don't match: %s is not a map", rec.path)
	}
	if err := compatible(expr.AsMap(from).ElemType.Type, to.Elem(), appendCompPath(rec, ".value")); err != nil {
		return err
	}
	return compatible(expr.AsMap(from).KeyType.Type, to.Key(), appendCompPath(rec, ".key"))
}

func compatibleObject(from expr.DataType, to reflect.Type, toName string, rec compRec) error {
	if to.Kind() != reflect.Struct {
		return fmt.Errorf("types don't match: %s is a %s, expected a struct", rec.path, toName)
	}
	obj := expr.AsObject(from)
	ma := expr.NewMappedAttributeExpr(&expr.AttributeExpr{Type: obj})
	for _, nat := range *obj {
		fname, field, ok := compatibleFieldLookup(ma, nat, to)
		if fname == "-" {
			continue
		}
		if !ok {
			return fmt.Errorf("types don't match: could not find field %q of external type %q matching attribute %q of type %q", fname, toName, nat.Name, from.Name())
		}
		if err := compatible(nat.Attribute.Type, field.Type, appendCompPath(rec, "."+fname)); err != nil {
			return err
		}
	}
	return nil
}

func compatibleFieldLookup(ma *expr.MappedAttributeExpr, nat *expr.NamedAttributeExpr, to reflect.Type) (string, reflect.StructField, bool) {
	if ef, ok := nat.Attribute.Meta["struct:field:external"]; ok {
		if ef[0] == "-" {
			return "-", reflect.StructField{}, false
		}
		field, found := to.FieldByName(ef[0])
		return ef[0], field, found
	}
	if ef, ok := nat.Attribute.Meta["struct.field.external"]; ok { // Deprecated syntax. Only present for backward compatibility.
		if ef[0] == "-" {
			return "-", reflect.StructField{}, false
		}
		field, found := to.FieldByName(ef[0])
		return ef[0], field, found
	}
	name := codegen.Goify(ma.ElemName(nat.Name), true)
	field, found := to.FieldByName(name)
	return name, field, found
}

func compatiblePrimitive(from expr.DataType, to reflect.Type) (bool, error) {
	if !isPrimitive(to) {
		return false, nil
	}
	var dt expr.DataType
	if err := buildDesignType(&dt, to, nil); err != nil {
		return false, err
	}
	return expr.Equal(dt, from), nil
}
