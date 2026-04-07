package testdata


var PayloadQueryArrayFloat32DecodeCode = `// DecodeMethodQueryArrayFloat32Request returns a decoder for requests sent to
// the ServiceQueryArrayFloat32 MethodQueryArrayFloat32 endpoint.
func DecodeMethodQueryArrayFloat32Request(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   []float32
			err error
		)
		{
			qRaw := r.URL.Query()["q"]
			if qRaw != nil {
				q = make([]float32, len(qRaw))
				for i, rv := range qRaw {
					v, err2 := strconv.ParseFloat(rv, 32)
					if err2 != nil {
						err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "array of floats"))
					}
					q[i] = float32(v)
				}
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryArrayFloat32Payload(q)

		return payload, nil
	}
}
`


