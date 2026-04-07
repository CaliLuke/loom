package testdata


var PayloadQueryPrimitiveArrayBoolValidateDecodeCode = `// DecodeMethodQueryPrimitiveArrayBoolValidateRequest returns a decoder for
// requests sent to the ServiceQueryPrimitiveArrayBoolValidate
// MethodQueryPrimitiveArrayBoolValidate endpoint.
func DecodeMethodQueryPrimitiveArrayBoolValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   []bool
			err error
		)
		{
			qRaw := r.URL.Query()["q"]
			if qRaw == nil {
				err = loom.MergeErrors(err, loom.MissingFieldError("q", "query string"))
			}
			q = make([]bool, len(qRaw))
			for i, rv := range qRaw {
				v, err2 := strconv.ParseBool(rv)
				if err2 != nil {
					err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "array of booleans"))
				}
				q[i] = v
			}
		}
		if len(q) < 1 {
			err = loom.MergeErrors(err, loom.InvalidLengthError("q", q, len(q), 1, true))
		}
		for _, e := range q {
			if !(e == true) {
				err = loom.MergeErrors(err, loom.InvalidEnumValueError("q[*]", e, []any{true}))
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


