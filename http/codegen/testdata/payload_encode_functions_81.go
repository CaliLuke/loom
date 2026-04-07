package testdata


var PayloadMultipartBodyUserTypeEncodeCode = `// EncodeMethodMultipartUserTypeRequest returns an encoder for requests sent to
// the ServiceMultipartUserType MethodMultipartUserType server.
func EncodeMethodMultipartUserTypeRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicemultipartusertype.MethodMultipartUserTypePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceMultipartUserType", "MethodMultipartUserType", "*servicemultipartusertype.MethodMultipartUserTypePayload", v)
		}
		if err := encoder(req).Encode(p); err != nil {
			return loomhttp.ErrEncodingError("ServiceMultipartUserType", "MethodMultipartUserType", err)
		}
		return nil
	}
}
`


