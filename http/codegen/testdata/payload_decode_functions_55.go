package testdata


var PayloadPathPrimitiveStringValidateDecodeCode = `// DecodeMethodPathPrimitiveStringValidateRequest returns a decoder for
// requests sent to the ServicePathPrimitiveStringValidate
// MethodPathPrimitiveStringValidate endpoint.
func DecodeMethodPathPrimitiveStringValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			p   string
			err error

			params = mux.Vars(r)
		)
		p = params["p"]
		if !(p == "val") {
			err = loom.MergeErrors(err, loom.InvalidEnumValueError("p", p, []any{"val"}))
		}
		if err != nil {
			return nil, err
		}
		payload := p

		return payload, nil
	}
}
`


