package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/encodingmeta"
	"github.com/CaliLuke/loom/internal/naming"
)

type (
	// NameScope defines a naming scope.
	NameScope struct {
		names        map[string]string                             // type hash to unique name
		counts       map[string]int                                // raw type name to occurrence count
		families     *nameFamilies                                 // shared declaration ownership, including type policy
		typeNames    func(expr.DataType, *Location) (string, bool) // immutable generated destination names
		typeLocation *Location                                     // physical package of bare nested declarations
		// packages maps the relative import paths of struct:pkg:path
		// packages to the names that qualify their types in the code
		// generated with the scope, when these differ from the package
		// names.
		packages map[string]string
	}

	// Hasher is the interface implemented by the objects that must be
	// scoped.
	Hasher interface {
		// Hash computes a unique instance hash suitable for indexing
		// in a map.
		Hash() string
	}

	// Scoper provides a scope for generating unique names.
	Scoper interface {
		Scope() *NameScope
	}
)

// NewNameScope creates an empty name scope.
func NewNameScope() *NameScope {
	return &NameScope{
		names:    make(map[string]string),
		counts:   make(map[string]int),
		families: &nameFamilies{members: make(map[string][]string)},
	}
}

// NewNameScopeWithPackageNames creates an empty name scope whose type
// references qualify the types of the struct:pkg:path package at each
// relative import path (Location.RelImportPath) of names with the mapped name
// instead of the package name. A generated file that imports such a package
// under an alias renders its code with such a scope.
func NewNameScopeWithPackageNames(names map[string]string) *NameScope {
	s := NewNameScope()
	s.packages = names
	return s
}

// NewNameScopeLike creates an empty name scope that qualifies the types of
// struct:pkg:path packages as s does.
func NewNameScopeLike(s *NameScope) *NameScope {
	if s == nil {
		return NewNameScope()
	}
	result := NewNameScopeWithPackageNames(s.packages)
	result.typeNames = s.typeNames
	result.typeLocation = s.typeLocation
	return result
}

// WithoutPackageNames returns a name scope that shares the names of s but
// qualifies the types of every struct:pkg:path package with the package name.
// Code that derives identifiers from type names uses it, so that the
// identifiers do not depend on the aliases under which the files that use
// them import the packages.
func (s *NameScope) WithoutPackageNames() *NameScope {
	return &NameScope{names: s.names, counts: s.counts, families: s.families, typeNames: s.typeNames, typeLocation: s.typeLocation}
}

// PackageName returns the name that qualifies the types generated at loc in
// the code generated with the scope: the name set with
// NewNameScopeWithPackageNames, or else loc.PackageName(). It returns the
// empty string when loc is nil.
func (s *NameScope) PackageName(loc *Location) string {
	if loc == nil {
		return ""
	}
	if s != nil {
		if name, ok := s.packages[loc.RelImportPath]; ok {
			return name
		}
	}
	return loc.PackageName()
}

// HashedUnique builds the unique name for key using name and - if not unique -
// appending suffix and - if still not unique - a counter value. It returns
// the same value when called multiple times for a key returning the same hash.
func (s *NameScope) HashedUnique(key Hasher, name string, suffix ...string) string {
	if n, ok := s.names[key.Hash()]; ok {
		return n
	}
	name = s.Unique(name, suffix...)
	s.names[key.Hash()] = name
	return name
}

// Unique returns a unique name for the given name. A suffix is appended to the
// name if given name is not unique. If suffixed name is still not unique, a
// counter value is added to the suffixed name until unique.
func (s *NameScope) Unique(name string, suffix ...string) string {
	c, ok := s.counts[name]
	if !ok {
		s.counts[name]++
		return name
	}
	if len(suffix) > 0 {
		name += suffix[0]
		c, ok = s.counts[name]
		if !ok {
			s.counts[name]++
			return name
		}
	}
	for i := c; ; i++ {
		ret := name + strconv.Itoa(i+1)
		if _, ok := s.counts[ret]; !ok {
			s.counts[ret]++
			return ret
		}
	}
}

