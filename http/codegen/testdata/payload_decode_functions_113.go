package testdata


var PayloadBodyPathObjectValidateDecodeCode = `// DecodeMethodBodyPathObjectValidateRequest returns a decoder for requests
// sent to the ServiceBodyPathObjectValidate MethodBodyPathObjectValidate
// endpoint.
func DecodeMethodBodyPathObjectValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body MethodBodyPathObjectValidateRequestBody
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
		err = ValidateMethodBodyPathObjectValidateRequestBody(&body)
		if err != nil {
			return nil, err
		}

		var (
			b string

			params = mux.Vars(r)
		)
		b = params["b"]
		err = loom.MergeErrors(err, loom.ValidatePattern("b", b, "patternb"))
		if err != nil {
			return nil, err
		}
		payload := NewMethodBodyPathObjectValidatePayload(&body, b)

		return payload, nil
	}
}
`


