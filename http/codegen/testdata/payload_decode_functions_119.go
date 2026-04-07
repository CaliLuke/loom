package testdata


var PayloadBodyQueryPathUserValidateDecodeCode = `// DecodeMethodBodyQueryPathUserValidateRequest returns a decoder for requests
// sent to the ServiceBodyQueryPathUserValidate MethodBodyQueryPathUserValidate
// endpoint.
func DecodeMethodBodyQueryPathUserValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body MethodBodyQueryPathUserValidateRequestBody
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
		err = ValidateMethodBodyQueryPathUserValidateRequestBody(&body)
		if err != nil {
			return nil, err
		}

		var (
			c2 string
			b  string

			params = mux.Vars(r)
		)
		c2 = params["c"]
		err = loom.MergeErrors(err, loom.ValidatePattern("c", c2, "patternc"))
		b = r.URL.Query().Get("b")
		if b == "" {
			err = loom.MergeErrors(err, loom.MissingFieldError("b", "query string"))
		}
		err = loom.MergeErrors(err, loom.ValidatePattern("b", b, "patternb"))
		if err != nil {
			return nil, err
		}
		payload := NewMethodBodyQueryPathUserValidatePayloadType(&body, c2, b)

		return payload, nil
	}
}
`