// PeekUnique returns the name that Unique would return for the same inputs,
// without mutating the scope. The result is never a name already reserved in
// the scope, and repeated calls return the same value until the scope changes.
// PeekUnique does not reserve the result, so distinct inputs may yield the
// same name; callers that need names distinct from each other must reserve
// them with Unique in a scope of their own.
//
// This is useful when synthesizing type names or identifiers that are later
// reserved via Hash-based naming (e.g., GoTypeName/HashedUnique) and therefore
// must not increment the scope counters twice.
func (s *NameScope) PeekUnique(name string, suffix ...string) string {
	c, ok := s.counts[name]
	if !ok {
		return name
	}
	if len(suffix) > 0 {
		name += suffix[0]
		c, ok = s.counts[name]
		if !ok {
			return name
		}
	}
	for i := c; ; i++ {
		ret := name + strconv.Itoa(i+1)
		if _, ok := s.counts[ret]; !ok {
			return ret
		}
	}
}

// GoTypeDef returns the Go code that defines a Go type which matches the data
// structure definition (the part that comes after `type foo`).
//
// ptr if true indicates that the attribute must be stored in a pointer
// (except array and map types which are always non-pointers)
//
// useDefault if true indicates that the attribute must not be a pointer
// if it has a default value.
func (s *NameScope) GoTypeDef(att *expr.AttributeExpr, ptr, useDefault bool) string {
	return s.goTypeDef(att, ptr, useDefault, s.attributePkgName(att))
}

// GoValueTypeDef returns the Go type definition for the concrete value of att
// without wrapping att itself in a presence type. Nested attributes retain
// their own presence semantics.
func (s *NameScope) GoValueTypeDef(att *expr.AttributeExpr, ptr, useDefault bool) string {
	return s.goValueTypeDefWithPkgOverride(att, ptr, useDefault, s.attributePkgName(att), "")
}

// GoTypeDefWithTargetPkg returns the Go type definition string, qualifying any
// user types inside inline structs with the provided target package. This helps
// when generating JSON-RPC client types that embed inline structs referencing
// user types defined in a separate package (e.g., gen/types).
func (s *NameScope) GoTypeDefWithTargetPkg(att *expr.AttributeExpr, ptr, useDefault bool, targetPkg string) string {
	return s.goTypeDefWithPkgOverride(att, ptr, useDefault, "", targetPkg)
}

// goTypeDefWithPkgOverride generates the Go type definition string for the attribute.
// When targetPkg is not empty, user types referenced inside inline structs are
// qualified against targetPkg unless they are defined in a different package,
// in which case their own package is used. When targetPkg is empty, the
// package qualification falls back to pkg and the user type location.
func (s *NameScope) goTypeDefWithPkgOverride(att *expr.AttributeExpr, ptr, useDefault bool, pkg, targetPkg string) string {
	if t, _ := GetMetaType(att); IsExplicitPresenceType(att) && t != "" {
		return t
	}
	if expr.IsNullable(att) {
		return "loom.Nullable[" + s.goValueTypeDefWithPkgOverride(att, ptr, useDefault, pkg, targetPkg) + "]"
	}
	return s.goValueTypeDefWithPkgOverride(att, ptr, useDefault, pkg, targetPkg)
}

func (s *NameScope) goValueTypeDefWithPkgOverride(att *expr.AttributeExpr, ptr, useDefault bool, pkg, targetPkg string) string {
	if location := attributeTypeLocation(att); location != nil {
		s = s.WithTypeNameLocation(location)
	}
	switch actual := att.Type.(type) {
	case expr.Primitive:
		return primitiveTypeDef(att, actual)
	case *expr.Array:
		return "[]" + s.collectionElemTypeDef(actual.ElemType, ptr, useDefault, pkg, targetPkg)
	case *expr.Map:
		return fmt.Sprintf("map[%s]%s",
			s.mapKeyTypeDef(actual.KeyType, ptr, useDefault, pkg, targetPkg),
			s.collectionElemTypeDef(actual.ElemType, ptr, useDefault, pkg, targetPkg),
		)
	case *expr.Union:
		// Unions are generated as named sum-type structs. Refer to the concrete
		// value here; the nullable wrapper, when present, is added by the caller.
		return s.goFullValueTypeName(att, targetPkg, nil)
	case *expr.Object:
		return s.objectTypeDefWithPkgOverride(att, actual, ptr, useDefault, pkg, targetPkg)
	case expr.UserType:
		return s.userTypeDefWithPkgOverride(att, actual, pkg, targetPkg)
	default:
		panic(NewError(nil, att, fmt.Errorf("unknown Go type definition data type %T", actual)))
	}
}

