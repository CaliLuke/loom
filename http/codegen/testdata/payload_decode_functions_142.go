package testdata


var PayloadPathCustomUIntDecodeCode = `// DecodeMethodPathCustomUIntRequest returns a decoder for requests sent to the
// ServicePathCustomUInt MethodPathCustomUInt endpoint.
func DecodeMethodPathCustomUIntRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			p   hide.Uint
			err error

			params = mux.Vars(r)
		)
		{
			pRaw := params["p"]
			v, err2 := strconv.ParseUint(pRaw, 10, strconv.IntSize)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("p", pRaw, "unsigned integer"))
			}
			p = hide.Uint(v)
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodPathCustomUIntPayload(p)

		return payload, nil
	}
}
`


