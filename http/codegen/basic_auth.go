package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
)

// BasicAuthData describes the presence and validation of a Basic Authorization header.
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
	// UsernameValidate applies the effective username predicates to a supplied physical string.
	UsernameValidate string
	// PasswordValidate applies the effective password predicates to a supplied physical string.
	PasswordValidate string
}

func buildBasicAuthData(scheme *service.SchemeData, payload *expr.AttributeExpr, scope *codegen.NameScope) *BasicAuthData {
	if scheme == nil {
		return nil
	}
	username := payload.Find(scheme.UsernameAttr)
	password := payload.Find(scheme.PasswordAttr)
	ctx := codegen.NewAttributeContext(false, false, false, "", scope)
	required := scheme.UsernameRequired || scheme.PasswordRequired
	return &BasicAuthData{
		SchemeData:       scheme,
		HeaderRequired:   required,
		AlwaysSend:       required || !scheme.UsernamePointer || !scheme.PasswordPointer,
		UsernameValidate: codegen.AttributeValidationCode(username, nil, ctx, true, expr.IsAlias(username.Type), "user", scheme.UsernameAttr),
		PasswordValidate: codegen.AttributeValidationCode(password, nil, ctx, true, expr.IsAlias(password.Type), "pass", scheme.PasswordAttr),
	}
}
