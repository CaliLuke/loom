package testdata


var PayloadQueryMapBoolArrayBoolValidateEncodeCode = `// EncodeMethodQueryMapBoolArrayBoolValidateRequest returns an encoder for
// requests sent to the ServiceQueryMapBoolArrayBoolValidate
// MethodQueryMapBoolArrayBoolValidate server.
func EncodeMethodQueryMapBoolArrayBoolValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequerymapboolarrayboolvalidate.MethodQueryMapBoolArrayBoolValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryMapBoolArrayBoolValidate", "MethodQueryMapBoolArrayBoolValidate", "*servicequerymapboolarrayboolvalidate.MethodQueryMapBoolArrayBoolValidatePayload", v)
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


