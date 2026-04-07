package testdata


var PayloadQueryArrayStringValidateDecodeCode = `// DecodeMethodQueryArrayStringValidateRequest returns a decoder for requests
// sent to the ServiceQueryArrayStringValidate MethodQueryArrayStringValidate
// endpoint.
func DecodeMethodQueryArrayStringValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   []string
			err error
		)
		q = r.URL.Query()["q"]
		if q == nil {
			err = loom.MergeErrors(err, loom.MissingFieldError("q", "query string"))
		}
		if len(q) < 1 {
			err = loom.MergeErrors(err, loom.InvalidLengthError("q", q, len(q), 1, true))
		}
		for _, e := range q {
			if !(e == "val") {
				err = loom.MergeErrors(err, loom.InvalidEnumValueError("q[*]", e, []any{"val"}))
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryArrayStringValidatePayload(q)

		return payload, nil
	}
}
`


