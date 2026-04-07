package testdata


var PayloadBodyUnionUserDecodeCode = `// DecodeMethodBodyUnionUserRequest returns a decoder for requests sent to the
// ServiceBodyUnionUser MethodBodyUnionUser endpoint.
func DecodeMethodBodyUnionUserRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body MethodBodyUnionUserRequestBody
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
		payload := NewMethodBodyUnionUserUnionUser(&body)

		return payload, nil
	}
}
`


