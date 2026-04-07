package testdata


var PayloadBodyUnionDecodeCode = `// DecodeMethodBodyUnionRequest returns a decoder for requests sent to the
// ServiceBodyUnion MethodBodyUnion endpoint.
func DecodeMethodBodyUnionRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body MethodBodyUnionRequestBody
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
		payload := NewMethodBodyUnionUnion(&body)

		return payload, nil
	}
}
`


