package testdata


var PayloadBodyCustomNameEncodeCode = `// EncodeMethodBodyCustomNameRequest returns an encoder for requests sent to
// the ServiceBodyCustomName MethodBodyCustomName server.
func EncodeMethodBodyCustomNameRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicebodycustomname.MethodBodyCustomNamePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceBodyCustomName", "MethodBodyCustomName", "*servicebodycustomname.MethodBodyCustomNamePayload", v)
		}
		body := NewMethodBodyCustomNameRequestBody(p)
		if err := encoder(req).Encode(&body); err != nil {
			return loomhttp.ErrEncodingError("ServiceBodyCustomName", "MethodBodyCustomName", err)
		}
		return nil
	}
}
`


