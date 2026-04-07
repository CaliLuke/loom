package testdata


var PayloadBodyPathUserDecodeCode = `// DecodeMethodBodyPathUserRequest returns a decoder for requests sent to the
// ServiceBodyPathUser MethodBodyPathUser endpoint.
func DecodeMethodBodyPathUserRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body MethodBodyPathUserRequestBody
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
			b string

			params = mux.Vars(r)
		)
		b = params["b"]
		payload := NewMethodBodyPathUserPayloadType(&body, b)

		return payload, nil
	}
}
`


