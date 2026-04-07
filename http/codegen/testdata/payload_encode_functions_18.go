package testdata


var PayloadQueryPrimitiveBoolValidateEncodeCode = `// EncodeMethodQueryPrimitiveBoolValidateRequest returns an encoder for
// requests sent to the ServiceQueryPrimitiveBoolValidate
// MethodQueryPrimitiveBoolValidate server.
func EncodeMethodQueryPrimitiveBoolValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(bool)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryPrimitiveBoolValidate", "MethodQueryPrimitiveBoolValidate", "bool", v)
		}
		values := req.URL.Query()
		pStr := strconv.FormatBool(p)
		values.Add("q", pStr)
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`