func primitiveTypeDef(att *expr.AttributeExpr, actual expr.Primitive) string {
	if t, _ := GetMetaType(att); t != "" {
		return t
	}
	if actual.Kind() == expr.AnyKind {
		return "loom.JSONValue"
	}
	return GoNativeTypeName(actual)
}

func (s *NameScope) mapKeyTypeDef(att *expr.AttributeExpr, ptr, useDefault bool, pkg, targetPkg string) string {
	if expr.IsAny(att.Type) {
		if metaType, _ := GetMetaType(att); metaType == "" {
			return "any"
		}
	}
	return s.collectionElemTypeDef(att, ptr, useDefault, pkg, targetPkg)
}

func (s *NameScope) collectionElemTypeDef(att *expr.AttributeExpr, ptr, useDefault bool, pkg, targetPkg string) string {
	return collectionElementReference(att, s.goTypeDefWithPkgOverride(att, ptr, useDefault, pkg, targetPkg))
}

// collectionElementReference applies the collection storage contract to a
// value type: ordinary objects use pointers; unions and presence types use values.
func collectionElementReference(att *expr.AttributeExpr, value string) string {
	if expr.IsObject(att.Type) && !IsExplicitPresenceType(att) {
		return "*" + value
	}
	return value
}

func (s *NameScope) objectTypeDefWithPkgOverride(att *expr.AttributeExpr, actual *expr.Object, ptr, useDefault bool, pkg, targetPkg string) string {
	ss := make([]string, 0, 1+len(*actual)+2)
	ss = append(ss, "struct {")
	for _, nat := range att.PhysicalMembers() {
		ss = append(ss, s.objectFieldTypeDef(att, nat.Name, nat.Attribute, ptr, useDefault, pkg, targetPkg))
	}
	if carrier := att.AdditionalMembersCarrier(); carrier != nil {
		// JSON v2 inlines the map as the object's undeclared members.
		ss = append(ss, fmt.Sprintf("\t%s %s `json:\",embed\"`", expr.AdditionalAttributesFieldName,
			s.goTypeDefWithPkgOverride(carrier, ptr, useDefault, pkg, targetPkg)))
	}
	ss = append(ss, "}")
	return strings.Join(ss, "\n")
}

func (s *NameScope) objectFieldTypeDef(parent *expr.AttributeExpr, name string, at *expr.AttributeExpr, ptr, useDefault bool, pkg, targetPkg string) string {
	fn := GoifyAtt(at, name, true)
	tdef := s.goTypeDefWithPkgOverride(at, ptr, useDefault, pkg, targetPkg)
	layout := expr.ServiceFieldLayout(parent, name, at, ptr, useDefault)
	if layout.Pointer {
		tdef = "*" + tdef
	}
	desc := ""
	if at.Description != "" {
		desc = Comment(at.Description) + "\n\t"
	}
	return fmt.Sprintf("\t%s%s %s%s", desc, fn, tdef, attributeTagsForLayout(name, at, layout.JSON))
}

func (s *NameScope) userTypeDefWithPkgOverride(att *expr.AttributeExpr, actual expr.UserType, pkg, targetPkg string) string {
	if actual == expr.Empty {
		return "struct {}"
	}
	prefix := s.userTypePkgPrefix(actual, pkg, targetPkg)
	// Qualification retains the declaration owner's allocated name. The shared
	// value-name resolver also preserves external package names independently
	// of service reservations, without adding a second presence wrapper.
	return s.goFullValueTypeName(att, strings.TrimSuffix(prefix, "."), nil)
}

func (s *NameScope) userTypePkgPrefix(actual expr.UserType, pkg, targetPkg string) string {
	if loc := UserTypeLocation(actual); loc != nil {
		name := s.PackageName(loc)
		if targetPkg != "" {
			if name != targetPkg {
				return name + "."
			}
			return targetPkg + "."
		}
		if name != pkg {
			return name + "."
		}
		return ""
	}
	if targetPkg != "" {
		return targetPkg + "."
	}
	return ""
}

func (s *NameScope) goTypeDef(att *expr.AttributeExpr, ptr, useDefault bool, pkg string) string {
	return s.goTypeDefWithPkgOverride(att, ptr, useDefault, pkg, "")
}

