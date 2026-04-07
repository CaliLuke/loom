package testdata


var PayloadQueryMapStringBoolValidateDecodeCode = `// DecodeMethodQueryMapStringBoolValidateRequest returns a decoder for requests
// sent to the ServiceQueryMapStringBoolValidate
// MethodQueryMapStringBoolValidate endpoint.
func DecodeMethodQueryMapStringBoolValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   map[string]bool
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
						q = make(map[string]bool)
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
					var vala bool
					{
						valaRaw := valRaw[0]
						v, err2 := strconv.ParseBool(valaRaw)
						if err2 != nil {
							err = loom.MergeErrors(err, loom.InvalidFieldTypeError("query", valaRaw, "boolean"))
						}
						vala = v
					}
					q[keya] = vala
				}
			}
		}
		if len(q) < 1 {
			err = loom.MergeErrors(err, loom.InvalidLengthError("q", q, len(q), 1, true))
		}
		for k, v := range q {
			if !(k == "key") {
				err = loom.MergeErrors(err, loom.InvalidEnumValueError("q.key", k, []any{"key"}))
			}
			if !(v == true) {
				err = loom.MergeErrors(err, loom.InvalidEnumValueError("q[key]", v, []any{true}))
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryMapStringBoolValidatePayload(q)

		return payload, nil
	}
}
`


