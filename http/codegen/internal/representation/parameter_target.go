package representation

import (
	"fmt"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

// normalizeParameterTarget gives text records their DSL member names. JSON tags
// belong only to JSON occurrences, including ordinary JSON-valued form fields.
// The caller owns target; independently copy each styled field because a form
// can reference the same declaration through both a styled and a JSON field.
func normalizeParameterTarget(target *expr.AttributeExpr, codec expr.ValueCodec) {
	switch codec {
	case expr.ValueCodecText:
		normalizeTextRecord(target)
	case expr.ValueCodecMultipart:
		contract, err := expr.HTTPMultipartBodyEncoding(target)
		if err != nil {
			panic(fmt.Errorf("validated multipart body: %w", err))
		}
		normalizeMultipartRoot(expr.UnwrapInlineHTTPBody(target), contract)
	case expr.ValueCodecForm:
		contract, err := expr.HTTPFormBodyEncoding(target)
		if err != nil {
			panic(fmt.Errorf("validated form body: %w", err))
		}
		normalizeFormTarget(expr.UnwrapInlineHTTPBody(target), contract)
	}
}

func normalizeFormTarget(target *expr.AttributeExpr, contract *expr.HTTPFormEncoding) {
	if union := expr.AsUnion(target.Type); union != nil && len(contract.Branches) > 0 {
		for index, branch := range union.Values {
			branch.Attribute = TargetGraph(branch.Attribute)
			normalizeFormTarget(branch.Attribute, contract.Branches[index])
		}
		return
	}
	if mapping := expr.AsMap(target.Type); mapping != nil {
		mapping.ElemType = TargetGraph(mapping.ElemType)
		if contract.Element != nil {
			normalizeFormFieldTarget(mapping.ElemType, *contract.Element)
		}
		return
	}
	object := expr.AsObject(target.Type)
	if object == nil {
		return
	}
	for _, field := range *object {
		captured := contract.Fields[field.Name]
		field.Attribute = TargetGraph(field.Attribute)
		if !captured.Omitted {
			normalizeFormFieldTarget(field.Attribute, captured.Codec)
		}
		if field.Attribute.Meta == nil {
			field.Attribute.Meta = make(expr.MetaExpr)
		}
		name := captured.Name
		if captured.Omitted {
			name = "-"
		}
		// The logical form object uses outer form names; nested JSON fields
		// retain their independently captured JSON contract.
		delete(field.Attribute.Meta, "struct:tag:json")
		field.Attribute.Meta["struct:tag:json:name"] = []string{name}
	}
}

func normalizeTextRecord(attribute *expr.AttributeExpr) {
	if union := expr.AsUnion(attribute.Type); union != nil && union.Untagged {
		for _, branch := range union.Values {
			branch.Attribute = TargetGraph(branch.Attribute)
			normalizeTextRecord(branch.Attribute)
		}
		return
	}
	object := expr.AsObject(attribute.Type)
	if object == nil {
		return
	}
	for _, field := range *object {
		delete(field.Attribute.Meta, "struct:tag:json")
		if field.Attribute.Meta == nil {
			field.Attribute.Meta = make(expr.MetaExpr)
		}
		field.Attribute.Meta["struct:tag:json:name"] = []string{expr.ElementName(field.Name)}
	}
}

func normalizeFormFieldTarget(attribute *expr.AttributeExpr, codec httpencoding.Codec) {
	if union := expr.AsUnion(attribute.Type); union != nil && len(codec.Branches) > 0 {
		for index, branch := range union.Values {
			branch.Attribute = TargetGraph(branch.Attribute)
			normalizeFormFieldTarget(branch.Attribute, codec.Branches[index].Codec)
		}
		return
	}
	if codec.Style != httpencoding.JSON {
		normalizeTextRecord(attribute)
	}
}
