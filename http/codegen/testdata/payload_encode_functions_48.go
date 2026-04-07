package testdata


var PayloadBodyUserEncodeCode = `// EncodeMethodBodyUserRequest returns an encoder for requests sent to the
// ServiceBodyUser MethodBodyUser server.
func EncodeMethodBodyUserRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicebodyuser.PayloadType)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceBodyUser", "MethodBodyUser", "*servicebodyuser.PayloadType", v)
		}
		body := NewMethodBodyUserRequestBody(p)
		if err := encoder(req).Encode(&body); err != nil {
			return loomhttp.ErrEncodingError("ServiceBodyUser", "MethodBodyUser", err)
		}
		return nil
	}
}
`


