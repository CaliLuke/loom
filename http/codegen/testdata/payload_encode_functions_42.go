package testdata


var PayloadHeaderStringDefaultEncodeCode = `// EncodeMethodHeaderStringDefaultRequest returns an encoder for requests sent
// to the ServiceHeaderStringDefault MethodHeaderStringDefault server.
func EncodeMethodHeaderStringDefaultRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*serviceheaderstringdefault.MethodHeaderStringDefaultPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceHeaderStringDefault", "MethodHeaderStringDefault", "*serviceheaderstringdefault.MethodHeaderStringDefaultPayload", v)
		}
		{
			head := p.H
			req.Header.Set("h", head)
		}
		return nil
	}
}
`


