package testdata


var PayloadPathCustomInt64DecodeCode = `// DecodeMethodPathCustomInt64Request returns a decoder for requests sent to
// the ServicePathCustomInt64 MethodPathCustomInt64 endpoint.
func DecodeMethodPathCustomInt64Request(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			p   hide.Int64
			err error

			params = mux.Vars(r)
		)
		{
			pRaw := params["p"]
			v, err2 := strconv.ParseInt(pRaw, 10, 64)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("p", pRaw, "integer"))
			}
			p = (hide.Int64)(v)
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodPathCustomInt64Payload(p)

		return payload, nil
	}
}
`


