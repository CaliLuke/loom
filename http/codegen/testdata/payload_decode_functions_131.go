package testdata


var QueryArrayAliasValidateDecodeCode = `// DecodeMethodARequest returns a decoder for requests sent to the
// ServiceQueryArrayAliasValidate MethodA endpoint.
func DecodeMethodARequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			array []uint
			err   error
		)
		{
			arrayRaw := r.URL.Query()["array"]
			if arrayRaw != nil {
				array = make([]uint, len(arrayRaw))
				for i, rv := range arrayRaw {
					v, err2 := strconv.ParseUint(rv, 10, strconv.IntSize)
					if err2 != nil {
						err = loom.MergeErrors(err, loom.InvalidFieldTypeError("array", arrayRaw, "array of unsigned integers"))
					}
					array[i] = uint(v)
				}
			}
		}
		if len(array) < 3 {
			err = loom.MergeErrors(err, loom.InvalidLengthError("array", array, len(array), 3, true))
		}
		for _, e := range array {
			if e < 10 {
				err = loom.MergeErrors(err, loom.InvalidRangeError("array[*]", e, 10, true))
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodAPayload(array)

		return payload, nil
	}
}
`


