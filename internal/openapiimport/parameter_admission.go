package openapiimport

import (
	"fmt"

	"github.com/CaliLuke/loom/internal/httpencoding"
)

func parameterLocation(location string) (httpencoding.Location, bool) {
	switch location {
	case "query":
		return httpencoding.NamedQuery, true
	case "path":
		return httpencoding.Path, true
	case "header":
		return httpencoding.Header, true
	case "cookie":
		return httpencoding.Cookie, true
	default:
		return 0, false
	}
}

func parameterSerialization(parameter Parameter) httpencoding.Serialization {
	source := parameter.Serialization
	if source.Style == "" {
		source.Style = parameter.Style
	}
	if parameter.AllowReserved {
		source.AllowReserved = &parameter.AllowReserved
	}
	location, ok := parameterLocation(parameter.In)
	if ok {
		source = httpencoding.NormalizeOpenAPIParameter(location, source)
	}
	return source
}

func (p *documentPlanner) parameterCodec(parameter Parameter) (httpencoding.Codec, httpencoding.KeySet, error) {
	location, ok := parameterLocation(parameter.In)
	if !ok {
		return httpencoding.Codec{}, httpencoding.KeySet{}, fmt.Errorf("unsupported parameter location %q", parameter.In)
	}
	return httpencoding.Resolve(location, parameter.Name, p.urlShape(parameter.Schema), parameterSerialization(parameter))
}

func (p *documentPlanner) admitParameter(parameter Parameter, path string) {
	_, _, err := p.parameterCodec(parameter)
	if err == nil {
		return
	}
	code := "parameter-serialization"
	if p.urlShape(parameter.Schema).Shape == httpencoding.UnsupportedShape {
		code = "parameter-shape"
		path += "/schema"
	}
	p.unsupported(code, path, err.Error())
}

func (p *documentPlanner) parameterNamespaces(operation *Operation, path string) {
	bindings := map[string]map[string]httpencoding.KeySet{"query": {}, "cookie": {}}
	for index, source := range operation.Parameters {
		parameter, _, err := resolveParameterReference(source, p.document.Components)
		if err != nil || bindings[parameter.In] == nil {
			continue
		}
		_, keys, err := p.parameterCodec(parameter)
		if err == nil {
			bindings[parameter.In][fmt.Sprintf("parameter %d (%s)", index, parameter.Name)] = keys
		}
	}
	requirements := p.document.Security
	if operation.SecurityDefined {
		requirements = operation.Security
	} else if !p.document.SecurityDefined {
		requirements = nil
	}
	seen := make(map[credentialSlot]bool)
	for _, requirement := range requirements {
		for _, reference := range requirement.Schemes {
			scheme := securitySchemeByName(p.document.Components.SecuritySchemes, reference.Name)
			if scheme == nil {
				continue
			}
			for _, source := range securityCredentialSources(*scheme) {
				if seen[source.slot] || bindings[source.slot.location] == nil {
					continue
				}
				seen[source.slot] = true
				bindings[source.slot.location]["credential "+source.slot.name] = httpencoding.KeySet{Literals: []string{source.slot.name}}
			}
		}
	}
	for _, name := range []string{"query", "cookie"} {
		location, _ := parameterLocation(name)
		if err := httpencoding.CheckNamespaces(location, bindings[name]); err != nil {
			p.unsupported("parameter-namespace", path+"/parameters", err.Error())
		}
	}
}
