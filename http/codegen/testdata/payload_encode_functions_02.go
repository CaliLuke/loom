package testdata


var PayloadQueryMapStringStringValidateEncodeCode = `// EncodeMethodQueryMapStringStringValidateRequest returns an encoder for
// requests sent to the ServiceQueryMapStringStringValidate
// MethodQueryMapStringStringValidate server.
func EncodeMethodQueryMapStringStringValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequerymapstringstringvalidate.MethodQueryMapStringStringValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryMapStringStringValidate", "MethodQueryMapStringStringValidate", "*servicequerymapstringstringvalidate.MethodQueryMapStringStringValidatePayload", v)
		}
		values := req.URL.Query()
		for k, value := range p.Q {
			key := fmt.Sprintf("q[%s]", k)
			values.Add(key, value)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`


