package testdata


var PayloadBodyMapStringDecodeCode = `// DecodeMethodBodyMapStringRequest returns a decoder for requests sent to the
// ServiceBodyMapString MethodBodyMapString endpoint.
func DecodeMethodBodyMapStringRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body MethodBodyMapStringRequestBody
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
		payload := NewMethodBodyMapStringPayload(&body)

		return payload, nil
	}
}
`


