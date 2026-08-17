package openapiimport

func supportedParameterStyle(location, style string) bool {
	switch location {
	case "path", "header":
		return style == "simple"
	case "query":
		return style == "form"
	case "cookie":
		return style == "form" || style == "cookie"
	default:
		return false
	}
}
