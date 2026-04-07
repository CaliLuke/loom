package testdata


var PayloadHeaderStringValidateEncodeCode = `// EncodeMethodHeaderStringValidateRequest returns an encoder for requests sent
// to the ServiceHeaderStringValidate MethodHeaderStringValidate server.
func EncodeMethodHeaderStringValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*serviceheaderstringvalidate.MethodHeaderStringValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceHeaderStringValidate", "MethodHeaderStringValidate", "*serviceheaderstringvalidate.MethodHeaderStringValidatePayload", v)
		}
		if p.H != nil {
			head := *p.H
			req.Header.Set("h", head)
		}
		return nil
	}
}
`


