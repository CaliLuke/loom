package codegen

import (
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

type (
	// bodyCarrierPlan projects captured codecs into physical storage roles.
	// It contains no encoding policy; the expression capture remains authoritative.
	bodyCarrierPlan struct {
		fields   map[string]fieldCarrierRole
		element  *fieldCarrierRole
		value    *fieldCarrierRole
		branches []*bodyCarrierPlan
	}
	fieldCarrierRole struct {
		json     bool
		branches []fieldCarrierRole
	}
)

func formBodyCarrierPlan(contract *expr.HTTPFormEncoding) *bodyCarrierPlan {
	if contract == nil {
		return nil
	}
	plan := &bodyCarrierPlan{fields: make(map[string]fieldCarrierRole, len(contract.Fields))}
	for name, field := range contract.Fields {
		if !field.Omitted {
			plan.fields[name] = formFieldCarrierRole(field.Codec)
		}
	}
	if contract.Element != nil {
		element := formFieldCarrierRole(*contract.Element)
		plan.element = &element
	}
	for _, branch := range contract.Branches {
		plan.branches = append(plan.branches, formBodyCarrierPlan(branch))
	}
	for _, branch := range contract.TaggedBranches {
		child := formBodyCarrierPlan(branch.Object)
		if branch.Value != nil {
			role := formFieldCarrierRole(branch.Value.Codec)
			child = &bodyCarrierPlan{value: &role}
		}
		plan.branches = append(plan.branches, child)
	}
	return plan
}

func multipartBodyCarrierPlan(contract *expr.HTTPMultipartEncoding) *bodyCarrierPlan {
	if contract == nil || contract.Custom {
		return nil
	}
	plan := &bodyCarrierPlan{fields: make(map[string]fieldCarrierRole, len(contract.Fields))}
	for name, field := range contract.Fields {
		if !field.Omitted {
			plan.fields[name] = multipartFieldCarrierRole(field.Codec)
		}
	}
	for _, branch := range contract.Branches {
		plan.branches = append(plan.branches, multipartBodyCarrierPlan(branch))
	}
	return plan
}

func formFieldCarrierRole(codec httpencoding.Codec) fieldCarrierRole {
	role := fieldCarrierRole{json: codec.Style == httpencoding.JSON}
	for _, branch := range codec.Branches {
		role.branches = append(role.branches, formFieldCarrierRole(branch.Codec))
	}
	return role
}

func multipartFieldCarrierRole(codec httpencoding.MultipartCodec) fieldCarrierRole {
	if codec.StyleCodec != nil {
		return formFieldCarrierRole(*codec.StyleCodec)
	}
	role := fieldCarrierRole{json: codec.Kind == httpencoding.JSONPart}
	for _, branch := range codec.Branches {
		role.branches = append(role.branches, multipartFieldCarrierRole(branch))
	}
	return role
}
