package testdata


var PayloadBodyPathUserValidateEncodeCode = `// EncodeMethodUserBodyPathValidateRequest returns an encoder for requests sent
// to the ServiceBodyPathUserValidate MethodUserBodyPathValidate server.
func EncodeMethodUserBodyPathValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicebodypathuservalidate.PayloadType)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceBodyPathUserValidate", "MethodUserBodyPathValidate", "*servicebodypathuservalidate.PayloadType", v)
		}
		body := NewMethodUserBodyPathValidateRequestBody(p)
		if err := encoder(req).Encode(&body); err != nil {
			return loomhttp.ErrEncodingError("ServiceBodyPathUserValidate", "MethodUserBodyPathValidate", err)
		}
		return nil
	}
}
`


