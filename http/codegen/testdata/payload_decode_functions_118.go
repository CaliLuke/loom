package testdata


var PayloadBodyQueryPathUserDecodeCode = `// DecodeMethodBodyQueryPathUserRequest returns a decoder for requests sent to
// the ServiceBodyQueryPathUser MethodBodyQueryPathUser endpoint.
func DecodeMethodBodyQueryPathUserRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body MethodBodyQueryPathUserRequestBody
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
			c2 string
			b  *string

			params = mux.Vars(r)
		)
		c2 = params["c"]
		bRaw := r.URL.Query().Get("b")
		if bRaw != "" {
			b = &bRaw
		}
		payload := NewMethodBodyQueryPathUserPayloadType(&body, c2, b)

		return payload, nil
	}
}
`


