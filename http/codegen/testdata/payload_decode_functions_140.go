package testdata


var PayloadPathCustomInt32DecodeCode = `// DecodeMethodPathCustomInt32Request returns a decoder for requests sent to
// the ServicePathCustomInt32 MethodPathCustomInt32 endpoint.
func DecodeMethodPathCustomInt32Request(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			p   hide.Int32
			err error

			params = mux.Vars(r)
		)
		{
			pRaw := params["p"]
			v, err2 := strconv.ParseInt(pRaw, 10, 32)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("p", pRaw, "integer"))
			}
			p = hide.Int32(v)
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodPathCustomInt32Payload(p)

		return payload, nil
	}
}
`


