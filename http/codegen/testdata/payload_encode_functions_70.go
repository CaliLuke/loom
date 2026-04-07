package testdata


var PayloadBodyPathObjectValidateEncodeCode = `// EncodeMethodBodyPathObjectValidateRequest returns an encoder for requests
// sent to the ServiceBodyPathObjectValidate MethodBodyPathObjectValidate
// server.
func EncodeMethodBodyPathObjectValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicebodypathobjectvalidate.MethodBodyPathObjectValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceBodyPathObjectValidate", "MethodBodyPathObjectValidate", "*servicebodypathobjectvalidate.MethodBodyPathObjectValidatePayload", v)
		}
		body := NewMethodBodyPathObjectValidateRequestBody(p)
		if err := encoder(req).Encode(&body); err != nil {
			return loomhttp.ErrEncodingError("ServiceBodyPathObjectValidate", "MethodBodyPathObjectValidate", err)
		}
		return nil
	}
}
`


