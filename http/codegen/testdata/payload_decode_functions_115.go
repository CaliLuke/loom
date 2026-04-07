package testdata


var PayloadBodyPathUserValidateDecodeCode = `// DecodeMethodUserBodyPathValidateRequest returns a decoder for requests sent
// to the ServiceBodyPathUserValidate MethodUserBodyPathValidate endpoint.
func DecodeMethodUserBodyPathValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body MethodUserBodyPathValidateRequestBody
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
		err = ValidateMethodUserBodyPathValidateRequestBody(&body)
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
		payload := NewMethodUserBodyPathValidatePayloadType(&body, b)

		return payload, nil
	}
}
`


