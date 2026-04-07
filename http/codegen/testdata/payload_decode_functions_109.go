package testdata


var PayloadBodyQueryObjectValidateDecodeCode = `// DecodeMethodBodyQueryObjectValidateRequest returns a decoder for requests
// sent to the ServiceBodyQueryObjectValidate MethodBodyQueryObjectValidate
// endpoint.
func DecodeMethodBodyQueryObjectValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body MethodBodyQueryObjectValidateRequestBody
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
		err = ValidateMethodBodyQueryObjectValidateRequestBody(&body)
		if err != nil {
			return nil, err
		}

		var (
			b string
		)
		b = r.URL.Query().Get("b")
		if b == "" {
			err = loom.MergeErrors(err, loom.MissingFieldError("b", "query string"))
		}
		err = loom.MergeErrors(err, loom.ValidatePattern("b", b, "patternb"))
		if err != nil {
			return nil, err
		}
		payload := NewMethodBodyQueryObjectValidatePayload(&body, b)

		return payload, nil
	}
}
`


