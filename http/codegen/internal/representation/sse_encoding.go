package representation

import "github.com/CaliLuke/loom/expr"

// SSEEncoding describes the dynamic value passed to EncodeSSEData. Named Go
// types and pointer/presence wrappers use its JSON fallback, not native dispatch.
type SSEEncoding struct {
	// Codec identifies the mapped field's physical wire encoding.
	Codec expr.ValueCodec
	// DereferenceString records the emitter's optional native string handling.
	DereferenceString bool
}

// MappedSSEEncoding classifies an actual emitted Go field type reference and
// pointer policy. It is shared by SSE payload emission and schema preparation.
func MappedSSEEncoding(typeRef string, pointer bool) SSEEncoding {
	switch typeRef {
	case "string":
		return SSEEncoding{Codec: expr.ValueCodecText, DereferenceString: pointer}
	case "[]byte":
		if !pointer {
			return SSEEncoding{Codec: expr.ValueCodecRaw}
		}
	}
	return SSEEncoding{Codec: expr.ValueCodecJSON}
}
