package testdata


var PayloadPathCustomUInt64DecodeCode = `// DecodeMethodPathCustomUInt64Request returns a decoder for requests sent to
// the ServicePathCustomUInt64 MethodPathCustomUInt64 endpoint.
func DecodeMethodPathCustomUInt64Request(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			p   hide.Uint64
			err error

			params = mux.Vars(r)
		)
		{
			pRaw := params["p"]
			v, err2 := strconv.ParseUint(pRaw, 10, 64)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("p", pRaw, "unsigned integer"))
			}
			p = (hide.Uint64)(v)
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodPathCustomUInt64Payload(p)

		return payload, nil
	}
}
`


