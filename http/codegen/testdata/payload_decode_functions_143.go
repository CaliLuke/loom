package testdata


var PayloadPathCustomUInt32DecodeCode = `// DecodeMethodPathCustomUInt32Request returns a decoder for requests sent to
// the ServicePathCustomUInt32 MethodPathCustomUInt32 endpoint.
func DecodeMethodPathCustomUInt32Request(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			p   hide.Uint32
			err error

			params = mux.Vars(r)
		)
		{
			pRaw := params["p"]
			v, err2 := strconv.ParseUint(pRaw, 10, 32)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("p", pRaw, "unsigned integer"))
			}
			p = hide.Uint32(v)
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodPathCustomUInt32Payload(p)

		return payload, nil
	}
}
`


