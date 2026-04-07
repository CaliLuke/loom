package testdata


var PayloadPathArrayStringDecodeCode = `// DecodeMethodPathArrayStringRequest returns a decoder for requests sent to
// the ServicePathArrayString MethodPathArrayString endpoint.
func DecodeMethodPathArrayStringRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			p []string

			params = mux.Vars(r)
		)
		{
			pRaw := params["p"]
			pRawSlice := strings.Split(pRaw, ",")
			p = make([]string, len(pRawSlice))
			for i, rv := range pRawSlice {
				p[i] = rv
			}
		}
		payload := NewMethodPathArrayStringPayload(p)

		return payload, nil
	}
}
`


