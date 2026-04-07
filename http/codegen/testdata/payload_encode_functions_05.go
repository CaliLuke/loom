package testdata


var PayloadQueryMapBoolStringEncodeCode = `// EncodeMethodQueryMapBoolStringRequest returns an encoder for requests sent
// to the ServiceQueryMapBoolString MethodQueryMapBoolString server.
func EncodeMethodQueryMapBoolStringRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequerymapboolstring.MethodQueryMapBoolStringPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryMapBoolString", "MethodQueryMapBoolString", "*servicequerymapboolstring.MethodQueryMapBoolStringPayload", v)
		}
		values := req.URL.Query()
		for kRaw, value := range p.Q {
			k := strconv.FormatBool(kRaw)
			key := fmt.Sprintf("q[%s]", k)
			values.Add(key, value)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`


