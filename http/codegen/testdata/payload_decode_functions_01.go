package testdata


var PayloadQueryBoolDecodeCode = `// DecodeMethodQueryBoolRequest returns a decoder for requests sent to the
// ServiceQueryBool MethodQueryBool endpoint.
func DecodeMethodQueryBoolRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   *bool
			err error
		)
		{
			qRaw := r.URL.Query().Get("q")
			if qRaw != "" {
				v, err2 := strconv.ParseBool(qRaw)
				if err2 != nil {
					err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "boolean"))
				}
				q = &v
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryBoolPayload(q)

		return payload, nil
	}
}
`

var PayloadQueryBoolValidateDecodeCode = `// DecodeMethodQueryBoolValidateRequest returns a decoder for requests sent to
// the ServiceQueryBoolValidate MethodQueryBoolValidate endpoint.
func DecodeMethodQueryBoolValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   bool
			err error
		)
		{
			qRaw := r.URL.Query().Get("q")
			if qRaw == "" {
				err = loom.MergeErrors(err, loom.MissingFieldError("q", "query string"))
			}
			v, err2 := strconv.ParseBool(qRaw)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "boolean"))
			}
			q = v
		}
		if !(q == true) {
			err = loom.MergeErrors(err, loom.InvalidEnumValueError("q", q, []any{true}))
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryBoolValidatePayload(q)

		return payload, nil
	}
}
`

var PayloadQueryIntDecodeCode = `// DecodeMethodQueryIntRequest returns a decoder for requests sent to the
// ServiceQueryInt MethodQueryInt endpoint.
func DecodeMethodQueryIntRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   *int
			err error
		)
		{
			qRaw := r.URL.Query().Get("q")
			if qRaw != "" {
				v, err2 := strconv.ParseInt(qRaw, 10, strconv.IntSize)
				if err2 != nil {
					err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "integer"))
				}
				pv := int(v)
				q = &pv
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryIntPayload(q)

		return payload, nil
	}
}
`

var PayloadQueryIntValidateDecodeCode = `// DecodeMethodQueryIntValidateRequest returns a decoder for requests sent to
// the ServiceQueryIntValidate MethodQueryIntValidate endpoint.
func DecodeMethodQueryIntValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   int
			err error
		)
		{
			qRaw := r.URL.Query().Get("q")
			if qRaw == "" {
				err = loom.MergeErrors(err, loom.MissingFieldError("q", "query string"))
			}
			v, err2 := strconv.ParseInt(qRaw, 10, strconv.IntSize)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "integer"))
			}
			q = int(v)
		}
		if q < 1 {
			err = loom.MergeErrors(err, loom.InvalidRangeError("q", q, 1, true))
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryIntValidatePayload(q)

		return payload, nil
	}
}
`

var PayloadQueryInt32DecodeCode = `// DecodeMethodQueryInt32Request returns a decoder for requests sent to the
// ServiceQueryInt32 MethodQueryInt32 endpoint.
func DecodeMethodQueryInt32Request(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   *int32
			err error
		)
		{
			qRaw := r.URL.Query().Get("q")
			if qRaw != "" {
				v, err2 := strconv.ParseInt(qRaw, 10, 32)
				if err2 != nil {
					err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "integer"))
				}
				pv := int32(v)
				q = &pv
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryInt32Payload(q)

		return payload, nil
	}
}
`

var PayloadQueryInt32ValidateDecodeCode = `// DecodeMethodQueryInt32ValidateRequest returns a decoder for requests sent to
// the ServiceQueryInt32Validate MethodQueryInt32Validate endpoint.
func DecodeMethodQueryInt32ValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   int32
			err error
		)
		{
			qRaw := r.URL.Query().Get("q")
			if qRaw == "" {
				err = loom.MergeErrors(err, loom.MissingFieldError("q", "query string"))
			}
			v, err2 := strconv.ParseInt(qRaw, 10, 32)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "integer"))
			}
			q = int32(v)
		}
		if q < 1 {
			err = loom.MergeErrors(err, loom.InvalidRangeError("q", q, 1, true))
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryInt32ValidatePayload(q)

		return payload, nil
	}
}
`

var PayloadQueryInt64DecodeCode = `// DecodeMethodQueryInt64Request returns a decoder for requests sent to the
// ServiceQueryInt64 MethodQueryInt64 endpoint.
func DecodeMethodQueryInt64Request(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   *int64
			err error
		)
		{
			qRaw := r.URL.Query().Get("q")
			if qRaw != "" {
				v, err2 := strconv.ParseInt(qRaw, 10, 64)
				if err2 != nil {
					err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "integer"))
				}
				q = &v
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryInt64Payload(q)

		return payload, nil
	}
}
`

