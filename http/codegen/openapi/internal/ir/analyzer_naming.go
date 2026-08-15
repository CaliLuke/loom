package ir

import (
	"fmt"
	"sort"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func schemaTypeNaming(attr *expr.AttributeExpr, t expr.UserType) (string, bool) {
	canonical := hasCanonicalOpenAPITypeName(attr.Meta) || hasCanonicalOpenAPITypeName(t.Attribute().Meta)
	var metaName string
	if n, ok := attr.Meta.Last("openapi:typename"); ok {
		metaName = n
	} else if n, ok := t.Attribute().Meta.Last("openapi:typename"); ok {
		metaName = n
	}
	if !canonical {
		metaName = codegen.Goify(metaName, true)
	}
	return metaName, canonical
}

func schemaTypeName(t expr.UserType, metaName string) string {
	if metaName != "" {
		return metaName
	}
	if n, ok := t.Attribute().Meta["name:original"]; ok {
		return n[0]
	}
	return t.Name()
}

func sortedUnionValues(union *expr.Union) []*expr.NamedAttributeExpr {
	if len(union.Values) < 2 {
		return union.Values
	}
	values := append([]*expr.NamedAttributeExpr(nil), union.Values...)
	sort.SliceStable(values, func(i, j int) bool {
		leftName := values[i].Name
		rightName := values[j].Name
		if leftName == rightName {
			return values[i].Attribute.Type.Hash() < values[j].Attribute.Type.Hash()
		}
		return leftName < rightName
	})
	return values
}

func deterministicUnionBranchSchemaName(union *expr.Union, val *expr.NamedAttributeExpr) string {
	if explicit, ok := val.Attribute.Meta.Last("openapi:component:unionEnvelope"); ok {
		if explicit = strings.TrimSpace(explicit); explicit != "" {
			return normalizedOpenAPINamePart(explicit)
		}
	}
	unionName := strings.TrimSpace(union.TypeName)
	if unionName == "" {
		unionName = "Union"
	}
	branchName := strings.TrimSpace(val.Name)
	if branchName == "" {
		branchName = "Value"
	}
	return normalizedOpenAPINamePart(unionName) + normalizedOpenAPINamePart(branchName) + "Envelope"
}

func normalizedOpenAPINamePart(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return !(r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z')
	})
	if len(parts) == 0 {
		return codegen.Goify(raw, true)
	}
	var out strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		out.WriteString(codegen.Goify(part, true))
	}
	if out.Len() == 0 {
		return codegen.Goify(raw, true)
	}
	return out.String()
}

func syntheticUnionBranchSchemaDescription(val *expr.NamedAttributeExpr) string {
	if explicit, ok := val.Attribute.Meta.Last("openapi:description:unionEnvelope"); ok {
		if explicit = strings.TrimSpace(explicit); explicit != "" {
			return explicit
		}
	}
	tag := strings.TrimSpace(expr.UnionVariantTag(val))
	if tag == "" {
		return "Synthetic wrapper for union variant."
	}
	return fmt.Sprintf(`Synthetic wrapper for union variant %q.`, tag)
}

func findMatchingSchemaRef(refs []schemaRef, explicitName string, canonical bool) string {
	for _, ref := range refs {
		if explicitName != "" {
			if canonical {
				if ref.ref == toRef(explicitName) {
					return ref.ref
				}
				continue
			}
			if ref.explicitName == explicitName {
				return ref.ref
			}
			continue
		}
		if ref.explicitName == "" {
			return ref.ref
		}
	}
	return ""
}

func hasCanonicalOpenAPITypeName(meta expr.MetaExpr) bool {
	value, ok := meta.Last("openapi:typename:canonical")
	return ok && value == "true"
}
