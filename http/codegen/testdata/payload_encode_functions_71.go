package testdata


var PayloadBodyPathUserEncodeCode = `// EncodeMethodBodyPathUserRequest returns an encoder for requests sent to the
// ServiceBodyPathUser MethodBodyPathUser server.
func EncodeMethodBodyPathUserRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicebodypathuser.PayloadType)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceBodyPathUser", "MethodBodyPathUser", "*servicebodypathuser.PayloadType", v)
		}
		body := NewMethodBodyPathUserRequestBody(p)
		if err := encoder(req).Encode(&body); err != nil {
			return loomhttp.ErrEncodingError("ServiceBodyPathUser", "MethodBodyPathUser", err)
		}
		return nil
	}
}
`


