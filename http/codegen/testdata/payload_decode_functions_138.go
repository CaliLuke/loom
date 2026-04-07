package testdata


var PayloadPathCustomFloat64DecodeCode = `// DecodeMethodPathCustomFloat64Request returns a decoder for requests sent to
// the ServicePathCustomFloat64 MethodPathCustomFloat64 endpoint.
func DecodeMethodPathCustomFloat64Request(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			p   hide.Float64
			err error

			params = mux.Vars(r)
		)
		{
			pRaw := params["p"]
			v, err2 := strconv.ParseFloat(pRaw, 64)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("p", pRaw, "float"))
			}
			p = (hide.Float64)(v)
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodPathCustomFloat64Payload(p)

		return payload, nil
	}
}
`


