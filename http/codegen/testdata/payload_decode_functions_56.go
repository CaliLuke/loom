package testdata


var PayloadPathPrimitiveBoolValidateDecodeCode = `// DecodeMethodPathPrimitiveBoolValidateRequest returns a decoder for requests
// sent to the ServicePathPrimitiveBoolValidate MethodPathPrimitiveBoolValidate
// endpoint.
func DecodeMethodPathPrimitiveBoolValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			p   bool
			err error

			params = mux.Vars(r)
		)
		{
			pRaw := params["p"]
			v, err2 := strconv.ParseBool(pRaw)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("p", pRaw, "boolean"))
			}
			p = v
		}
		if !(p == true) {
			err = loom.MergeErrors(err, loom.InvalidEnumValueError("p", p, []any{true}))
		}
		if err != nil {
			return nil, err
		}
		payload := p

		return payload, nil
	}
}
`


