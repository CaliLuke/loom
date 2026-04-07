package testdata


var PayloadQueryArrayFloat64DecodeCode = `// DecodeMethodQueryArrayFloat64Request returns a decoder for requests sent to
// the ServiceQueryArrayFloat64 MethodQueryArrayFloat64 endpoint.
func DecodeMethodQueryArrayFloat64Request(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   []float64
			err error
		)
		{
			qRaw := r.URL.Query()["q"]
			if qRaw != nil {
				q = make([]float64, len(qRaw))
				for i, rv := range qRaw {
					v, err2 := strconv.ParseFloat(rv, 64)
					if err2 != nil {
						err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "array of floats"))
					}
					q[i] = v
				}
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryArrayFloat64Payload(q)

		return payload, nil
	}
}
`


