package codegen

import (
	"slices"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
)

// collectMessages recurses through the attribute to gather all the messages.
func collectMessages(at *expr.AttributeExpr, sd *ServiceData, seen map[string]struct{}) (data []*service.UserTypeData, imports []string) {
	if at == nil {
		return data, imports
	}
	imports = append(imports, collectProtoImports(at, sd)...)
	if expr.IsPrimitive(at.Type) {
		imports = append(imports, collectPrimitiveMessageImports(at, imports)...)
		return data, imports
	}
	collect := func(at *expr.AttributeExpr) ([]*service.UserTypeData, []string) {
		return collectMessages(at, sd, seen)
	}
	switch dt := at.Type.(type) {
	case expr.UserType:
		d, i := collectUserTypeMessages(at, dt, sd, seen, collect)
		data = append(data, d...)
		imports = append(imports, i...)
	case *expr.Object:
		return collectObjectMessages(dt, collect, data, imports)
	case *expr.Array:
		childData, childImports := collect(dt.ElemType)
		return appendCollectedMessages(data, imports, childData, childImports)
	case *expr.Map:
		keyData, keyImports := collect(dt.KeyType)
		data, imports = appendCollectedMessages(data, imports, keyData, keyImports)
		elemData, elemImports := collect(dt.ElemType)
		return appendCollectedMessages(data, imports, elemData, elemImports)
	case *expr.Union:
		return collectUnionMessages(dt, collect, data, imports)
	}
	return data, imports
}

func collectProtoImports(at *expr.AttributeExpr, sd *ServiceData) []string {
	proto := at.Meta["struct:field:proto"]
	if len(proto) <= 1 {
		return nil
	}
	imp := proto[1]
	if protoImportExists(sd.ProtoGoImports, imp) {
		return nil
	}
	if len(proto) > 3 {
		elems := strings.Split(proto[3], "/")
		sd.ProtoGoImports = append(sd.ProtoGoImports, &codegen.ImportSpec{Path: proto[3], Name: elems[len(elems)-1]})
	}
	return []string{imp}
}

func protoImportExists(imports []*codegen.ImportSpec, path string) bool {
	for _, i := range imports {
		if i.Path == path {
			return true
		}
	}
	return false
}

func collectPrimitiveMessageImports(at *expr.AttributeExpr, imports []string) []string {
	if at.Type.Kind() != expr.AnyKind || slices.Contains(imports, "google/protobuf/struct.proto") {
		return nil
	}
	return []string{"google/protobuf/struct.proto"}
}

func collectUserTypeMessages(
	at *expr.AttributeExpr,
	dt expr.UserType,
	sd *ServiceData,
	seen map[string]struct{},
	collect func(*expr.AttributeExpr) ([]*service.UserTypeData, []string),
) ([]*service.UserTypeData, []string) {
	name := protoStructName(at, dt)
	if _, ok := seen[name]; ok {
		return nil, nil
	}
	att := userTypeAttribute(dt)
	seen[name] = struct{}{}
	data := make([]*service.UserTypeData, 0, 1)
	data = append(data, &service.UserTypeData{
		Name:        name,
		VarName:     protoBufMessageName(at, sd.Scope),
		Description: dt.Attribute().Description,
		Def:         protoBufMessageDef(att, sd),
		Ref:         protoBufGoFullTypeRef(at, sd.PkgName, sd.Scope),
		Type:        dt,
	})
	childData, childImports := collect(att)
	return append(data, childData...), childImports
}

func protoStructName(at *expr.AttributeExpr, dt expr.UserType) string {
	name := dt.Name()
	if n := at.Meta["struct:name:proto"]; n != nil {
		name = n[0]
	}
	return name
}

func collectObjectMessages(
	dt *expr.Object,
	collect func(*expr.AttributeExpr) ([]*service.UserTypeData, []string),
	data []*service.UserTypeData,
	imports []string,
) ([]*service.UserTypeData, []string) {
	for _, nat := range *dt {
		childData, childImports := collect(nat.Attribute)
		data, imports = appendCollectedMessages(data, imports, childData, childImports)
	}
	return data, imports
}

func collectUnionMessages(
	dt *expr.Union,
	collect func(*expr.AttributeExpr) ([]*service.UserTypeData, []string),
	data []*service.UserTypeData,
	imports []string,
) ([]*service.UserTypeData, []string) {
	for _, nat := range dt.Values {
		childData, childImports := collect(nat.Attribute)
		data, imports = appendCollectedMessages(data, imports, childData, childImports)
	}
	return data, imports
}

func appendCollectedMessages(
	data []*service.UserTypeData,
	imports []string,
	collectedData []*service.UserTypeData,
	collectedImports []string,
) ([]*service.UserTypeData, []string) {
	data = append(data, collectedData...)
	imports = append(imports, collectedImports...)
	return data, imports
}
