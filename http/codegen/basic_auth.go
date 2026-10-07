package codegen

import "github.com/CaliLuke/loom/codegen/service"

// BasicAuthData describes the presence contract of a Basic Authorization header.
// Encoding and decoding share this plan rather than deriving header presence
// independently from the two credential fields.
type BasicAuthData struct {
	// SchemeData retains the credential selectors, types and field presence.
	*service.SchemeData
	// HeaderRequired requires a valid Basic header when either component is required.
	HeaderRequired bool
	// AlwaysSend emits the header when either field has a value representation
	// (including a declared default), or either component is required. Otherwise
	// the encoder emits it only when at least one pointer field is present.
	AlwaysSend bool
}

func buildBasicAuthData(scheme *service.SchemeData) *BasicAuthData {
	if scheme == nil {
		return nil
	}
	required := scheme.UsernameRequired || scheme.PasswordRequired
	return &BasicAuthData{
		SchemeData:     scheme,
		HeaderRequired: required,
		AlwaysSend:     required || !scheme.UsernamePointer || !scheme.PasswordPointer,
	}
}
