package testdata


var PayloadHeaderPrimitiveArrayBoolValidateEncodeCode = `// EncodeMethodHeaderPrimitiveArrayBoolValidateRequest returns an encoder for
// requests sent to the ServiceHeaderPrimitiveArrayBoolValidate
// MethodHeaderPrimitiveArrayBoolValidate server.
func EncodeMethodHeaderPrimitiveArrayBoolValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.([]bool)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceHeaderPrimitiveArrayBoolValidate", "MethodHeaderPrimitiveArrayBoolValidate", "[]bool", v)
		}
		return nil
	}
}
`


