package service

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

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
	case reflect.Pointer:
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
	if to.Kind() == reflect.Pointer {
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
