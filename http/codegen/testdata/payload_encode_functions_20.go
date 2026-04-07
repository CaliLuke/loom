package testdata


var PayloadQueryPrimitiveArrayBoolValidateEncodeCode = `// EncodeMethodQueryPrimitiveArrayBoolValidateRequest returns an encoder for
// requests sent to the ServiceQueryPrimitiveArrayBoolValidate
// MethodQueryPrimitiveArrayBoolValidate server.
func EncodeMethodQueryPrimitiveArrayBoolValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.([]bool)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryPrimitiveArrayBoolValidate", "MethodQueryPrimitiveArrayBoolValidate", "[]bool", v)
		}
		values := req.URL.Query()
		for _, value := range p {
			valueStr := strconv.FormatBool(value)
			values.Add("q", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`


