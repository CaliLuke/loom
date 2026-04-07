package testdata


var QueryIntAliasValidateDecodeCode = `// DecodeMethodARequest returns a decoder for requests sent to the
// ServiceQueryIntAliasValidate MethodA endpoint.
func DecodeMethodARequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			int_   *int
			int32_ *int32
			int64_ *int64
			err    error
		)
		qp := r.URL.Query()
		{
			int_Raw := qp.Get("int")
			if int_Raw != "" {
				v, err2 := strconv.ParseInt(int_Raw, 10, strconv.IntSize)
				if err2 != nil {
					err = loom.MergeErrors(err, loom.InvalidFieldTypeError("int", int_Raw, "integer"))
				}
				pv := int(v)
				int_ = &pv
			}
		}
		if int_ != nil {
			if *int_ < 10 {
				err = loom.MergeErrors(err, loom.InvalidRangeError("int", *int_, 10, true))
			}
		}
		{
			int32_Raw := qp.Get("int32")
			if int32_Raw != "" {
				v, err2 := strconv.ParseInt(int32_Raw, 10, 32)
				if err2 != nil {
					err = loom.MergeErrors(err, loom.InvalidFieldTypeError("int32", int32_Raw, "integer"))
				}
				pv := int32(v)
				int32_ = &pv
			}
		}
		if int32_ != nil {
			if *int32_ > 100 {
				err = loom.MergeErrors(err, loom.InvalidRangeError("int32", *int32_, 100, false))
			}
		}
		{
			int64_Raw := qp.Get("int64")
			if int64_Raw != "" {
				v, err2 := strconv.ParseInt(int64_Raw, 10, 64)
				if err2 != nil {
					err = loom.MergeErrors(err, loom.InvalidFieldTypeError("int64", int64_Raw, "integer"))
				}
				int64_ = &v
			}
		}
		if int64_ != nil {
			if *int64_ < 0 {
				err = loom.MergeErrors(err, loom.InvalidRangeError("int64", *int64_, 0, true))
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodAPayload(int_, int32_, int64_)

		return payload, nil
	}
}
`