var PayloadQueryInt64ValidateDecodeCode = `// DecodeMethodQueryInt64ValidateRequest returns a decoder for requests sent to
// the ServiceQueryInt64Validate MethodQueryInt64Validate endpoint.
func DecodeMethodQueryInt64ValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   int64
			err error
		)
		{
			qRaw := r.URL.Query().Get("q")
			if qRaw == "" {
				err = loom.MergeErrors(err, loom.MissingFieldError("q", "query string"))
			}
			v, err2 := strconv.ParseInt(qRaw, 10, 64)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "integer"))
			}
			q = v
		}
		if q < 1 {
			err = loom.MergeErrors(err, loom.InvalidRangeError("q", q, 1, true))
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryInt64ValidatePayload(q)

		return payload, nil
	}
}
`

var PayloadQueryUIntDecodeCode = `// DecodeMethodQueryUIntRequest returns a decoder for requests sent to the
// ServiceQueryUInt MethodQueryUInt endpoint.
func DecodeMethodQueryUIntRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   *uint
			err error
		)
		{
			qRaw := r.URL.Query().Get("q")
			if qRaw != "" {
				v, err2 := strconv.ParseUint(qRaw, 10, strconv.IntSize)
				if err2 != nil {
					err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "unsigned integer"))
				}
				pv := uint(v)
				q = &pv
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryUIntPayload(q)

		return payload, nil
	}
}
`

var PayloadQueryUIntValidateDecodeCode = `// DecodeMethodQueryUIntValidateRequest returns a decoder for requests sent to
// the ServiceQueryUIntValidate MethodQueryUIntValidate endpoint.
func DecodeMethodQueryUIntValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   uint
			err error
		)
		{
			qRaw := r.URL.Query().Get("q")
			if qRaw == "" {
				err = loom.MergeErrors(err, loom.MissingFieldError("q", "query string"))
			}
			v, err2 := strconv.ParseUint(qRaw, 10, strconv.IntSize)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "unsigned integer"))
			}
			q = uint(v)
		}
		if q < 1 {
			err = loom.MergeErrors(err, loom.InvalidRangeError("q", q, 1, true))
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryUIntValidatePayload(q)

		return payload, nil
	}
}
`

var PayloadQueryUInt32DecodeCode = `// DecodeMethodQueryUInt32Request returns a decoder for requests sent to the
// ServiceQueryUInt32 MethodQueryUInt32 endpoint.
func DecodeMethodQueryUInt32Request(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   *uint32
			err error
		)
		{
			qRaw := r.URL.Query().Get("q")
			if qRaw != "" {
				v, err2 := strconv.ParseUint(qRaw, 10, 32)
				if err2 != nil {
					err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "unsigned integer"))
				}
				pv := uint32(v)
				q = &pv
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryUInt32Payload(q)

		return payload, nil
	}
}
`

var PayloadQueryUInt32ValidateDecodeCode = `// DecodeMethodQueryUInt32ValidateRequest returns a decoder for requests sent
// to the ServiceQueryUInt32Validate MethodQueryUInt32Validate endpoint.
func DecodeMethodQueryUInt32ValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   uint32
			err error
		)
		{
			qRaw := r.URL.Query().Get("q")
			if qRaw == "" {
				err = loom.MergeErrors(err, loom.MissingFieldError("q", "query string"))
			}
			v, err2 := strconv.ParseUint(qRaw, 10, 32)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "unsigned integer"))
			}
			q = uint32(v)
		}
		if q < 1 {
			err = loom.MergeErrors(err, loom.InvalidRangeError("q", q, 1, true))
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryUInt32ValidatePayload(q)

		return payload, nil
	}
}
`

var PayloadQueryUInt64DecodeCode = `// DecodeMethodQueryUInt64Request returns a decoder for requests sent to the
// ServiceQueryUInt64 MethodQueryUInt64 endpoint.
func DecodeMethodQueryUInt64Request(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   *uint64
			err error
		)
		{
			qRaw := r.URL.Query().Get("q")
			if qRaw != "" {
				v, err2 := strconv.ParseUint(qRaw, 10, 64)
				if err2 != nil {
					err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "unsigned integer"))
				}
				q = &v
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryUInt64Payload(q)

		return payload, nil
	}
}
`

