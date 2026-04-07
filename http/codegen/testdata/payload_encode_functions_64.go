package testdata


var PayloadBodyPrimitiveFieldArrayUserValidateEncodeCode = `// EncodeMethodBodyPrimitiveArrayUserValidateRequest returns an encoder for
// requests sent to the ServiceBodyPrimitiveArrayUserValidate
// MethodBodyPrimitiveArrayUserValidate server.
func EncodeMethodBodyPrimitiveArrayUserValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicebodyprimitivearrayuservalidate.PayloadType)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceBodyPrimitiveArrayUserValidate", "MethodBodyPrimitiveArrayUserValidate", "*servicebodyprimitivearrayuservalidate.PayloadType", v)
		}
		body := p.A
		if err := encoder(req).Encode(&body); err != nil {
			return loomhttp.ErrEncodingError("ServiceBodyPrimitiveArrayUserValidate", "MethodBodyPrimitiveArrayUserValidate", err)
		}
		return nil
	}
}
`


