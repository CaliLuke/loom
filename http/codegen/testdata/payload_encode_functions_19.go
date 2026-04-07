package testdata


var PayloadQueryPrimitiveArrayStringValidateEncodeCode = `// EncodeMethodQueryPrimitiveArrayStringValidateRequest returns an encoder for
// requests sent to the ServiceQueryPrimitiveArrayStringValidate
// MethodQueryPrimitiveArrayStringValidate server.
func EncodeMethodQueryPrimitiveArrayStringValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.([]string)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryPrimitiveArrayStringValidate", "MethodQueryPrimitiveArrayStringValidate", "[]string", v)
		}
		values := req.URL.Query()
		for _, value := range p {
			values.Add("q", value)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`


