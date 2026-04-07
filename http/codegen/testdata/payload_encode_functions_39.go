package testdata


var PayloadHeaderPrimitiveBoolValidateEncodeCode = `// EncodeMethodHeaderPrimitiveBoolValidateRequest returns an encoder for
// requests sent to the ServiceHeaderPrimitiveBoolValidate
// MethodHeaderPrimitiveBoolValidate server.
func EncodeMethodHeaderPrimitiveBoolValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(bool)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceHeaderPrimitiveBoolValidate", "MethodHeaderPrimitiveBoolValidate", "bool", v)
		}
		return nil
	}
}
`


