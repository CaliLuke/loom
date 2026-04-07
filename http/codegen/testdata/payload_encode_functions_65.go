package testdata


var PayloadBodyQueryObjectEncodeCode = `// EncodeMethodBodyQueryObjectRequest returns an encoder for requests sent to
// the ServiceBodyQueryObject MethodBodyQueryObject server.
func EncodeMethodBodyQueryObjectRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicebodyqueryobject.MethodBodyQueryObjectPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceBodyQueryObject", "MethodBodyQueryObject", "*servicebodyqueryobject.MethodBodyQueryObjectPayload", v)
		}
		values := req.URL.Query()
		if p.B != nil {
			values.Add("b", *p.B)
		}
		req.URL.RawQuery = values.Encode()
		body := NewMethodBodyQueryObjectRequestBody(p)
		if err := encoder(req).Encode(&body); err != nil {
			return loomhttp.ErrEncodingError("ServiceBodyQueryObject", "MethodBodyQueryObject", err)
		}
		return nil
	}
}
`


