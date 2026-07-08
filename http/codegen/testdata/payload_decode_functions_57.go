package testdata

var PayloadPathPrimitiveArrayStringValidateDecodeCode = `// DecodeMethodPathPrimitiveArrayStringValidateRequest returns a decoder for
// requests sent to the ServicePathPrimitiveArrayStringValidate
// MethodPathPrimitiveArrayStringValidate endpoint.
func DecodeMethodPathPrimitiveArrayStringValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			p   []string
			err error

			params = mux.Vars(r)
		)
		{
			pRaw := params["p"]
			pRawSlice := strings.Split(pRaw, ",")
			p = make([]string, len(pRawSlice))
			for i, rv := range pRawSlice {
				rvDecoded, err2 := url.PathUnescape(rv)
				if err2 != nil {
					err = loom.MergeErrors(err, loom.InvalidFieldTypeError("p", pRaw, "path-escaped array"))
				}
				rv = rvDecoded
				p[i] = rv
			}
		}
		if len(p) < 1 {
			err = loom.MergeErrors(err, loom.InvalidLengthError("p", p, len(p), 1, true))
		}
		for _, e := range p {
			if !(e == "val") {
				err = loom.MergeErrors(err, loom.InvalidEnumValueError("p[*]", e, []any{"val"}))
			}
		}
		if err != nil {
			return nil, err
		}
		payload := p

		return payload, nil
	}
}
`
