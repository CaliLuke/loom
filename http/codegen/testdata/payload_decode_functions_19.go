package testdata


var PayloadQueryArrayAnyValidateDecodeCode = `// DecodeMethodQueryArrayAnyValidateRequest returns a decoder for requests sent
// to the ServiceQueryArrayAnyValidate MethodQueryArrayAnyValidate endpoint.
func DecodeMethodQueryArrayAnyValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   []any
			err error
		)
		{
			qRaw := r.URL.Query()["q"]
			if qRaw == nil {
				err = loom.MergeErrors(err, loom.MissingFieldError("q", "query string"))
			}
			q = make([]any, len(qRaw))
			for i, rv := range qRaw {
				q[i] = rv
			}
		}
		if len(q) < 1 {
			err = loom.MergeErrors(err, loom.InvalidLengthError("q", q, len(q), 1, true))
		}
		for _, e := range q {
			if !(e == "val" || e == 1) {
				err = loom.MergeErrors(err, loom.InvalidEnumValueError("q[*]", e, []any{"val", 1}))
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryArrayAnyValidatePayload(q)

		return payload, nil
	}
}
`