var PayloadQueryUInt64ValidateDecodeCode = `// DecodeMethodQueryUInt64ValidateRequest returns a decoder for requests sent
// to the ServiceQueryUInt64Validate MethodQueryUInt64Validate endpoint.
func DecodeMethodQueryUInt64ValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   uint64
			err error
		)
		{
			qRaw := r.URL.Query().Get("q")
			if qRaw == "" {
				err = loom.MergeErrors(err, loom.MissingFieldError("q", "query string"))
			}
			v, err2 := strconv.ParseUint(qRaw, 10, 64)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "unsigned integer"))
			}
			q = v
		}
		if q < 1 {
			err = loom.MergeErrors(err, loom.InvalidRangeError("q", q, 1, true))
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryUInt64ValidatePayload(q)

		return payload, nil
	}
}
`

var PayloadQueryFloat32DecodeCode = `// DecodeMethodQueryFloat32Request returns a decoder for requests sent to the
// ServiceQueryFloat32 MethodQueryFloat32 endpoint.
func DecodeMethodQueryFloat32Request(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   *float32
			err error
		)
		{
			qRaw := r.URL.Query().Get("q")
			if qRaw != "" {
				v, err2 := strconv.ParseFloat(qRaw, 32)
				if err2 != nil {
					err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "float"))
				}
				pv := float32(v)
				q = &pv
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryFloat32Payload(q)

		return payload, nil
	}
}
`

var PayloadQueryFloat32ValidateDecodeCode = `// DecodeMethodQueryFloat32ValidateRequest returns a decoder for requests sent
// to the ServiceQueryFloat32Validate MethodQueryFloat32Validate endpoint.
func DecodeMethodQueryFloat32ValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   float32
			err error
		)
		{
			qRaw := r.URL.Query().Get("q")
			if qRaw == "" {
				err = loom.MergeErrors(err, loom.MissingFieldError("q", "query string"))
			}
			v, err2 := strconv.ParseFloat(qRaw, 32)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "float"))
			}
			q = float32(v)
		}
		if q < 1 {
			err = loom.MergeErrors(err, loom.InvalidRangeError("q", q, 1, true))
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryFloat32ValidatePayload(q)

		return payload, nil
	}
}
`

var PayloadQueryFloat64DecodeCode = `// DecodeMethodQueryFloat64Request returns a decoder for requests sent to the
// ServiceQueryFloat64 MethodQueryFloat64 endpoint.
func DecodeMethodQueryFloat64Request(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   *float64
			err error
		)
		{
			qRaw := r.URL.Query().Get("q")
			if qRaw != "" {
				v, err2 := strconv.ParseFloat(qRaw, 64)
				if err2 != nil {
					err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "float"))
				}
				q = &v
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryFloat64Payload(q)

		return payload, nil
	}
}
`

var PayloadQueryFloat64ValidateDecodeCode = `// DecodeMethodQueryFloat64ValidateRequest returns a decoder for requests sent
// to the ServiceQueryFloat64Validate MethodQueryFloat64Validate endpoint.
func DecodeMethodQueryFloat64ValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   float64
			err error
		)
		{
			qRaw := r.URL.Query().Get("q")
			if qRaw == "" {
				err = loom.MergeErrors(err, loom.MissingFieldError("q", "query string"))
			}
			v, err2 := strconv.ParseFloat(qRaw, 64)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "float"))
			}
			q = v
		}
		if q < 1 {
			err = loom.MergeErrors(err, loom.InvalidRangeError("q", q, 1, true))
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryFloat64ValidatePayload(q)

		return payload, nil
	}
}
`

var PayloadQueryStringDecodeCode = `// DecodeMethodQueryStringRequest returns a decoder for requests sent to the
// ServiceQueryString MethodQueryString endpoint.
func DecodeMethodQueryStringRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q *string
		)
		qRaw := r.URL.Query().Get("q")
		if qRaw != "" {
			q = &qRaw
		}
		payload := NewMethodQueryStringPayload(q)

		return payload, nil
	}
}
`

var PayloadQueryStringValidateDecodeCode = `// DecodeMethodQueryStringValidateRequest returns a decoder for requests sent
// to the ServiceQueryStringValidate MethodQueryStringValidate endpoint.
func DecodeMethodQueryStringValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   string
			err error
		)
		q = r.URL.Query().Get("q")
		if q == "" {
			err = loom.MergeErrors(err, loom.MissingFieldError("q", "query string"))
		}
		if !(q == "val") {
			err = loom.MergeErrors(err, loom.InvalidEnumValueError("q", q, []any{"val"}))
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryStringValidatePayload(q)

		return payload, nil
	}
}
`

