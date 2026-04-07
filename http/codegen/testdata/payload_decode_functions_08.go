package testdata


var PayloadQueryArrayUInt64DecodeCode = `// DecodeMethodQueryArrayUInt64Request returns a decoder for requests sent to
// the ServiceQueryArrayUInt64 MethodQueryArrayUInt64 endpoint.
func DecodeMethodQueryArrayUInt64Request(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   []uint64
			err error
		)
		{
			qRaw := r.URL.Query()["q"]
			if qRaw != nil {
				q = make([]uint64, len(qRaw))
				for i, rv := range qRaw {
					v, err2 := strconv.ParseUint(rv, 10, 64)
					if err2 != nil {
						err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "array of unsigned integers"))
					}
					q[i] = v
				}
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryArrayUInt64Payload(q)

		return payload, nil
	}
}
`


