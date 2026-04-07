package testdata


var PayloadQueryPrimitiveMapBoolArrayBoolValidateEncodeCode = `// EncodeMethodQueryPrimitiveMapBoolArrayBoolValidateRequest returns an encoder
// for requests sent to the ServiceQueryPrimitiveMapBoolArrayBoolValidate
// MethodQueryPrimitiveMapBoolArrayBoolValidate server.
func EncodeMethodQueryPrimitiveMapBoolArrayBoolValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(map[bool][]bool)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryPrimitiveMapBoolArrayBoolValidate", "MethodQueryPrimitiveMapBoolArrayBoolValidate", "map[bool][]bool", v)
		}
		values := req.URL.Query()
		for kRaw, value := range p {
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


