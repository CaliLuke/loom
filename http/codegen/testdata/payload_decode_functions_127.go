package testdata


var WithParamsAndHeadersBlockDecodeCode = `// DecodeMethodARequest returns a decoder for requests sent to the
// ServiceWithParamsAndHeadersBlock MethodA endpoint.
func DecodeMethodARequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body MethodARequestBody
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

		var (
			path                      uint
			optional                  *int
			optionalButRequiredParam  float32
			required                  string
			optionalButRequiredHeader float32

			params = mux.Vars(r)
		)
		{
			pathRaw := params["path"]
			v, err2 := strconv.ParseUint(pathRaw, 10, strconv.IntSize)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("path", pathRaw, "unsigned integer"))
			}
			path = uint(v)
		}
		qp := r.URL.Query()
		{
			optionalRaw := qp.Get("optional")
			if optionalRaw != "" {
				v, err2 := strconv.ParseInt(optionalRaw, 10, strconv.IntSize)
				if err2 != nil {
					err = loom.MergeErrors(err, loom.InvalidFieldTypeError("optional", optionalRaw, "integer"))
				}
				pv := int(v)
				optional = &pv
			}
		}
		{
			optionalButRequiredParamRaw := qp.Get("optional_but_required_param")
			if optionalButRequiredParamRaw == "" {
				err = loom.MergeErrors(err, loom.MissingFieldError("optional_but_required_param", "query string"))
			}
			v, err2 := strconv.ParseFloat(optionalButRequiredParamRaw, 32)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("optional_but_required_param", optionalButRequiredParamRaw, "float"))
			}
			optionalButRequiredParam = float32(v)
		}
		required = r.Header.Get("required")
		if required == "" {
			err = loom.MergeErrors(err, loom.MissingFieldError("required", "header"))
		}
		{
			optionalButRequiredHeaderRaw := r.Header.Get("optional_but_required_header")
			if optionalButRequiredHeaderRaw == "" {
				err = loom.MergeErrors(err, loom.MissingFieldError("optional_but_required_header", "header"))
			}
			v, err2 := strconv.ParseFloat(optionalButRequiredHeaderRaw, 32)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("optional_but_required_header", optionalButRequiredHeaderRaw, "float"))
			}
			optionalButRequiredHeader = float32(v)
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodAPayload(&body, path, optional, optionalButRequiredParam, required, optionalButRequiredHeader)

		return payload, nil
	}
}
`


