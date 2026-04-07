package testdata


var PayloadPathArrayStringValidateDecodeCode = `// DecodeMethodPathArrayStringValidateRequest returns a decoder for requests
// sent to the ServicePathArrayStringValidate MethodPathArrayStringValidate
// endpoint.
func DecodeMethodPathArrayStringValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
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
				p[i] = rv
			}
		}
		if !(p == []string{"val"}) {
			err = loom.MergeErrors(err, loom.InvalidEnumValueError("p", p, []any{[]string{"val"}}))
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodPathArrayStringValidatePayload(p)

		return payload, nil
	}
}
`


