package testdata


var PayloadBodyObjectRequiredDecodeCode = `// DecodeMethodBodyObjectRequiredRequest returns a decoder for requests sent to
// the ServiceBodyObjectRequired MethodBodyObjectRequired endpoint.
func DecodeMethodBodyObjectRequiredRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body struct {
				B *string ` + "`" + `form:"b" json:"b" xml:"b"` + "`" + `
			}
			err error
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
		if body.B == nil {
			err = loom.MergeErrors(err, loom.MissingFieldError("b", "body"))
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodBodyObjectRequiredPayload(body)

		return payload, nil
	}
}
`


