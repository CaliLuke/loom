package httpencoding

// NormalizeOpenAPIParameter supplies OpenAPI schema-parameter defaults while
// preserving content contracts and the presence of authored serialization flags.
func NormalizeOpenAPIParameter(location Location, source Serialization) Serialization {
	if source.MediaType == "" && source.Style == "" {
		switch location {
		case Path, Header:
			source.Style = "simple"
		case NamedQuery, Cookie:
			source.Style = "form"
		}
	}
	return source
}
