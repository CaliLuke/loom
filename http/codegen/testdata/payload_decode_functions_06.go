package testdata


var PayloadQueryArrayUInt32DecodeCode = `// DecodeMethodQueryArrayUInt32Request returns a decoder for requests sent to
// the ServiceQueryArrayUInt32 MethodQueryArrayUInt32 endpoint.
func DecodeMethodQueryArrayUInt32Request(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   []uint32
			err error
		)
		{
			qRaw := r.URL.Query()["q"]
			if qRaw != nil {
				q = make([]uint32, len(qRaw))
				for i, rv := range qRaw {
					v, err2 := strconv.ParseUint(rv, 10, 32)
					if err2 != nil {
						err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "array of unsigned integers"))
					}
					q[i] = uint32(v)
				}
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryArrayUInt32Payload(q)

		return payload, nil
	}
}
`


