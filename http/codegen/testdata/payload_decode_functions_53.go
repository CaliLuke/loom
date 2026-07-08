package testdata

var PayloadPathArrayStringDecodeCode = `// DecodeMethodPathArrayStringRequest returns a decoder for requests sent to
// the ServicePathArrayString MethodPathArrayString endpoint.
func DecodeMethodPathArrayStringRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			p   []string
			err error

			params = mux.Vars(r)
		)
		{
			pRaw := params["p"]
			pRawSlice := strings.Split(pRaw, ",")
			p = make([]string, len(pRawSlice))
			for i, rv := range pRawSlice {
				rvDecoded, err2 := url.PathUnescape(rv)
				if err2 != nil {
					err = loom.MergeErrors(err, loom.InvalidFieldTypeError("p", pRaw, "path-escaped array"))
				}
				rv = rvDecoded
				p[i] = rv
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodPathArrayStringPayload(p)

		return payload, nil
	}
}
`
