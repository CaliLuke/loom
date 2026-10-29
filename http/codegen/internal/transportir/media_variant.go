package transportir

import (
	"mime"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

// MediaVariant owns one HTTP representation of a semantic body union branch.
// Codec, examples and schema plans belong to this occurrence, not to other media
// that happen to use an equal schema.
type MediaVariant struct {
	// ContentType is the authored media type or range.
	ContentType string
	// Branch is the semantic union branch name.
	Branch string
	// Tag is the generated union discriminator for this branch.
	Tag string
	// Body retains the branch's effective transport attribute.
	Body *expr.AttributeExpr
	// Value retains service provenance and the selected branch's representation plan.
	Value *ValueTarget
	// FormFieldModes records the field encoding policy for this media alone.
	FormFieldModes map[string]httpencoding.Mode
	// PartTypes binds multipart fields to their effective wire Content-Type.
	PartTypes map[string]string
	// Metadata retains occurrence-first media annotations through aliases.
	Metadata expr.MetaExpr
}

func buildMediaVariants(body *expr.AttributeExpr, mappings []*expr.HTTPMediaTypeExpr) []*MediaVariant {
	variants := make([]*MediaVariant, 0, len(mappings))
	for _, mapping := range mappings {
		branch := expr.HTTPMediaBranch(body, mapping.Branch)
		if branch == nil {
			continue // Expression validation owns malformed mappings.
		}
		base, _, err := mime.ParseMediaType(mapping.ContentType)
		if err != nil {
			continue // Expression validation owns malformed media types.
		}
		if base == "application/x-www-form-urlencoded" || base == "multipart/form-data" {
			branch.Attribute = mediaFormCarrier(branch.Attribute)
		}
		var parts map[string]string
		if base == "multipart/form-data" {
			parts, err = expr.HTTPMediaPartTypes(branch.Attribute)
			if err != nil {
				panic(err) // Expression validation establishes this contract.
			}
		}
		variants = append(variants, &MediaVariant{
			ContentType:    mapping.ContentType,
			PartTypes:      parts,
			Metadata:       expr.HTTPMediaMetadata(branch.Attribute),
			Branch:         branch.Name,
			Tag:            expr.UnionVariantTag(branch),
			Body:           branch.Attribute,
			FormFieldModes: formFieldModes(branch.Attribute, base == "application/x-www-form-urlencoded"),
		})
	}
	return variants
}

// mediaFormCarrier gives the form root an independent physical identity. Its
// nested object fields retain JSON layouts because the form codec serializes
// those fields as JSON. Only the root and its aliases use native form presence.
func mediaFormCarrier(attribute *expr.AttributeExpr) *expr.AttributeExpr {
	attribute = expr.DupAtt(attribute)
	current := attribute
	seen := make(map[expr.UserType]bool)
	for {
		named, ok := current.Type.(expr.UserType)
		if !ok || seen[named] {
			return attribute
		}
		seen[named] = true
		named.Rename(named.Name() + "MediaForm")
		current = named.Attribute()
		if current.Meta == nil {
			current.Meta = expr.MetaExpr{}
		}
		current.Meta["http:media:form-root"] = []string{"true"}
	}
}
