package testdata


var PayloadQueryArrayBytesValidateDecodeCode = `// DecodeMethodQueryArrayBytesValidateRequest returns a decoder for requests
// sent to the ServiceQueryArrayBytesValidate MethodQueryArrayBytesValidate
// endpoint.
func DecodeMethodQueryArrayBytesValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   [][]byte
			err error
		)
		{
			qRaw := r.URL.Query()["q"]
			if qRaw == nil {
				err = loom.MergeErrors(err, loom.MissingFieldError("q", "query string"))
			}
			q = make([][]byte, len(qRaw))
			for i, rv := range qRaw {
				q[i] = []byte(rv)
			}
		}
		if len(q) < 1 {
			err = loom.MergeErrors(err, loom.InvalidLengthError("q", q, len(q), 1, true))
		}
		for _, e := range q {
			if len(e) < 2 {
				err = loom.MergeErrors(err, loom.InvalidLengthError("q[*]", e, len(e), 2, true))
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryArrayBytesValidatePayload(q)

		return payload, nil
	}
}
`


