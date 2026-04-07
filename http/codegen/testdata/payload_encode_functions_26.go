package testdata


var PayloadQueryStringMappedEncodeCode = `// EncodeMethodQueryStringMappedRequest returns an encoder for requests sent to
// the ServiceQueryStringMapped MethodQueryStringMapped server.
func EncodeMethodQueryStringMappedRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequerystringmapped.MethodQueryStringMappedPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryStringMapped", "MethodQueryStringMapped", "*servicequerystringmapped.MethodQueryStringMappedPayload", v)
		}
		values := req.URL.Query()
		if p.Query != nil {
			values.Add("q", *p.Query)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`


