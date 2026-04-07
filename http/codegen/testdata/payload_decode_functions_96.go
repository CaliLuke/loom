package testdata


var PayloadBodyMapUserDecodeCode = `// DecodeMethodBodyMapUserRequest returns a decoder for requests sent to the
// ServiceBodyMapUser MethodBodyMapUser endpoint.
func DecodeMethodBodyMapUserRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body MethodBodyMapUserRequestBody
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
		err = ValidateMethodBodyMapUserRequestBody(&body)
		if err != nil {
			return nil, err
		}
		payload := NewMethodBodyMapUserPayload(&body)

		return payload, nil
	}
}
`


