package testdata


var PayloadBodyPathObjectEncodeCode = `// EncodeMethodBodyPathObjectRequest returns an encoder for requests sent to
// the ServiceBodyPathObject MethodBodyPathObject server.
func EncodeMethodBodyPathObjectRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicebodypathobject.MethodBodyPathObjectPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceBodyPathObject", "MethodBodyPathObject", "*servicebodypathobject.MethodBodyPathObjectPayload", v)
		}
		body := NewMethodBodyPathObjectRequestBody(p)
		if err := encoder(req).Encode(&body); err != nil {
			return loomhttp.ErrEncodingError("ServiceBodyPathObject", "MethodBodyPathObject", err)
		}
		return nil
	}
}
`


