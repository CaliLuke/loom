package testdata


var PayloadQueryArrayUIntDecodeCode = `// DecodeMethodQueryArrayUIntRequest returns a decoder for requests sent to the
// ServiceQueryArrayUInt MethodQueryArrayUInt endpoint.
func DecodeMethodQueryArrayUIntRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   []uint
			err error
		)
		{
			qRaw := r.URL.Query()["q"]
			if qRaw != nil {
				q = make([]uint, len(qRaw))
				for i, rv := range qRaw {
					v, err2 := strconv.ParseUint(rv, 10, strconv.IntSize)
					if err2 != nil {
						err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "array of unsigned integers"))
					}
					q[i] = uint(v)
				}
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryArrayUIntPayload(q)

		return payload, nil
	}
}
`


