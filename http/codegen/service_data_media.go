package codegen

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/codegen"
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
		// Field is the generated union field and setter suffix.
		Field string
		// TypeRef is the actual generated branch carrier.
		TypeRef string
		// AllowsNull records explicit JSON null admission for this branch.
		AllowsNull bool
		// Value retains the actual branch carrier's runtime representation plan.
		Value *transportir.ValueTarget
		// Parts is the Go literal of per-field multipart content types.
		Parts string
	}
)

func attachMediaBodyData(body *TypeData, variants []*transportir.MediaVariant) {
	if body == nil || len(variants) == 0 {
		return
	}
	data := &MediaBodyData{}
	contracts := make([]string, len(variants))
	for i, variant := range variants {
		branch := expr.HTTPMediaBranch(body.attribute, expr.AttributeName(variant.Branch))
		if branch == nil {
			panic(fmt.Sprintf("HTTP media branch %q is absent from emitted body %s", variant.Branch, body.Name))
		}
		contracts[i] = strconv.Quote(variant.ContentType)
		data.Variants = append(data.Variants, &MediaVariantData{
			Tag:        variant.Tag,
			Parts:      mediaPartTypesLiteral(variant.PartTypes),
			Field:      codegen.Goify(branch.Name, true),
			TypeRef:    body.context.Scope.Scope().GoTypeRef(branch.Attribute),
			AllowsNull: expr.AllowsNull(branch.Attribute),
			Value:      representation.BuildValuePlan(variant.Value, branch.Attribute, body.context, expr.ValuePlanRuntime),
		})
	}
	data.Contracts = "[]string{" + strings.Join(contracts, ", ") + "}"
	body.Media = data
}

func mediaPartTypesLiteral(parts map[string]string) string {
	if len(parts) == 0 {
		return "nil"
	}
	entries := make([]string, 0, len(parts))
	for _, name := range slices.Sorted(maps.Keys(parts)) {
		entries = append(entries, strconv.Quote(name)+":"+strconv.Quote(parts[name]))
	}
	return "map[string]string{" + strings.Join(entries, ",") + "}"
}
