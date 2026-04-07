package testdata


var PayloadQueryMapBoolArrayBoolEncodeCode = `// EncodeMethodQueryMapBoolArrayBoolRequest returns an encoder for requests
// sent to the ServiceQueryMapBoolArrayBool MethodQueryMapBoolArrayBool server.
func EncodeMethodQueryMapBoolArrayBoolRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequerymapboolarraybool.MethodQueryMapBoolArrayBoolPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryMapBoolArrayBool", "MethodQueryMapBoolArrayBool", "*servicequerymapboolarraybool.MethodQueryMapBoolArrayBoolPayload", v)
		}
		values := req.URL.Query()
		for kRaw, value := range p.Q {
			k := strconv.FormatBool(kRaw)
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


