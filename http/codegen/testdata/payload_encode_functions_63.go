package testdata


var PayloadBodyPrimitiveFieldArrayUserEncodeCode = `// EncodeMethodBodyPrimitiveArrayUserRequest returns an encoder for requests
// sent to the ServiceBodyPrimitiveArrayUser MethodBodyPrimitiveArrayUser
// server.
func EncodeMethodBodyPrimitiveArrayUserRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicebodyprimitivearrayuser.PayloadType)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceBodyPrimitiveArrayUser", "MethodBodyPrimitiveArrayUser", "*servicebodyprimitivearrayuser.PayloadType", v)
		}
		body := p.A
		if err := encoder(req).Encode(&body); err != nil {
			return loomhttp.ErrEncodingError("ServiceBodyPrimitiveArrayUser", "MethodBodyPrimitiveArrayUser", err)
		}
		return nil
	}
}
`


