package service

import (
	"crypto/sha256"
	"fmt"
	"maps"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type (
	// projectionConversion binds a semantic selection to its physical Go
	// carriers. Named carrier declarations remain complete and immutable.
	projectionConversion struct {
		types   map[projectionConversionKey]expr.DataType
		pointer bool
	}

	projectionConversionKey struct {
		semantic expr.DataType
		physical expr.DataType
	}
)

const projectionConversionIdentity = "loom:projection:conversion"

func buildProjectionConversion(att, projected *expr.AttributeExpr, view, name, sourceVar, targetVar string, sourceCtx, targetCtx *codegen.AttributeContext, toView bool) (string, []*codegen.TransformFunctionData) {
	selected, err := expr.Project(att.Type.(*expr.ResultTypeExpr), view)
	if err != nil {
		panic(err)
	}
	selection := &expr.AttributeExpr{Type: selected}
	plan := &projectionConversion{types: make(map[projectionConversionKey]expr.DataType)}
	service := plan.attribute(selection, att)
	carriers := &projectionConversion{types: make(map[projectionConversionKey]expr.DataType), pointer: true}
	carrier := carriers.attribute(selection, projected)
	source, target := service, carrier
	if !toView {
		source, target = carrier, service
	}
	names := codegen.NewNameScope()
	allocated := make(map[string]string)
	options := codegen.TransformOptions{HelperName: func(source, target *expr.AttributeExpr) string {
		if _, ok := source.Type.(expr.UserType); !ok {
			return ""
		}
		if _, ok := target.Type.(expr.UserType); !ok {
			return ""
		}
		sourceID := projectionConversionTypeIdentity(source.Type)
		targetID := projectionConversionTypeIdentity(target.Type)
		key := fmt.Sprintf("%d:%s%d:%s", len(sourceID), sourceID, len(targetID), targetID)
		if previous, ok := allocated[key]; ok {
			return previous
		}
		base := "transform" + name + codegen.Goify(sourceCtx.Scope.Name(source, sourceCtx.NamePkg(source), sourceCtx.Pointer, sourceCtx.UseDefault), true) + "To" + codegen.Goify(targetCtx.Scope.Name(target, targetCtx.NamePkg(target), targetCtx.Pointer, targetCtx.UseDefault), true)
		allocated[key] = names.Unique(base)
		return allocated[key]
	}}
	code, helpers, err := codegen.GoTransform(source, target, sourceVar, targetVar, sourceCtx, targetCtx, "transform", true, options)
	if err != nil {
		panic(err)
	}
	if expr.IsObject(source.Type) {
		code = "if " + sourceVar + " == nil {\n\treturn nil\n}\n" + code
	}
	return code + "\nreturn " + targetVar, helpers
}

func projectionConversionTypeIdentity(dt expr.DataType) string {
	if user, ok := dt.(expr.UserType); ok {
		if identity, ok := user.Attribute().Meta.Last(projectionConversionIdentity); ok {
			return identity
		}
	}
	return expr.Hash(dt, false, false, false)
}

func (p *projectionConversion) attribute(semantic, physical *expr.AttributeExpr) *expr.AttributeExpr {
	copy := *physical
	copy.Type = p.typ(semantic.Type, physical.Type)
	if _, named := physical.Type.(expr.UserType); named && p.pointer && expr.IsObject(physical.Type) {
		copy.Validation = projectionCarrierValidation(physical)
	}
	return &copy
}

func (p *projectionConversion) typ(semantic, physical expr.DataType) expr.DataType {
	key := projectionConversionKey{semantic: semantic, physical: physical}
	if prior, ok := p.types[key]; ok {
		return prior
	}
	if user, ok := physical.(expr.UserType); ok {
		selected := semantic.(expr.UserType)
		attribute := *user.Attribute()
		if p.pointer && expr.IsObject(physical) {
			attribute.Validation = projectionCarrierValidation(user.Attribute())
		}
		attribute.Meta = maps.Clone(attribute.Meta)
		if attribute.Meta == nil {
			attribute.Meta = make(expr.MetaExpr)
		}
		identity := fmt.Sprintf("%x", sha256.Sum256([]byte(expr.Hash(semantic, false, false, false)+"\x00"+expr.Hash(physical, false, false, false))))
		attribute.Meta[projectionConversionIdentity] = []string{identity}
		copy := user.Dup(&attribute)
		// DupAtt memoizes by declaration ID. Distinct selected shapes sharing
		// a physical name must not collapse when the transformer copies them.
		switch actual := copy.(type) {
		case *expr.ResultTypeExpr:
			actual.Identifier = identity
		case *expr.UserTypeExpr:
			actual.UID = identity
		}
		p.types[key] = copy
		attribute.Type = p.typ(selected.Attribute().Type, user.Attribute().Type)
		return copy
	}
	switch actual := physical.(type) {
	case *expr.Object:
		selected := expr.AsObject(semantic)
		copy := &expr.Object{}
		p.types[key] = copy
		for _, field := range *selected {
			physicalField := actual.Attribute(field.Name)
			if physicalField == nil {
				panic("selected projection field missing from physical carrier: " + field.Name)
			}
			copy.Set(field.Name, p.attribute(field.Attribute, physicalField))
		}
		return copy
	case *expr.Array:
		selected := expr.AsArray(semantic)
		copy := &expr.Array{NonNullableElems: actual.NonNullableElems}
		p.types[key] = copy
		copy.ElemType = p.attribute(selected.ElemType, actual.ElemType)
		return copy
	case *expr.Map:
		selected := expr.AsMap(semantic)
		copy := &expr.Map{}
		p.types[key] = copy
		copy.KeyType = p.attribute(selected.KeyType, actual.KeyType)
		copy.ElemType = p.attribute(selected.ElemType, actual.ElemType)
		return copy
	case *expr.Union:
		selected := expr.AsUnion(semantic)
		copy := *actual
		copy.Values = make([]*expr.NamedAttributeExpr, len(actual.Values))
		p.types[key] = &copy
		for i, branch := range actual.Values {
			copy.Values[i] = &expr.NamedAttributeExpr{Name: branch.Name, Attribute: p.attribute(selected.Values[i].Attribute, branch.Attribute)}
		}
		return &copy
	default:
		return physical
	}
}

// projectionCarrierValidation retains required union value layout in named
// carriers; other fields remain optional until selected-view validation. Inline
// objects retain physical requiredness because it also determines their Go tags.
func projectionCarrierValidation(att *expr.AttributeExpr) *expr.ValidationExpr {
	validation := &expr.ValidationExpr{}
	if att.Validation != nil {
		validation = att.Validation.Dup()
	}
	validation.Required = nil
	for _, field := range *expr.AsObject(att.Type) {
		if expr.IsUnion(field.Attribute.Type) && att.IsRequired(field.Name) {
			validation.Required = append(validation.Required, field.Name)
		}
	}
	return validation
}