// GoVar returns the Go code that returns the address of a variable of the Go type
// which matches the given attribute type.
func (*NameScope) GoVar(varName string, dt expr.DataType) string {
	// For a raw struct, no need to indirecting
	if isRawStruct(dt) {
		return varName
	}
	return "&" + varName
}

// GoTypeRef returns the Go code that refers to the Go type which matches the
// given attribute type.
func (s *NameScope) GoTypeRef(att *expr.AttributeExpr) string {
	name := s.GoTypeName(att)
	if IsExplicitPresenceType(att) {
		return name
	}
	return goTypeRef(name, att.Type)
}

// GoValueTypeRef returns a reference to the concrete value type of att without
// wrapping att itself in a presence type.
func (s *NameScope) GoValueTypeRef(att *expr.AttributeExpr) string {
	return goTypeRef(s.GoValueTypeName(att), att.Type)
}

// GoTypeRefWithDefaults returns the Go code that refers to the Go type which
// matches the given attribute type. The result of this function differs from
// GoTypeRef when the attribute type is an object (note: not a user type) and
// the reference is thus an inline struct definition. In this case accounting
// for default values may cause child attributes to use non-pointer fields.
func (s *NameScope) GoTypeRefWithDefaults(att *expr.AttributeExpr) string {
	name := s.GoTypeNameWithDefaults(att)
	return goTypeRef(name, att.Type)
}

// GoFullTypeRef returns the Go code that refers to the Go type which matches
// the given attribute type defined in the given package if a user type.
func (s *NameScope) GoFullTypeRef(att *expr.AttributeExpr, pkg string) string {
	return s.goFullTypeRef(att, pkg, nil)
}

// GoFullTypeRefWithPackages returns the same reference as GoFullTypeRef,
// except that it qualifies every user type generated in a struct:pkg:path
// package, including array elements and map keys and values at any depth,
// with pkgName(loc) instead of the package's own name. It lets a file import
// such packages under aliases.
func (s *NameScope) GoFullTypeRefWithPackages(att *expr.AttributeExpr, pkg string, pkgName func(*Location) string) string {
	return s.goFullTypeRef(att, s.pkgWithDefault(att.Type, pkg, pkgName), pkgName)
}

// GoTypeName returns the Go type name of the given attribute type.
func (s *NameScope) GoTypeName(att *expr.AttributeExpr) string {
	return s.GoFullTypeName(att, "")
}

// GoValueTypeName returns the concrete Go type name of att without wrapping
// att itself in a presence type.
func (s *NameScope) GoValueTypeName(att *expr.AttributeExpr) string {
	return s.goFullValueTypeName(att, "", nil)
}

// GoTypeNameWithDefaults returns the Go type name of the given attribute type.
// The result of this function differs from GoTypeName when the attribute type
// is an object (note: not a user type) and the name is thus an inline struct
// definition. In this case accounting for default values may cause child
// attributes to use non-pointer fields.
func (s *NameScope) GoTypeNameWithDefaults(att *expr.AttributeExpr) string {
	if _, ok := att.Type.(*expr.Object); ok {
		return s.GoTypeDef(att, false, true)
	}
	return s.GoTypeName(att)
}

// GoFullTypeName returns the Go type name of the given data type qualified with
// the given package name if applicable and if not the empty string.
func (s *NameScope) GoFullTypeName(att *expr.AttributeExpr, pkg string) string {
	return s.goFullTypeName(att, pkg, nil)
}

// goFullTypeRef implements GoFullTypeRef. pkgName names the packages of
// user types generated in struct:pkg:path packages; nil selects the names of
// the scope, see PackageName.
func (s *NameScope) goFullTypeRef(att *expr.AttributeExpr, pkg string, pkgName func(*Location) string) string {
	name := s.goFullTypeName(att, pkg, pkgName)
	if IsExplicitPresenceType(att) {
		return name
	}
	return goTypeRef(name, att.Type)
}

// goFullTypeName implements GoFullTypeName. pkgName is as in goFullTypeRef.
func (s *NameScope) goFullTypeName(att *expr.AttributeExpr, pkg string, pkgName func(*Location) string) string {
	if t, _ := GetMetaType(att); IsExplicitPresenceType(att) && t != "" {
		return t
	}
	if expr.IsNullable(att) {
		return "loom.Nullable[" + s.goFullValueTypeName(att, pkg, pkgName) + "]"
	}
	return s.goFullValueTypeName(att, pkg, pkgName)
}

