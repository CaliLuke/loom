package testdata


var PayloadQueryArrayInt64DecodeCode = `// DecodeMethodQueryArrayInt64Request returns a decoder for requests sent to
// the ServiceQueryArrayInt64 MethodQueryArrayInt64 endpoint.
func DecodeMethodQueryArrayInt64Request(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   []int64
			err error
		)
		{
			qRaw := r.URL.Query()["q"]
			if qRaw != nil {
				q = make([]int64, len(qRaw))
				for i, rv := range qRaw {
					v, err2 := strconv.ParseInt(rv, 10, 64)
					if err2 != nil {
						err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "array of integers"))
					}
					q[i] = v
				}
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryArrayInt64Payload(q)

		return payload, nil
	}
}
`


