package testdata


var PayloadHeaderStringEncodeCode = `// EncodeMethodHeaderStringRequest returns an encoder for requests sent to the
// ServiceHeaderString MethodHeaderString server.
func EncodeMethodHeaderStringRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*serviceheaderstring.MethodHeaderStringPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceHeaderString", "MethodHeaderString", "*serviceheaderstring.MethodHeaderStringPayload", v)
		}
		if p.H != nil {
			head := *p.H
			req.Header.Set("h", head)
		}
		return nil
	}
}
`


