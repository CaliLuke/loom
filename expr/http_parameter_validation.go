package expr

import (
	"fmt"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

func validateParameterBinding(attribute *AttributeExpr, name string, loc httpencoding.Location) (httpencoding.KeySet, error) {
	if err := validateParameterMetadata(HTTPMediaMetadata(attribute), "openapi:"); err != nil {
		return httpencoding.KeySet{}, err
	}
	if IsUnion(attribute.Type) {
		return httpencoding.KeySet{}, fmt.Errorf("unions have no parameter codec")
	}
	_, keys, err := httpencoding.Resolve(loc, name, HTTPValueShape(attribute), HTTPParameterSerialization(attribute, "openapi:"))
	return keys, err
}

func validateParameterMetadata(meta MetaExpr, prefix string) error {
	for _, suffix := range []string{"explode", "allowReserved"} {
		if values, found := meta[prefix+suffix]; found && (len(values) != 1 || values[0] != "true" && values[0] != "false") {
			return fmt.Errorf("%s must be a single boolean value", suffix)
		}
	}
	if values, found := meta[prefix+"style"]; found && (len(values) != 1 || values[0] == "") {
		return fmt.Errorf("style must be one nonempty name")
	}
	return nil
}

func validateMappedParameterCodecs(attributes *MappedAttributeExpr, loc httpencoding.Location, parent eval.Expression, verr *eval.ValidationErrors) {
	if attributes == nil || attributes.IsEmpty() {
		return
	}
	for _, field := range *AsObject(attributes.Type) {
		name := attributes.ElemName(field.Name)
		if _, err := validateParameterBinding(field.Attribute, name, loc); err != nil {
			verr.Add(parent, "%s %q: %s", parameterLocationName(loc), name, err)
		}
	}
}

func (e *HTTPEndpointExpr) validateParameterSerialization(verr *eval.ValidationErrors) {
	_, query := e.transportParamsForValidation()
	_, cookies := e.transportHeadersAndCookiesForValidation()
	e.validateParameterNamespace(query, httpencoding.NamedQuery, verr)
	e.validateParameterNamespace(cookies, httpencoding.Cookie, verr)
	body, _ := e.requestBodySource()
	validateBodyParameterMetadata(body, "body", e, verr, make(map[*AttributeExpr]bool))
	if e.FormRequest && body != nil && IsObject(body.Type) {
		if err := validateMediaFields(body, "application/x-www-form-urlencoded"); err != nil {
			verr.Add(e, "%s", err)
		}
	}
}

func (e *HTTPEndpointExpr) validateParameterNamespace(attributes *MappedAttributeExpr, loc httpencoding.Location, verr *eval.ValidationErrors) {
	bindings := make(map[string]httpencoding.KeySet)
	if attributes != nil {
		for _, field := range *AsObject(attributes.Type) {
			keys, err := validateParameterBinding(field.Attribute, attributes.ElemName(field.Name), loc)
			if err == nil {
				bindings[field.Name] = keys
			}
		}
	}
	for _, credential := range e.CredentialBindings() {
		if credential.Location != parameterLocationName(loc) {
			continue
		}
		mapped := false
		for _, source := range credential.Sources {
			if _, exists := bindings[source]; exists {
				mapped = true
			}
		}
		if !mapped {
			bindings["credential "+credential.Name] = httpencoding.KeySet{Literals: []string{credential.Name}}
		}
	}
	checkLocation := loc
	if loc == httpencoding.NamedQuery && e.MapQueryParams != nil {
		bindings["MapParams"] = httpencoding.KeySet{Residual: true}
		checkLocation = httpencoding.WholeQuery
	}
	if err := httpencoding.CheckNamespaces(checkLocation, bindings); err != nil {
		verr.Add(e, "%s: %s", parameterLocationName(loc), err)
	}
}

func validateResponseHeaderCodecs(headers *MappedAttributeExpr, source *AttributeExpr, parent eval.Expression, verr *eval.ValidationErrors) {
	if headers == nil || source == nil || headers.IsEmpty() {
		return
	}
	prepared := DupMappedAtt(headers)
	initAttr(prepared, source)
	validateMappedParameterCodecs(prepared, httpencoding.Header, parent, verr)
}

func validateBodyParameterMetadata(attribute *AttributeExpr, name string, parent eval.Expression, verr *eval.ValidationErrors, seen map[*AttributeExpr]bool) {
	if attribute == nil || seen[attribute] {
		return
	}
	seen[attribute] = true
	meta := HTTPMediaMetadata(attribute)
	for _, key := range []string{"openapi:style", "openapi:explode", "openapi:allowReserved"} {
		if _, exists := meta[key]; exists {
			verr.Add(parent, "body field %q cannot declare parameter metadata %q", name, key)
		}
	}
	typ := httpParameterBaseType(attribute.Type)
	switch value := typ.(type) {
	case *Object:
		for _, field := range *value {
			validateBodyParameterMetadata(field.Attribute, name+"."+ElementName(field.Name), parent, verr, seen)
		}
	case *Array:
		validateBodyParameterMetadata(value.ElemType, name+"[]", parent, verr, seen)
	case *Map:
		validateBodyParameterMetadata(value.ElemType, name+"[]", parent, verr, seen)
	case *Union:
		for _, branch := range value.Values {
			validateBodyParameterMetadata(branch.Attribute, name+"."+branch.Name, parent, verr, seen)
		}
	}
}

func parameterLocationName(location httpencoding.Location) string {
	switch location {
	case httpencoding.Path:
		return "path parameter"
	case httpencoding.NamedQuery:
		return "query"
	case httpencoding.Header:
		return "header"
	case httpencoding.Cookie:
		return "cookie"
	default:
		return "parameter"
	}
}
