package testdata


var PayloadBodyArrayUserDecodeCode = `// DecodeMethodBodyArrayUserRequest returns a decoder for requests sent to the
// ServiceBodyArrayUser MethodBodyArrayUser endpoint.
func DecodeMethodBodyArrayUserRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body MethodBodyArrayUserRequestBody
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
		err = ValidateMethodBodyArrayUserRequestBody(&body)
		if err != nil {
			return nil, err
		}
		payload := NewMethodBodyArrayUserPayload(&body)

		return payload, nil
	}
}
`


