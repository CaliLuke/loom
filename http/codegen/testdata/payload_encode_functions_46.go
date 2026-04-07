package testdata


var PayloadBodyStringEncodeCode = `// EncodeMethodBodyStringRequest returns an encoder for requests sent to the
// ServiceBodyString MethodBodyString server.
func EncodeMethodBodyStringRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicebodystring.MethodBodyStringPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceBodyString", "MethodBodyString", "*servicebodystring.MethodBodyStringPayload", v)
		}
		body := NewMethodBodyStringRequestBody(p)
		if err := encoder(req).Encode(&body); err != nil {
			return loomhttp.ErrEncodingError("ServiceBodyString", "MethodBodyString", err)
		}
		return nil
	}
}
`


