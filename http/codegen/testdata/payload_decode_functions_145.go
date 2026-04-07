package testdata


var PayloadBodyCustomNameDecodeCode = `// DecodeMethodBodyCustomNameRequest returns a decoder for requests sent to the
// ServiceBodyCustomName MethodBodyCustomName endpoint.
func DecodeMethodBodyCustomNameRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body MethodBodyCustomNameRequestBody
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
		payload := NewMethodBodyCustomNamePayload(&body)

		return payload, nil
	}
}
`


