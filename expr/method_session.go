package expr

import (
	"github.com/CaliLuke/loom/eval"
)

func (m *MethodExpr) injectSessionAuthPayloadFields() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	sessionAuths := m.validationSessionAuths()
	if len(sessionAuths) == 0 {
		return verr
	}
	if m.Payload == nil || m.Payload.Type == Empty {
		m.Payload = &AttributeExpr{Type: &Object{}}
	}
	obj := AsObject(m.Payload.Type)
	if obj == nil {
		verr.Add(m, "payload of method %q of service %q must be an object to inject session auth fields", m.Name, m.Service.Name)
		return verr
	}
	if ut, ok := m.Payload.Type.(UserType); ok {
		dupped := Dup(ut).(UserType)
		if renamer, ok := dupped.(interface{ Rename(string) }); ok {
			renamer.Rename(ut.Name() + "_" + m.Name + "_SessionPayload")
		}
		m.Payload.Type = dupped
		obj = AsObject(m.Payload.Type)
	}
	for _, sessionAuth := range sessionAuths {
		for _, transport := range sessionAuth.Transports {
			verr.Merge(m.injectSessionTransportField(sessionAuth, transport, obj))
		}
	}
	return verr
}

func (m *MethodExpr) injectSessionTransportField(sessionAuth *SessionAuthExpr, transport *SessionTransportExpr, obj *Object) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if transport == nil || transport.Scheme == nil {
		return verr
	}
	if !transport.PayloadOwned() {
		return verr
	}
	if terr := transport.Validate(); terr != nil && len(terr.Errors) > 0 {
		return verr
	}
	if existing := obj.Attribute(transport.FieldName); existing != nil {
		if sessionTransportFieldCompatible(existing, transport) {
			return verr
		}
		verr.Add(
			m,
			"payload of method %q of service %q defines field %q which conflicts with session auth %q %s transport",
			m.Name,
			m.Service.Name,
			transport.FieldName,
			sessionAuth.Name,
			transport.Kind,
		)
		return verr
	}
	att := &AttributeExpr{Type: String}
	switch transport.Kind {
	case SessionBearerTransportKind:
		if transport.Scheme.Kind == OAuth2Kind {
			att.AddMeta("security:accesstoken")
		} else {
			att.AddMeta("security:token")
		}
	case SessionCookieTransportKind:
		att.AddMeta("security:apikey:"+transport.Scheme.SchemeName, transport.Scheme.SchemeName)
	}
	obj.Set(transport.FieldName, att)
	return verr
}

func sessionTransportFieldCompatible(att *AttributeExpr, transport *SessionTransportExpr) bool {
	if att == nil || att.Type != String {
		return false
	}
	switch transport.Kind {
	case SessionBearerTransportKind:
		if transport.Scheme.Kind == OAuth2Kind {
			_, ok := att.Meta["security:accesstoken"]
			return ok
		}
		_, ok := att.Meta["security:token"]
		return ok
	case SessionCookieTransportKind:
		_, ok := att.Meta["security:apikey:"+transport.Scheme.SchemeName]
		return ok
	default:
		return false
	}
}
