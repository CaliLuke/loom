package testdata


var PayloadQueryPrimitiveArrayStringValidateDecodeCode = `// DecodeMethodQueryPrimitiveArrayStringValidateRequest returns a decoder for
// requests sent to the ServiceQueryPrimitiveArrayStringValidate
// MethodQueryPrimitiveArrayStringValidate endpoint.
func DecodeMethodQueryPrimitiveArrayStringValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
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
		payload := q

		return payload, nil
	}
}
`


