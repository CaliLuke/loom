package representation

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/httpencoding"
	loom "github.com/CaliLuke/loom/pkg"
)

// MultipartJSONShape captures one multipart object's logical schema. Physical
// outer names come from the captured part bindings; JSON contents keep JSON tags.
func MultipartJSONShape(source *expr.AttributeExpr, contract *expr.HTTPMultipartEncoding, context *codegen.AttributeContext) (*loom.JSONShape, error) {
	if source == nil || contract == nil || context == nil {
		return nil, fmt.Errorf("multipart schema requires an occurrence, contract and context")
	}
	target := TargetGraph(source)
	normalizeMultipartRoot(target, contract)
	return formJSONShape(source, target, context)
}

// MultipartFieldJSONShape captures a field's schema independently of its typed
// decoder, preserving the captured per-item branch grammar.
func MultipartFieldJSONShape(source *expr.AttributeExpr, codec httpencoding.MultipartCodec, context *codegen.AttributeContext) (*loom.JSONShape, error) {
	target := TargetGraph(source)
	normalizeMultipartFieldTarget(target, codec)
	return formJSONShape(source, target, context)
}

func normalizeMultipartFieldTarget(target *expr.AttributeExpr, codec httpencoding.MultipartCodec) {
	if codec.StyleCodec != nil {
		normalizeFormFieldTarget(target, *codec.StyleCodec)
		return
	}
	if codec.Kind == httpencoding.BranchPart {
		for index, branch := range expr.AsUnion(target.Type).Values {
			branch.Attribute = TargetGraph(branch.Attribute)
			normalizeMultipartFieldTarget(branch.Attribute, codec.Branches[index])
		}
	} else if codec.Kind == httpencoding.TextPart {
		normalizeTextRecord(target)
	}
}

func normalizeMultipartRoot(target *expr.AttributeExpr, contract *expr.HTTPMultipartEncoding) {
	if contract.Custom {
		return
	}
	if union := expr.AsUnion(target.Type); union != nil && len(contract.Branches) > 0 {
		for index, branch := range union.Values {
			branch.Attribute = TargetGraph(branch.Attribute)
			normalizeMultipartRoot(branch.Attribute, contract.Branches[index])
		}
		return
	}
	for _, field := range *expr.AsObject(target.Type) {
		captured := contract.Fields[field.Name]
		field.Attribute = TargetGraph(field.Attribute)
		if !captured.Omitted {
			normalizeMultipartFieldTarget(field.Attribute, captured.Codec)
		}
		if field.Attribute.Meta == nil {
			field.Attribute.Meta = expr.MetaExpr{}
		}
		name := captured.Name
		if captured.Omitted {
			name = "-"
		}
		delete(field.Attribute.Meta, "struct:tag:json")
		field.Attribute.Meta["struct:tag:json:name"] = []string{name}
	}
}
