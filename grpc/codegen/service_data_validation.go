package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// addValidation adds a validation function (if any) for the given user type
// and recurses through the user type adding other validation functions
// (if any).
//
// req if true indicates that the validation is generated for validating
// request (server-side) messages.
func addValidation(att *expr.AttributeExpr, attName string, sd *ServiceData, req bool) *ValidationData {
	ut, ok := att.Type.(expr.UserType)
	if !ok {
		return nil
	}
	vtx := protoBufTypeContext(sd.PkgName, sd.Scope, false)
	// Validation helper names must be derived from the same protobuf-aware
	// scope used by the validation templates so that function declarations
	// and call sites (e.g. Message_) stay in sync regardless of traversal
	// order or reserved-name handling.
	name := vtx.Scope.Name(att, "", vtx.Pointer, vtx.UseDefault)
	ref := protoBufGoFullTypeRef(att, sd.PkgName, sd.Scope)
	kind := validateClient
	if req {
		kind = validateServer
	}
	att = userTypeAttribute(ut)
	for _, n := range sd.validations {
		if n.SrcName == name {
			if n.Kind != kind {
				n.Kind = validateBoth
				collectValidations(att, attName, req, sd)
			}
			return n
		}
	}
	removeMeta(att)
	if def := codegen.ValidationCode(att, ut, vtx, true, expr.IsAlias(att.Type), false, attName); def != "" {
		v := &ValidationData{
			// Validation function names must match the identifiers used by
			// validation templates. The template uses the scoped type name
			// directly (no Goify) to preserve proto-reserved names like Message_.
			Name:    "Validate" + name,
			Def:     def,
			ArgName: attName,
			SrcName: name,
			SrcRef:  ref,
			Kind:    kind,
		}
		sd.validations = append(sd.validations, v)
		collectValidations(att, attName, req, sd)
		return v
	}
	return nil
}

// collectValidations recurses through the attribute and collects the
// validation functions.
//
// req if true indicates that the validations are generated for validating
// request messages.
func collectValidations(att *expr.AttributeExpr, attName string, req bool, sd *ServiceData) {
	collectValidationsR(att, attName, req, sd, make(map[string]struct{}))
}

// collectValidationsR recurses through the attribute and collects validation
// functions with cycle detection using a seen set of user type IDs.
func collectValidationsR(att *expr.AttributeExpr, attName string, req bool, sd *ServiceData, seen map[string]struct{}) {
	gattName := codegen.Goify(attName, false)
	switch dt := att.Type.(type) {
	case expr.UserType:
		if expr.IsPrimitive(dt) {
			// Alias type - validation is generate inline in parent type validation code.
			return
		}
		// Cycle guard: avoid infinite recursion on recursive user types.
		if id := dt.ID(); id != "" {
			if _, ok := seen[id]; ok {
				return
			}
			seen[id] = struct{}{}
		}
		vtx := protoBufTypeContext(sd.PkgName, sd.Scope, false)
		def := codegen.AttributeValidationCode(att, dt, vtx, true, false, gattName, attName)
		// Match helper function identifiers with validation template calls by
		// using the same protobuf-aware scope for the type name. This keeps
		// names like Message_ consistent between declarations and call sites.
		name := vtx.Scope.Name(att, "", vtx.Pointer, vtx.UseDefault)
		kind := validateClient
		if req {
			kind = validateServer
		}
		for _, n := range sd.validations {
			if n.SrcName == name {
				if n.Kind != validateBoth && n.Kind != kind {
					n.Kind = validateBoth
					goto collect
				}
				return
			}
		}
		if def != "" {
			sd.validations = append(sd.validations, &ValidationData{
				// Match helper function identifiers with validation template
				// calls. The template uses the scoped type name directly (no
				// Goify) to preserve proto-reserved names like Message_.
				Name:    "Validate" + name,
				Def:     def,
				ArgName: gattName,
				SrcName: name,
				SrcRef:  protoBufGoFullTypeRef(att, sd.PkgName, sd.Scope),
				Kind:    kind,
			})
		}
	collect:
		att := userTypeAttribute(dt)
		collectValidationsR(att, attName, req, sd, seen)
	case *expr.Object:
		for _, nat := range *dt {
			collectValidationsR(nat.Attribute, nat.Name, req, sd, seen)
		}
	case *expr.Array:
		collectValidationsR(dt.ElemType, "elem", req, sd, seen)
	case *expr.Map:
		collectValidationsR(dt.KeyType, "key", req, sd, seen)
		collectValidationsR(dt.ElemType, "val", req, sd, seen)
	case *expr.Union:
		for _, nat := range dt.Values {
			collectValidationsR(nat.Attribute, nat.Name, req, sd, seen)
		}
	}
}

// userTypeAttribute returns the attribute of the given user type.
func userTypeAttribute(ut expr.UserType) *expr.AttributeExpr {
	att := ut.Attribute()
	if rt, ok := ut.(*expr.ResultTypeExpr); ok {
		if a := unwrapAttr(expr.DupAtt(rt.Attribute())); expr.IsArray(a.Type) {
			// result type collection
			att = &expr.AttributeExpr{Type: expr.AsObject(rt)}
		}
	}
	return att
}
