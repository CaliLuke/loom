package testdata


var PayloadQueryArrayAnyDecodeCode = `// DecodeMethodQueryArrayAnyRequest returns a decoder for requests sent to the
// ServiceQueryArrayAny MethodQueryArrayAny endpoint.
func DecodeMethodQueryArrayAnyRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q []any
		)
		{
			qRaw := r.URL.Query()["q"]
			if qRaw != nil {
				q = make([]any, len(qRaw))
				for i, rv := range qRaw {
					q[i] = rv
				}
			}
		}
		payload := NewMethodQueryArrayAnyPayload(q)

		return payload, nil
	}
}
`


