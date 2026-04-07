package testdata


var PayloadQueryMapStringArrayStringValidateEncodeCode = `// EncodeMethodQueryMapStringArrayStringValidateRequest returns an encoder for
// requests sent to the ServiceQueryMapStringArrayStringValidate
// MethodQueryMapStringArrayStringValidate server.
func EncodeMethodQueryMapStringArrayStringValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequerymapstringarraystringvalidate.MethodQueryMapStringArrayStringValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryMapStringArrayStringValidate", "MethodQueryMapStringArrayStringValidate", "*servicequerymapstringarraystringvalidate.MethodQueryMapStringArrayStringValidatePayload", v)
		}
		values := req.URL.Query()
		for k, value := range p.Q {
			key := fmt.Sprintf("q[%s]", k)
			values[key] = value
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`


