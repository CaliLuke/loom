package testdata


var PayloadBodyQueryObjectDecodeCode = `// DecodeMethodBodyQueryObjectRequest returns a decoder for requests sent to
// the ServiceBodyQueryObject MethodBodyQueryObject endpoint.
func DecodeMethodBodyQueryObjectRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body MethodBodyQueryObjectRequestBody
			err  error
		)
		err = decoder(r).Decode(&body)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, loom.MissingPayloadError()
			}
			var gerr *loom.ServiceError
			if errors.As(err, &gerr) {
				return nil, gerr
			}
			return nil, loom.DecodePayloadError(err.Error())
		}

		var (
			b *string
		)
		bRaw := r.URL.Query().Get("b")
		if bRaw != "" {
			b = &bRaw
		}
		payload := NewMethodBodyQueryObjectPayload(&body, b)

		return payload, nil
	}
}
`


