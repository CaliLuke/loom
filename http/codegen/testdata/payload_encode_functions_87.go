package testdata


var QueryArrayAliasValidateEncodeCode = `// EncodeMethodARequest returns an encoder for requests sent to the
// ServiceQueryArrayAliasValidate MethodA server.
func EncodeMethodARequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarrayaliasvalidate.MethodAPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayAliasValidate", "MethodA", "*servicequeryarrayaliasvalidate.MethodAPayload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Array {
			valueStr := strconv.FormatUint(uint64(value), 10)
			values.Add("array", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`


