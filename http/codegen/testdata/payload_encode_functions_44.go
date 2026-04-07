package testdata


var PayloadJWTAuthorizationHeaderEncodeCode = `// EncodeMethodHeaderPrimitiveStringDefaultRequest returns an encoder for
// requests sent to the ServiceHeaderPrimitiveStringDefault
// MethodHeaderPrimitiveStringDefault server.
func EncodeMethodHeaderPrimitiveStringDefaultRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*serviceheaderprimitivestringdefault.MethodHeaderPrimitiveStringDefaultPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceHeaderPrimitiveStringDefault", "MethodHeaderPrimitiveStringDefault", "*serviceheaderprimitivestringdefault.MethodHeaderPrimitiveStringDefaultPayload", v)
		}
		if p.Token != nil {
			head := *p.Token
			if !strings.Contains(head, " ") {
				req.Header.Set("Authorization", "Bearer "+head)
			} else {
				req.Header.Set("Authorization", head)
			}
		}
		return nil
	}
}
`


