package openapiimport

type stringSchemaRepresentation struct {
	typeName   string
	formatName string
	known      bool
	raw        bool
}

// stringRepresentation selects one carrier and validation contract before any
// renderer or occurrence admission consumes a string format.
func stringRepresentation(format string) stringSchemaRepresentation {
	representation := stringSchemaRepresentation{typeName: "String", known: true}
	switch format {
	case "":
	case "binary":
		representation.typeName = "Bytes"
		representation.raw = true
	case "byte":
		representation.formatName = "FormatBase64"
	case "date":
		representation.formatName = "FormatDate"
	case "date-time":
		representation.formatName = "FormatDateTime"
	case "uuid":
		representation.formatName = "FormatUUID"
	case "email":
		representation.formatName = "FormatEmail"
	case "hostname":
		representation.formatName = "FormatHostname"
	case "ipv4":
		representation.formatName = "FormatIPv4"
	case "ipv6":
		representation.formatName = "FormatIPv6"
	case "uri":
		representation.formatName = "FormatURI"
	case "uri-reference":
		representation.formatName = "FormatURIReference"
	default:
		representation.known = false
	}
	return representation
}
