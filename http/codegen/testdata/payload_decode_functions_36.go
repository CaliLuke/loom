package testdata


var PayloadQueryPrimitiveStringValidateDecodeCode = `// DecodeMethodQueryPrimitiveStringValidateRequest returns a decoder for
// requests sent to the ServiceQueryPrimitiveStringValidate
// MethodQueryPrimitiveStringValidate endpoint.
func DecodeMethodQueryPrimitiveStringValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   string
			err error
		)
		q = r.URL.Query().Get("q")
		if q == "" {
			err = loom.MergeErrors(err, loom.MissingFieldError("q", "query string"))
		}
		if !(q == "val") {
			err = loom.MergeErrors(err, loom.InvalidEnumValueError("q", q, []any{"val"}))
		}
		if err != nil {
			return nil, err
		}
		payload := q

		return payload, nil
	}
}
`


