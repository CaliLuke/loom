package expr

import (
	"fmt"
	"mime"
	"strings"
)

func validateMediaFields(body *AttributeExpr, media string) error {
	base, _, err := mime.ParseMediaType(media)
	if err != nil {
		return err
	}
	switch base {
	case "application/x-www-form-urlencoded":
		_, err = HTTPFormBodyEncoding(body)
	case "multipart/form-data":
		var contract *HTTPMultipartEncoding
		contract, err = HTTPMultipartBodyEncoding(body)
		if err == nil && contract.Custom {
			return fmt.Errorf("multipart media requires a generated object body contract")
		}
	}
	return err
}

func httpMediaBaseType(typ DataType) DataType {
	for {
		named, ok := typ.(UserType)
		if !ok {
			return typ
		}
		typ = named.Attribute().Type
	}
}

// HTTPMediaMetadata resolves occurrence-first metadata through body aliases.
// It returns a detached map so transport owners can retain the effective contract.
func HTTPMediaMetadata(attribute *AttributeExpr) MetaExpr {
	result := MetaExpr{}
	seen := make(map[*AttributeExpr]bool)
	for attribute != nil && !seen[attribute] {
		seen[attribute] = true
		for name, values := range attribute.Meta {
			if _, found := result[name]; !found {
				result[name] = append([]string(nil), values...)
			}
		}
		named, ok := attribute.Type.(UserType)
		if !ok {
			return result
		}
		attribute = named.Attribute()
	}
	return result
}

func validateMediaEncodingKeys(meta MetaExpr, object *Object) error {
	allowed := map[string]bool{"http:encoding:sourceType": true}
	for _, field := range *object {
		for _, suffix := range []string{"contentType", "style", "explode", "allowReserved"} {
			allowed["openapi:encoding:"+ElementName(field.Name)+":"+suffix] = true
		}
	}
	for key := range meta {
		if (strings.HasPrefix(key, "openapi:encoding:") || strings.HasPrefix(key, "http:encoding:")) && !allowed[key] || strings.HasPrefix(key, "openapi:itemEncoding:") || strings.HasPrefix(key, "openapi:prefixEncoding:") {
			return fmt.Errorf("media encoding %q has no generated field contract", key)
		}
	}
	return nil
}

func captureHTTPMediaMetadata(body *AttributeExpr, inherited MetaExpr) (MetaExpr, error) {
	metadata := HTTPMediaMetadata(body)
	properties := httpMediaRootProperties(body, make(map[DataType]bool))
	if err := validateMediaEncodingKeys(metadata, properties); err != nil {
		return nil, err
	}
	for _, field := range *properties {
		if err := validateParameterMetadata(metadata, "openapi:encoding:"+ElementName(field.Name)+":"); err != nil {
			return nil, fmt.Errorf("media field %q: %w", field.Name, err)
		}
	}
	for key, values := range inherited {
		if strings.HasPrefix(key, "openapi:encoding:") {
			metadata[key] = append([]string(nil), values...)
		}
	}
	return metadata, nil
}

func httpMediaRootProperties(body *AttributeExpr, active map[DataType]bool) *Object {
	result := &Object{}
	if body == nil {
		return result
	}
	typ := httpParameterBaseType(body.Type)
	if typ == nil || active[typ] {
		return result
	}
	active[typ] = true
	defer delete(active, typ)
	switch actual := typ.(type) {
	case *Object:
		*result = append(*result, (*actual)...)
	case *Union:
		if !actual.Untagged {
			return result
		}
		for _, branch := range actual.Values {
			*result = append(*result, (*httpMediaRootProperties(branch.Attribute, active))...)
		}
	}
	return result
}

func validateHTTPMediaSourceTypes(attribute *AttributeExpr, seen map[*AttributeExpr]bool) error {
	if attribute == nil || seen[attribute] {
		return nil
	}
	seen[attribute] = true
	if values, found := attribute.Meta["http:encoding:sourceType"]; found {
		if len(values) != 1 {
			return fmt.Errorf("http:encoding:sourceType must contain one source type")
		}
		switch values[0] {
		case "absent", "array", "object", "string", "integer", "number", "boolean":
		default:
			return fmt.Errorf("http:encoding:sourceType %q is not a supported source type", values[0])
		}
		if err := validateHTTPSourceType(attribute, values[0]); err != nil {
			return err
		}
	}
	var children []*AttributeExpr
	switch actual := attribute.Type.(type) {
	case UserType:
		children = append(children, actual.Attribute())
	case *Object:
		for _, field := range *actual {
			children = append(children, field.Attribute)
		}
	case *Array:
		children = append(children, actual.ElemType)
	case *Map:
		children = append(children, actual.KeyType, actual.ElemType)
	case *Union:
		for _, branch := range actual.Values {
			children = append(children, branch.Attribute)
		}
	}
	for _, child := range children {
		if err := validateHTTPMediaSourceTypes(child, seen); err != nil {
			return err
		}
	}
	return nil
}
