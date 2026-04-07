package testdata


var PayloadQueryPrimitiveMapStringArrayStringValidateDecodeCode = `// DecodeMethodQueryPrimitiveMapStringArrayStringValidateRequest returns a
// decoder for requests sent to the
// ServiceQueryPrimitiveMapStringArrayStringValidate
// MethodQueryPrimitiveMapStringArrayStringValidate endpoint.
func DecodeMethodQueryPrimitiveMapStringArrayStringValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   map[string][]string
			err error
		)
		{
			qRaw := r.URL.Query()
			if len(qRaw) == 0 {
				err = loom.MergeErrors(err, loom.MissingFieldError("q", "query string"))
			}
			for keyRaw, valRaw := range qRaw {
				if strings.HasPrefix(keyRaw, "q[") {
					if q == nil {
						q = make(map[string][]string)
					}
					var keya string
					{
						openIdx := strings.IndexRune(keyRaw, '[')
						closeIdx := strings.IndexRune(keyRaw, ']')
						if openIdx == -1 || closeIdx == -1 || closeIdx <= openIdx {
							err = loom.MergeErrors(err, loom.DecodePayloadError("invalid query string: malformed brackets"))
						} else {
							keya = keyRaw[openIdx+1 : closeIdx]
						}
					}
					q[keya] = valRaw
				}
			}
		}
		if len(q) < 1 {
			err = loom.MergeErrors(err, loom.InvalidLengthError("q", q, len(q), 1, true))
		}
		for k, v := range q {
			err = loom.MergeErrors(err, loom.ValidatePattern("q.key", k, "key"))
			if len(v) < 2 {
				err = loom.MergeErrors(err, loom.InvalidLengthError("q[key]", v, len(v), 2, true))
			}
			for _, e := range v {
				err = loom.MergeErrors(err, loom.ValidatePattern("q[key][*]", e, "val"))
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


