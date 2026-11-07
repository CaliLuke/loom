package expr

import (
	"fmt"

	"github.com/CaliLuke/loom/internal/encodingmeta"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

// HTTPFormEncoding retains the form representation of one body occurrence.
// Branches follow the authored union order; Fields and Element describe object
// and map leaves respectively. Capturing a contract never mutates a declaration.
type HTTPFormEncoding struct {
	// Body is the body occurrence with inline generated wrappers removed and
	// local constraints retained.
	Body *AttributeExpr
	// ObjectOpenness is the effective occurrence policy for unknown physical keys.
	ObjectOpenness ObjectOpenness
	// Metadata is detached occurrence metadata with inherited Encoding Object slots.
	Metadata MetaExpr
	// Fields maps full authored binding identities to wire names, omission and grammar.
	Fields map[string]httpencoding.FieldCodec
	// Element is the value codec for a map root's dynamically named fields.
	Element *httpencoding.Codec
	// Branches retains untagged alternatives in authored order, including nested unions.
	Branches []*HTTPFormEncoding
}

// HTTPFormBodyEncoding resolves a complete form body before code generation.
// Untagged root alternatives must be non-nullable objects or maps. Tagged
// roots retain their discriminator/value protocol and have no captured branches.
// Root Encoding Object slots override branch defaults without changing declarations.
func HTTPFormBodyEncoding(body *AttributeExpr) (*HTTPFormEncoding, error) {
	if err := validateHTTPMediaSourceTypes(body, make(map[*AttributeExpr]bool)); err != nil {
		return nil, err
	}
	if containsTypedObject(body, make(map[*AttributeExpr]bool)) {
		return nil, fmt.Errorf("HTTP form and multipart bodies do not support objects with AdditionalAttributes")
	}
	return captureHTTPFormBody(body, nil, make(map[DataType]bool), true)
}

func captureHTTPFormBody(
	body *AttributeExpr,
	inherited MetaExpr,
	active map[DataType]bool,
	root bool,
) (*HTTPFormEncoding, error) {
	if body == nil || body.Type == nil {
		return nil, fmt.Errorf("form body requires an object or map")
	}
	body = UnwrapInlineHTTPBody(body)
	typ := httpParameterBaseType(body.Type)
	if typ == nil || active[typ] {
		return nil, fmt.Errorf("form body has a recursive root alternative")
	}
	if IsNullable(body) {
		return nil, fmt.Errorf("form body alternatives must be non-nullable")
	}
	active[typ] = true
	defer delete(active, typ)
	metadata, err := captureHTTPMediaMetadata(body, inherited)
	if err != nil {
		return nil, err
	}
	result := &HTTPFormEncoding{Body: body, ObjectOpenness: body.effectiveObjectOpenness(), Metadata: metadata}
	switch actual := typ.(type) {
	case *Union:
		if err := result.captureBranches(actual, active, root); err != nil {
			return nil, err
		}
	case *Object:
		scoped := *body
		scoped.Meta = metadata
		fields, err := captureHTTPFormFields(&scoped)
		if err != nil {
			return nil, err
		}
		result.Fields = fields
	case *Map:
		if _, scalar := httpScalarKind(httpParameterBaseType(actual.KeyType.Type)); !scalar || IsNullable(actual.KeyType) {
			return nil, fmt.Errorf("form map keys require non-nullable scalar values")
		}
		codec, _, err := httpencoding.Resolve(httpencoding.FormField, "", HTTPValueShape(actual.ElemType), httpencoding.Serialization{})
		if err != nil {
			return nil, fmt.Errorf("form map value: %w", err)
		}
		result.Element = &codec
	default:
		return nil, fmt.Errorf("form body requires an object or map")
	}
	return result, nil
}

func captureHTTPFormFields(body *AttributeExpr) (map[string]httpencoding.FieldCodec, error) {
	object := AsObject(body.Type)
	result := make(map[string]httpencoding.FieldCodec, len(*object))
	claims := make([]httpencoding.Binding, 0, len(*object))
	for _, field := range *object {
		element := ElementName(field.Name)
		name, visible := encodingmeta.FormFieldName(field.Attribute.Meta, element)
		captured := httpencoding.FieldCodec{Name: name, Omitted: !visible}
		if visible {
			codec, keys, err := httpencoding.Resolve(httpencoding.FormField, name, HTTPValueShape(field.Attribute), HTTPParameterSerialization(body, "openapi:encoding:"+element+":"))
			if err != nil {
				return nil, fmt.Errorf("form field %q: %w", field.Name, err)
			}
			captured.Codec, captured.Keys = codec, keys
			claims = append(claims, httpencoding.Binding{Name: field.Name, Keys: keys})
		}
		result[field.Name] = captured
	}
	if err := httpencoding.CheckNamespaces(httpencoding.FormField, claims); err != nil {
		return nil, err
	}
	return result, nil
}

func (form *HTTPFormEncoding) captureBranches(union *Union, active map[DataType]bool, root bool) error {
	if !union.Untagged {
		if !root {
			return fmt.Errorf("form root alternatives require objects or maps, not tagged unions")
		}
		if containsUntaggedUnion(form.Body) {
			return fmt.Errorf("tagged form bodies do not support nested untagged OneOf values")
		}
		return nil
	}
	if len(union.Values) == 0 {
		return fmt.Errorf("form union body requires at least one alternative")
	}
	for _, branch := range union.Values {
		child, err := captureHTTPFormBody(branch.Attribute, form.Metadata, active, false)
		if err != nil {
			return fmt.Errorf("form branch %q: %w", branch.Name, err)
		}
		form.Branches = append(form.Branches, child)
	}
	return nil
}
