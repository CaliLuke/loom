package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/codegen/unionnames"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

type (
	// MediaBodyData describes dispatch for one generated HTTP union body.
	MediaBodyData struct {
		// Contracts is the Go literal of the ordered media map keys.
		Contracts string
		// Variants binds each key to the actual emitted union field.
		Variants []*MediaVariantData
	}
	// MediaVariantData is one typed codec branch of a generated HTTP body.
	MediaVariantData struct {
		// Tag is the service union discriminator selected by this media.
		Tag string
		// Field is the generated union storage field.
		Field string
		// Method is the allocated setter and accessor suffix.
		Method string
		// TypeRef is the actual generated branch carrier.
		TypeRef string
		// AllowsNull records explicit JSON null admission for this branch.
		AllowsNull bool
		// Value retains the actual branch carrier's runtime representation plan.
		Value *transportir.ValueTarget
		// Multipart retains the complete captured contract for generated part codecs.
		Multipart *expr.HTTPMultipartEncoding
		// StyledFormFields retain per-occurrence form parameter grammars.
		StyledFormFields []*FormFieldData
		// OrdinaryFormFields retain JSON and repeated codecs beside styled fields.
		OrdinaryFormFields []*FormFieldData
		formBody           *TypeData
		form               *expr.HTTPFormEncoding
	}
)

func attachMediaBodyData(body *TypeData, variants []*transportir.MediaVariant) {
	if body == nil || len(variants) == 0 {
		return
	}
	members := unionnames.Resolve(expr.AsUnion(body.attribute.Type))
	data := &MediaBodyData{}
	contracts := make([]string, len(variants))
	for i, variant := range variants {
		branch := expr.HTTPMediaBranch(body.attribute, expr.AttributeName(variant.Branch))
		if branch == nil {
			panic(fmt.Sprintf("HTTP media branch %q is absent from emitted body %s", variant.Branch, body.Name))
		}
		branchContext := body.context.Dup()
		representation.ApplyLayout(branchContext, branch.Attribute)
		contracts[i] = strconv.Quote(variant.ContentType)
		var styled, ordinary []*FormFieldData
		if variant.Form != nil {
			styled, ordinary = formFieldData(variant.Body, variant.Form.Fields)
		}
		data.Variants = append(data.Variants, &MediaVariantData{
			Tag:                variant.Tag,
			Multipart:          variant.Multipart,
			Field:              members[branch.Name].Field,
			Method:             members[branch.Name].Method,
			TypeRef:            body.context.Scope.Scope().GoTypeRef(branch.Attribute),
			AllowsNull:         expr.AllowsNull(branch.Attribute),
			Value:              representation.BuildValuePlan(variant.Value, branch.Attribute, branchContext, expr.ValuePlanRuntime),
			StyledFormFields:   styled,
			OrdinaryFormFields: ordinary,
			form:               variant.Form,
			formBody:           &TypeData{Name: body.context.Scope.Scope().GoTypeName(branch.Attribute), attribute: branch.Attribute, context: branchContext},
		})
	}
	data.Contracts = "[]string{" + strings.Join(contracts, ", ") + "}"
	body.Media = data
}
