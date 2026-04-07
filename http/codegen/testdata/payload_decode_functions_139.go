package testdata


var PayloadPathCustomIntDecodeCode = `// DecodeMethodPathCustomIntRequest returns a decoder for requests sent to the
// ServicePathCustomInt MethodPathCustomInt endpoint.
func DecodeMethodPathCustomIntRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			p   hide.Int
			err error

			params = mux.Vars(r)
		)
		{
			pRaw := params["p"]
			v, err2 := strconv.ParseInt(pRaw, 10, strconv.IntSize)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("p", pRaw, "integer"))
			}
			p = hide.Int(v)
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodPathCustomIntPayload(p)

		return payload, nil
	}
}
`