var PayloadQueryStringNotRequiredValidateDecodeCode = `// DecodeMethodQueryStringNotRequiredValidateRequest returns a decoder for
// requests sent to the ServiceQueryStringNotRequiredValidate
// MethodQueryStringNotRequiredValidate endpoint.
func DecodeMethodQueryStringNotRequiredValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   *string
			err error
		)
		qRaw := r.URL.Query().Get("q")
		if qRaw != "" {
			q = &qRaw
		}
		if q != nil {
			if !(*q == "val") {
				err = loom.MergeErrors(err, loom.InvalidEnumValueError("q", *q, []any{"val"}))
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryStringNotRequiredValidatePayload(q)

		return payload, nil
	}
}
`

var PayloadQueryBytesDecodeCode = `// DecodeMethodQueryBytesRequest returns a decoder for requests sent to the
// ServiceQueryBytes MethodQueryBytes endpoint.
func DecodeMethodQueryBytesRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q []byte
		)
		{
			qRaw := r.URL.Query().Get("q")
			if qRaw != "" {
				q = []byte(qRaw)
			}
		}
		payload := NewMethodQueryBytesPayload(q)

		return payload, nil
	}
}
`

var PayloadQueryBytesValidateDecodeCode = `// DecodeMethodQueryBytesValidateRequest returns a decoder for requests sent to
// the ServiceQueryBytesValidate MethodQueryBytesValidate endpoint.
func DecodeMethodQueryBytesValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   []byte
			err error
		)
		{
			qRaw := r.URL.Query().Get("q")
			if qRaw == "" {
				err = loom.MergeErrors(err, loom.MissingFieldError("q", "query string"))
			}
			q = []byte(qRaw)
		}
		if len(q) < 1 {
			err = loom.MergeErrors(err, loom.InvalidLengthError("q", q, len(q), 1, true))
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryBytesValidatePayload(q)

		return payload, nil
	}
}
`

var PayloadQueryAnyDecodeCode = `// DecodeMethodQueryAnyRequest returns a decoder for requests sent to the
// ServiceQueryAny MethodQueryAny endpoint.
func DecodeMethodQueryAnyRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q any
		)
		qRaw := r.URL.Query().Get("q")
		if qRaw != "" {
			q = qRaw
		}
		payload := NewMethodQueryAnyPayload(q)

		return payload, nil
	}
}
`

var PayloadQueryAnyValidateDecodeCode = `// DecodeMethodQueryAnyValidateRequest returns a decoder for requests sent to
// the ServiceQueryAnyValidate MethodQueryAnyValidate endpoint.
func DecodeMethodQueryAnyValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   any
			err error
		)
		q = r.URL.Query().Get("q")
		if q == "" {
			err = loom.MergeErrors(err, loom.MissingFieldError("q", "query string"))
		}
		if !(q == "val" || q == 1) {
			err = loom.MergeErrors(err, loom.InvalidEnumValueError("q", q, []any{"val", 1}))
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryAnyValidatePayload(q)

		return payload, nil
	}
}
`

var PayloadQueryArrayBoolDecodeCode = `// DecodeMethodQueryArrayBoolRequest returns a decoder for requests sent to the
// ServiceQueryArrayBool MethodQueryArrayBool endpoint.
func DecodeMethodQueryArrayBoolRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   []bool
			err error
		)
		{
			qRaw := r.URL.Query()["q"]
			if qRaw != nil {
				q = make([]bool, len(qRaw))
				for i, rv := range qRaw {
					v, err2 := strconv.ParseBool(rv)
					if err2 != nil {
						err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "array of booleans"))
					}
					q[i] = v
				}
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryArrayBoolPayload(q)

		return payload, nil
	}
}
`

