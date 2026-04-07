package testdata


var PayloadBodyQueryPathUserEncodeCode = `// EncodeMethodBodyQueryPathUserRequest returns an encoder for requests sent to
// the ServiceBodyQueryPathUser MethodBodyQueryPathUser server.
func EncodeMethodBodyQueryPathUserRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicebodyquerypathuser.PayloadType)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceBodyQueryPathUser", "MethodBodyQueryPathUser", "*servicebodyquerypathuser.PayloadType", v)
		}
		values := req.URL.Query()
		if p.B != nil {
			values.Add("b", *p.B)
		}
		req.URL.RawQuery = values.Encode()
		body := NewMethodBodyQueryPathUserRequestBody(p)
		if err := encoder(req).Encode(&body); err != nil {
			return loomhttp.ErrEncodingError("ServiceBodyQueryPathUser", "MethodBodyQueryPathUser", err)
		}
		return nil
	}
}
`


