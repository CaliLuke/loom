package testdata


var PayloadJWTAuthorizationCustomHeaderEncodeCode = `// EncodeMethodHeaderPrimitiveStringDefaultRequest returns an encoder for
// requests sent to the ServiceHeaderPrimitiveStringDefault
// MethodHeaderPrimitiveStringDefault server.
func EncodeMethodHeaderPrimitiveStringDefaultRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*serviceheaderprimitivestringdefault.MethodHeaderPrimitiveStringDefaultPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceHeaderPrimitiveStringDefault", "MethodHeaderPrimitiveStringDefault", "*serviceheaderprimitivestringdefault.MethodHeaderPrimitiveStringDefaultPayload", v)
		}
		{
			head := p.Token
			req.Header.Set("X-Auth", head)
		}
		return nil
	}
}
`