var PayloadQueryArrayBoolValidateDecodeCode = `// DecodeMethodQueryArrayBoolValidateRequest returns a decoder for requests
// sent to the ServiceQueryArrayBoolValidate MethodQueryArrayBoolValidate
// endpoint.
func DecodeMethodQueryArrayBoolValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   []bool
			err error
		)
		{
			qRaw := r.URL.Query()["q"]
			if qRaw == nil {
				err = loom.MergeErrors(err, loom.MissingFieldError("q", "query string"))
			}
			q = make([]bool, len(qRaw))
			for i, rv := range qRaw {
				v, err2 := strconv.ParseBool(rv)
				if err2 != nil {
					err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "array of booleans"))
				}
				q[i] = v
			}
		}
		if len(q) < 1 {
			err = loom.MergeErrors(err, loom.InvalidLengthError("q", q, len(q), 1, true))
		}
		for _, e := range q {
			if !(e == true) {
				err = loom.MergeErrors(err, loom.InvalidEnumValueError("q[*]", e, []any{true}))
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryArrayBoolValidatePayload(q)

		return payload, nil
	}
}
`

var PayloadQueryArrayIntDecodeCode = `// DecodeMethodQueryArrayIntRequest returns a decoder for requests sent to the
// ServiceQueryArrayInt MethodQueryArrayInt endpoint.
func DecodeMethodQueryArrayIntRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   []int
			err error
		)
		{
			qRaw := r.URL.Query()["q"]
			if qRaw != nil {
				q = make([]int, len(qRaw))
				for i, rv := range qRaw {
					v, err2 := strconv.ParseInt(rv, 10, strconv.IntSize)
					if err2 != nil {
						err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "array of integers"))
					}
					q[i] = int(v)
				}
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryArrayIntPayload(q)

		return payload, nil
	}
}
`

var PayloadQueryArrayIntValidateDecodeCode = `// DecodeMethodQueryArrayIntValidateRequest returns a decoder for requests sent
// to the ServiceQueryArrayIntValidate MethodQueryArrayIntValidate endpoint.
func DecodeMethodQueryArrayIntValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   []int
			err error
		)
		{
			qRaw := r.URL.Query()["q"]
			if qRaw == nil {
				err = loom.MergeErrors(err, loom.MissingFieldError("q", "query string"))
			}
			q = make([]int, len(qRaw))
			for i, rv := range qRaw {
				v, err2 := strconv.ParseInt(rv, 10, strconv.IntSize)
				if err2 != nil {
					err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "array of integers"))
				}
				q[i] = int(v)
			}
		}
		if len(q) < 1 {
			err = loom.MergeErrors(err, loom.InvalidLengthError("q", q, len(q), 1, true))
		}
		for _, e := range q {
			if e < 1 {
				err = loom.MergeErrors(err, loom.InvalidRangeError("q[*]", e, 1, true))
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryArrayIntValidatePayload(q)

		return payload, nil
	}
}
`

var PayloadQueryArrayInt32DecodeCode = `// DecodeMethodQueryArrayInt32Request returns a decoder for requests sent to
// the ServiceQueryArrayInt32 MethodQueryArrayInt32 endpoint.
func DecodeMethodQueryArrayInt32Request(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   []int32
			err error
		)
		{
			qRaw := r.URL.Query()["q"]
			if qRaw != nil {
				q = make([]int32, len(qRaw))
				for i, rv := range qRaw {
					v, err2 := strconv.ParseInt(rv, 10, 32)
					if err2 != nil {
						err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "array of integers"))
					}
					q[i] = int32(v)
				}
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryArrayInt32Payload(q)

		return payload, nil
	}
}
`

var PayloadQueryArrayInt32ValidateDecodeCode = `// DecodeMethodQueryArrayInt32ValidateRequest returns a decoder for requests
// sent to the ServiceQueryArrayInt32Validate MethodQueryArrayInt32Validate
// endpoint.
func DecodeMethodQueryArrayInt32ValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   []int32
			err error
		)
		{
			qRaw := r.URL.Query()["q"]
			if qRaw == nil {
				err = loom.MergeErrors(err, loom.MissingFieldError("q", "query string"))
			}
			q = make([]int32, len(qRaw))
			for i, rv := range qRaw {
				v, err2 := strconv.ParseInt(rv, 10, 32)
				if err2 != nil {
					err = loom.MergeErrors(err, loom.InvalidFieldTypeError("q", qRaw, "array of integers"))
				}
				q[i] = int32(v)
			}
		}
		if len(q) < 1 {
			err = loom.MergeErrors(err, loom.InvalidLengthError("q", q, len(q), 1, true))
		}
		for _, e := range q {
			if e < 1 {
				err = loom.MergeErrors(err, loom.InvalidRangeError("q[*]", e, 1, true))
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryArrayInt32ValidatePayload(q)

		return payload, nil
	}
}
`


