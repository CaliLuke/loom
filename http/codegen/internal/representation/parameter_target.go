package representation

import (
	"fmt"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

// parameterContract is the form or multipart contract that normalizes a
// parameter target, captured by the media contract owners.
type parameterContract struct {
	form      *expr.HTTPFormEncoding
	multipart *expr.HTTPMultipartEncoding
}

// captureParameterContract captures the contract of target under codec. Only
// form and multipart bodies have one.
func captureParameterContract(target *expr.AttributeExpr, codec expr.ValueCodec) (parameterContract, error) {
	var contract parameterContract
	var err error
	switch codec {
	case expr.ValueCodecMultipart:
		contract.multipart, err = expr.HTTPMultipartBodyEncoding(target)
	case expr.ValueCodecForm:
		contract.form, err = expr.HTTPFormBodyEncoding(target)
	}
	return contract, err
}

// normalizeParameterTarget gives text records their DSL member names. JSON tags
// belong only to JSON occurrences, including ordinary JSON-valued form fields.
// The caller owns target; independently copy each styled field because a form
// can reference the same declaration through both a styled and a JSON field.
// A nil contract is captured from target itself. It returns the normalized
// occurrences that a form or multipart contract encodes as JSON content; each
// is private to its field, branch or map value.
func normalizeParameterTarget(target *expr.AttributeExpr, codec expr.ValueCodec, contract *parameterContract) []*expr.AttributeExpr {
	if contract == nil {
		captured, err := captureParameterContract(target, codec)
		if err != nil {
			panic(fmt.Errorf("validated parameter body: %w", err))
		}
		contract = &captured
	}
	var boundaries []*expr.AttributeExpr
	switch codec {
	case expr.ValueCodecText:
		normalizeTextRecord(target)
	case expr.ValueCodecMultipart:
		normalizeMultipartRoot(expr.UnwrapInlineHTTPBody(target), contract.multipart, &boundaries)
	case expr.ValueCodecForm:
		normalizeFormTarget(expr.UnwrapInlineHTTPBody(target), contract.form, &boundaries)
	}
	return boundaries
}

// normalizeFormTarget normalizes a form root alternative and, when boundaries
// is not nil, collects its JSON content occurrences.
func normalizeFormTarget(target *expr.AttributeExpr, contract *expr.HTTPFormEncoding, boundaries *[]*expr.AttributeExpr) {
	if union := expr.AsUnion(target.Type); union != nil && len(contract.TaggedBranches) > 0 {
		for index, branch := range union.Values {
			captured := contract.TaggedBranches[index]
			branch.Attribute = TargetGraph(branch.Attribute)
			if captured.Object != nil {
				normalizeFormTarget(branch.Attribute, captured.Object, boundaries)
			} else {
				normalizeFormFieldTarget(branch.Attribute, captured.Value.Codec, boundaries)
			}
		}
		return
	}
	if union := expr.AsUnion(target.Type); union != nil && len(contract.Branches) > 0 {
		for index, branch := range union.Values {
			branch.Attribute = TargetGraph(branch.Attribute)
			normalizeFormTarget(branch.Attribute, contract.Branches[index], boundaries)
		}
		return
	}
	if mapping := expr.AsMap(target.Type); mapping != nil {
		mapping.ElemType = TargetGraph(mapping.ElemType)
		if contract.Element != nil {
			normalizeFormFieldTarget(mapping.ElemType, *contract.Element, boundaries)
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
			normalizeFormFieldTarget(field.Attribute, captured.Codec, boundaries)
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

// normalizeFormFieldTarget applies one field value's resolved grammar. A JSON
// grammar makes the value, or each item under item application, a JSON content
// boundary; other grammars use element names.
func normalizeFormFieldTarget(attribute *expr.AttributeExpr, codec httpencoding.Codec, boundaries *[]*expr.AttributeExpr) {
	if union := expr.AsUnion(attribute.Type); union != nil && len(codec.Branches) > 0 {
		for index, branch := range union.Values {
			branch.Attribute = TargetGraph(branch.Attribute)
			normalizeFormFieldTarget(branch.Attribute, codec.Branches[index].Codec, boundaries)
		}
		return
	}
	if codec.Style == httpencoding.JSON {
		recordJSONContent(boundaries, attribute, codec.Application)
		return
	}
	normalizeTextRecord(attribute)
}

// recordJSONContent records the JSON content boundary of a normalized value.
// Under item application each array item is encoded separately, so the
// boundary is the array's element and the array keeps the enclosing codec.
func recordJSONContent(boundaries *[]*expr.AttributeExpr, value *expr.AttributeExpr, application httpencoding.Application) {
	if boundaries == nil {
		return
	}
	if application == httpencoding.Items {
		array := expr.AsArray(value.Type)
		if array == nil {
			panic(fmt.Errorf("item-applied JSON content requires an array value"))
		}
		value = array.ElemType
	}
	*boundaries = append(*boundaries, value)
}
