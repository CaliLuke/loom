package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/grpc/codegen/internal/transportir"
)

// extractMetadata collects the request/response metadata from the given
// metadata attribute and service type (payload/result).
func extractMetadata(a *expr.MappedAttributeExpr, service *expr.AttributeExpr, scope *codegen.NameScope, services ServicesData) []*MetadataData {
	var metadata []*MetadataData
	ctx := serviceTypeContext("", scope)
	codegen.WalkMappedAttr(a, func(name, elem string, required bool, c *expr.AttributeExpr) error { // nolint: errcheck
		arr := expr.AsArray(c.Type)
		mp := expr.AsMap(c.Type)
		typeRef := scope.GoTypeRef(unalias(c))
		ft := service.Type
		varn := scope.Name(codegen.Goify(name, false))
		fieldName := codegen.Goify(name, true)
		var pointer bool
		if !expr.IsObject(service.Type) {
			fieldName = ""
		} else {
			pointer = service.IsPrimitivePointer(name, true)
			ft = service.Find(name).Type
		}
		if pointer {
			typeRef = "*" + typeRef
		}
		metadata = append(metadata, &MetadataData{
			Name:          elem,
			AttributeName: name,
			Description:   c.Description,
			FieldName:     fieldName,
			FieldType:     ft,
			VarName:       varn,
			Required:      required,
			Type:          c.Type,
			TypeName:      scope.GoTypeName(unalias(c)),
			TypeRef:       typeRef,
			Pointer:       pointer,
			Slice:         arr != nil,
			StringSlice:   arr != nil && arr.ElemType.Type.Kind() == expr.StringKind,
			Map:           mp != nil,
			MapStringSlice: mp != nil &&
				mp.KeyType.Type.Kind() == expr.StringKind &&
				mp.ElemType.Type.Kind() == expr.ArrayKind &&
				expr.AsArray(mp.ElemType.Type).ElemType.Type.Kind() == expr.StringKind,
			Validate:     codegen.AttributeValidationCode(c, nil, ctx, required, false, varn, name),
			DefaultValue: c.DefaultValue,
			Example:      c.Example(services.Root.API.ExampleGenerator),
		})
		return nil
	})
	return metadata
}

func unalias(att *expr.AttributeExpr) *expr.AttributeExpr {
	if ut, ok := att.Type.(expr.UserType); ok {
		if _, ok := ut.Attribute().Type.(expr.Primitive); ok {
			return ut.Attribute()
		}
		return unalias(ut.Attribute())
	}
	return att
}

// serviceTypeContext returns a contextual attribute for service types. Service
// types are Go types and uses non-pointers to hold attributes having default
// values.
func serviceTypeContext(pkg string, scope *codegen.NameScope) *codegen.AttributeContext {
	return codegen.NewAttributeContext(false, false, true, pkg, scope)
}

// resultContext returns the method result attribute and the result context for the given
// endpoint.
func resultContext(endpoint *transportir.Endpoint, sd *ServiceData) (*expr.AttributeExpr, *codegen.AttributeContext) {
	svc := sd.Service
	md := svc.Method(endpoint.Name)
	if md.ViewedResult != nil {
		vresAtt := expr.AsObject(md.ViewedResult.Type).Attribute("projected")
		// return projected type context
		return vresAtt, codegen.NewAttributeContext(true, false, true, svc.ViewsPkg, svc.ViewScope)
	}
	pkg := pkgWithDefault(md.ResultLoc, svc.PkgName)
	return endpoint.Response.Result, serviceTypeContext(pkg, svc.Scope)
}

// pkgWithDefault returns the package name of the given location if not nil, def otherwise.
func pkgWithDefault(loc *codegen.Location, def string) string {
	if loc == nil {
		return def
	}
	return loc.PackageName()
}

// getPrimitive returns the primitive expression if the given expression is an alias to one
func getPrimitive(att *expr.AttributeExpr) *expr.AttributeExpr {
	if ut, ok := att.Type.(*expr.UserTypeExpr); ok {
		if _, ok := ut.Type.(expr.Primitive); ok {
			return ut.AttributeExpr
		}
		return getPrimitive(ut.AttributeExpr)
	}
	return nil
}

// isEmpty returns true if given type is empty.
func isEmpty(dt expr.DataType) bool {
	if dt == expr.Empty {
		return true
	}
	if o := expr.AsObject(dt); o != nil && len(*o) == 0 {
		return true
	}
	return false
}

// hasAnyType recursively checks if the given attribute uses the Any type.
func hasAnyType(att *expr.AttributeExpr) bool {
	if att == nil {
		return false
	}
	if att.Type.Kind() == expr.AnyKind {
		return true
	}
	switch dt := att.Type.(type) {
	case expr.UserType:
		return hasAnyType(dt.Attribute())
	case *expr.Object:
		for _, nat := range *dt {
			if hasAnyType(nat.Attribute) {
				return true
			}
		}
	case *expr.Array:
		return hasAnyType(dt.ElemType)
	case *expr.Map:
		return hasAnyType(dt.KeyType) || hasAnyType(dt.ElemType)
	case *expr.Union:
		for _, nat := range dt.Values {
			if hasAnyType(nat.Attribute) {
				return true
			}
		}
	}
	return false
}
