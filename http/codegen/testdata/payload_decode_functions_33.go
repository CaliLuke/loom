package testdata


var PayloadQueryMapBoolArrayStringValidateDecodeCode = `// DecodeMethodQueryMapBoolArrayStringValidateRequest returns a decoder for
// requests sent to the ServiceQueryMapBoolArrayStringValidate
// MethodQueryMapBoolArrayStringValidate endpoint.
func DecodeMethodQueryMapBoolArrayStringValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   map[bool][]string
			err error
		)
		{
			qRaw := r.URL.Query()
			if len(qRaw) != 0 {
				for keyRaw, valRaw := range qRaw {
					if strings.HasPrefix(keyRaw, "q[") {
						if q == nil {
							q = make(map[bool][]string)
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
						q[keya] = valRaw
					}
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
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryMapBoolArrayStringValidatePayload(q)

		return payload, nil
	}
}
`


