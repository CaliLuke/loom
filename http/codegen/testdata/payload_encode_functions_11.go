package testdata


var PayloadQueryMapStringArrayBoolEncodeCode = `// EncodeMethodQueryMapStringArrayBoolRequest returns an encoder for requests
// sent to the ServiceQueryMapStringArrayBool MethodQueryMapStringArrayBool
// server.
func EncodeMethodQueryMapStringArrayBoolRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequerymapstringarraybool.MethodQueryMapStringArrayBoolPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryMapStringArrayBool", "MethodQueryMapStringArrayBool", "*servicequerymapstringarraybool.MethodQueryMapStringArrayBoolPayload", v)
		}
		values := req.URL.Query()
		for k, value := range p.Q {
			key := fmt.Sprintf("q[%s]", k)
			for _, val := range value {
				valStr := strconv.FormatBool(val)
				values.Add(key, valStr)
			}
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`


