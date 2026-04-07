package testdata


var PayloadMapQueryPrimitivePrimitiveDecodeCode = `// DecodeMapQueryPrimitivePrimitiveRequest returns a decoder for requests sent
// to the ServiceMapQueryPrimitivePrimitive MapQueryPrimitivePrimitive endpoint.
func DecodeMapQueryPrimitivePrimitiveRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			query map[string]string
			err   error
		)
		{
			queryRaw := r.URL.Query()
			if len(queryRaw) == 0 {
				err = loom.MergeErrors(err, loom.MissingFieldError("query", "query string"))
			}
			for keyRaw, valRaw := range queryRaw {
				if strings.HasPrefix(keyRaw, "query[") {
					if query == nil {
						query = make(map[string]string)
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
					query[keya] = valRaw[0]
				}
			}
		}
		if err != nil {
			return nil, err
		}
		payload := query

		return payload, nil
	}
}
`


