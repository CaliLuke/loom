package testdata


var PayloadQueryPrimitiveMapBoolArrayBoolValidateDecodeCode = `// DecodeMethodQueryPrimitiveMapBoolArrayBoolValidateRequest returns a decoder
// for requests sent to the ServiceQueryPrimitiveMapBoolArrayBoolValidate
// MethodQueryPrimitiveMapBoolArrayBoolValidate endpoint.
func DecodeMethodQueryPrimitiveMapBoolArrayBoolValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   map[bool][]bool
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
						q = make(map[bool][]bool)
					}
					var keya bool
					{
						openIdx := strings.IndexRune(keyRaw, '[')
						closeIdx := strings.IndexRune(keyRaw, ']')
						if openIdx == -1 || closeIdx == -1 || closeIdx <= openIdx {
							err = loom.MergeErrors(err, loom.DecodePayloadError("invalid query string: malformed brackets"))
						} else {
							keyaRaw := keyRaw[openIdx+1 : closeIdx]
							v, err2 := strconv.ParseBool(keyaRaw)
							if err2 != nil {
								err = loom.MergeErrors(err, loom.InvalidFieldTypeError("query", keyaRaw, "boolean"))
							}
							keya = v
						}
					}
					var val []bool
					{
						val = make([]bool, len(valRaw))
						for i, rv := range valRaw {
							v, err2 := strconv.ParseBool(rv)
							if err2 != nil {
								err = loom.MergeErrors(err, loom.InvalidFieldTypeError("query", valRaw, "array of booleans"))
							}
							val[i] = v
						}
					}
					q[keya] = val
				}
			}
		}
		if len(q) < 1 {
			err = loom.MergeErrors(err, loom.InvalidLengthError("q", q, len(q), 1, true))
		}
		for k, v := range q {
			if !(k == true) {
				err = loom.MergeErrors(err, loom.InvalidEnumValueError("q.key", k, []any{true}))
			}
			if len(v) < 2 {
				err = loom.MergeErrors(err, loom.InvalidLengthError("q[key]", v, len(v), 2, true))
			}
			for _, e := range v {
				if !(e == false) {
					err = loom.MergeErrors(err, loom.InvalidEnumValueError("q[key][*]", e, []any{false}))
				}
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


