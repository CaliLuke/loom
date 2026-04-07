package testdata


var PayloadQueryMapStringArrayBoolDecodeCode = `// DecodeMethodQueryMapStringArrayBoolRequest returns a decoder for requests
// sent to the ServiceQueryMapStringArrayBool MethodQueryMapStringArrayBool
// endpoint.
func DecodeMethodQueryMapStringArrayBoolRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   map[string][]bool
			err error
		)
		{
			qRaw := r.URL.Query()
			if len(qRaw) != 0 {
				for keyRaw, valRaw := range qRaw {
					if strings.HasPrefix(keyRaw, "q[") {
						if q == nil {
							q = make(map[string][]bool)
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
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryMapStringArrayBoolPayload(q)

		return payload, nil
	}
}
`