func (s *NameScope) goFullValueTypeName(att *expr.AttributeExpr, pkg string, pkgName func(*Location) string) string {
	switch actual := att.Type.(type) {
	case expr.Primitive:
		if t, _ := GetMetaType(att); t != "" {
			return t
		}
		return primitiveTypeDef(att, actual)
	case *expr.Array:
		return "[]" + s.goFullCollectionElemTypeRef(actual.ElemType, s.pkgWithDefault(actual.ElemType.Type, pkg, pkgName), pkgName)
	case *expr.Map:
		return fmt.Sprintf("map[%s]%s",
			s.goFullMapKeyTypeName(actual.KeyType, s.pkgWithDefault(actual.KeyType.Type, pkg, pkgName), pkgName),
			s.goFullCollectionElemTypeRef(actual.ElemType, s.pkgWithDefault(actual.ElemType.Type, pkg, pkgName), pkgName))
	case *expr.Object:
		return s.goValueTypeDefWithPkgOverride(att, false, false, "", "")
	case expr.UserType, *expr.Union:
		return s.goNominalValueTypeName(actual, pkg)
	case expr.CompositeExpr:
		return s.goFullTypeName(actual.Attribute(), s.pkgWithDefault(actual.Attribute().Type, pkg, pkgName), pkgName)
	default:
		panic(NewError(nil, att, fmt.Errorf("unknown collection element data type %T", actual)))
	}
}

// goFullCollectionElemTypeRef preserves the value representation of unions in
// collections while retaining object pointers and explicit presence wrappers.
func (s *NameScope) goFullCollectionElemTypeRef(att *expr.AttributeExpr, pkg string, pkgName func(*Location) string) string {
	return collectionElementReference(att, s.goFullTypeName(att, pkg, pkgName))
}

func (s *NameScope) goFullMapKeyTypeName(att *expr.AttributeExpr, pkg string, pkgName func(*Location) string) string {
	if expr.IsAny(att.Type) {
		if metaType, _ := GetMetaType(att); metaType == "" {
			return "any"
		}
	}
	return s.goFullTypeRef(att, pkg, pkgName)
}

// IsExplicitPresenceType reports whether att uses explicit presence semantics
// and must not be wrapped in an additional pointer.
func IsExplicitPresenceType(att *expr.AttributeExpr) bool {
	if att == nil {
		return false
	}
	return encodingmeta.ExplicitPresence(att.Meta, expr.IsNullable(att), att.Type == expr.Any)
}

// attributePkgName returns the name of the package that defines the type of
// att as selected by the struct:pkg:path metadata of the user type or of att,
// or the empty string if neither sets one. The scope names the package, see
// PackageName.
func (s *NameScope) attributePkgName(att *expr.AttributeExpr) string {
	return s.PackageName(attributeTypeLocation(att))
}

func attributeTypeLocation(att *expr.AttributeExpr) *Location {
	if loc := UserTypeLocation(att.Type); loc != nil {
		return loc
	}
	if p, ok := att.Meta.Last("struct:pkg:path"); ok && p != "" {
		return &Location{RelImportPath: naming.EscapeNonASCII(p)}
	}
	return nil
}

// pkgWithDefault returns the package defining the given type. If the types is a
// user type with "struct:pkg:path" metadata then it returns the corresponding
// value, named by pkgName when not nil and by the scope otherwise, see
// PackageName. Otherwise it returns pkg.
func (s *NameScope) pkgWithDefault(dt expr.DataType, pkg string, pkgName func(*Location) string) string {
	if loc := UserTypeLocation(dt); loc != nil {
		if pkgName != nil {
			return pkgName(loc)
		}
		return s.PackageName(loc)
	}
	return pkg
}

func goTypeRef(name string, dt expr.DataType) string {
	// For a raw struct, no need to dereference
	if isRawStruct(dt) {
		return name
	}
	return "*" + name
}

func isRawStruct(dt expr.DataType) bool {
	if _, ok := dt.(*expr.Object); ok {
		return true
	}
	if expr.IsObject(dt) {
		return false
	}
	if expr.IsUnion(dt) {
		return false
	}
	return true
}
