package testdata


var PayloadQueryPrimitiveBoolValidateDecodeCode = `// DecodeMethodQueryPrimitiveBoolValidateRequest returns a decoder for requests
// sent to the ServiceQueryPrimitiveBoolValidate
// MethodQueryPrimitiveBoolValidate endpoint.
func DecodeMethodQueryPrimitiveBoolValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   bool
			err error
		)
		{
			qRaw := r.URL.Query().Get("q")
			if qRaw == "" {
				err = loom.MergeErrors(err, loom.MissingFieldError("q", "query string"))
			}
			v, err2 := strconv.ParseBool(qRaw)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "boolean"))
			}
			q = v
		}
		if !(q == true) {
			err = loom.MergeErrors(err, loom.InvalidEnumValueError("q", q, []any{true}))
		}
		if err != nil {
			return nil, err
		}
		payload := q

		return payload, nil
	}
}
`


