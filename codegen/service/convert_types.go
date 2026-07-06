package service

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

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
	case reflect.Pointer:
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
