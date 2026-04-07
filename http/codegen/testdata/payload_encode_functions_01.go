package testdata


var PayloadQueryBoolEncodeCode = `// EncodeMethodQueryBoolRequest returns an encoder for requests sent to the
// ServiceQueryBool MethodQueryBool server.
func EncodeMethodQueryBoolRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequerybool.MethodQueryBoolPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryBool", "MethodQueryBool", "*servicequerybool.MethodQueryBoolPayload", v)
		}
		values := req.URL.Query()
		if p.Q != nil {
			values.Add("q", fmt.Sprintf("%v", *p.Q))
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryBoolValidateEncodeCode = `// EncodeMethodQueryBoolValidateRequest returns an encoder for requests sent to
// the ServiceQueryBoolValidate MethodQueryBoolValidate server.
func EncodeMethodQueryBoolValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryboolvalidate.MethodQueryBoolValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryBoolValidate", "MethodQueryBoolValidate", "*servicequeryboolvalidate.MethodQueryBoolValidatePayload", v)
		}
		values := req.URL.Query()
		values.Add("q", fmt.Sprintf("%v", p.Q))
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryIntEncodeCode = `// EncodeMethodQueryIntRequest returns an encoder for requests sent to the
// ServiceQueryInt MethodQueryInt server.
func EncodeMethodQueryIntRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryint.MethodQueryIntPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryInt", "MethodQueryInt", "*servicequeryint.MethodQueryIntPayload", v)
		}
		values := req.URL.Query()
		if p.Q != nil {
			values.Add("q", fmt.Sprintf("%v", *p.Q))
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryIntValidateEncodeCode = `// EncodeMethodQueryIntValidateRequest returns an encoder for requests sent to
// the ServiceQueryIntValidate MethodQueryIntValidate server.
func EncodeMethodQueryIntValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryintvalidate.MethodQueryIntValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryIntValidate", "MethodQueryIntValidate", "*servicequeryintvalidate.MethodQueryIntValidatePayload", v)
		}
		values := req.URL.Query()
		values.Add("q", fmt.Sprintf("%v", p.Q))
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryInt32EncodeCode = `// EncodeMethodQueryInt32Request returns an encoder for requests sent to the
// ServiceQueryInt32 MethodQueryInt32 server.
func EncodeMethodQueryInt32Request(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryint32.MethodQueryInt32Payload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryInt32", "MethodQueryInt32", "*servicequeryint32.MethodQueryInt32Payload", v)
		}
		values := req.URL.Query()
		if p.Q != nil {
			values.Add("q", fmt.Sprintf("%v", *p.Q))
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryInt32ValidateEncodeCode = `// EncodeMethodQueryInt32ValidateRequest returns an encoder for requests sent
// to the ServiceQueryInt32Validate MethodQueryInt32Validate server.
func EncodeMethodQueryInt32ValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryint32validate.MethodQueryInt32ValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryInt32Validate", "MethodQueryInt32Validate", "*servicequeryint32validate.MethodQueryInt32ValidatePayload", v)
		}
		values := req.URL.Query()
		values.Add("q", fmt.Sprintf("%v", p.Q))
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryInt64EncodeCode = `// EncodeMethodQueryInt64Request returns an encoder for requests sent to the
// ServiceQueryInt64 MethodQueryInt64 server.
func EncodeMethodQueryInt64Request(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryint64.MethodQueryInt64Payload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryInt64", "MethodQueryInt64", "*servicequeryint64.MethodQueryInt64Payload", v)
		}
		values := req.URL.Query()
		if p.Q != nil {
			values.Add("q", fmt.Sprintf("%v", *p.Q))
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryInt64ValidateEncodeCode = `// EncodeMethodQueryInt64ValidateRequest returns an encoder for requests sent
// to the ServiceQueryInt64Validate MethodQueryInt64Validate server.
func EncodeMethodQueryInt64ValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryint64validate.MethodQueryInt64ValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryInt64Validate", "MethodQueryInt64Validate", "*servicequeryint64validate.MethodQueryInt64ValidatePayload", v)
		}
		values := req.URL.Query()
		values.Add("q", fmt.Sprintf("%v", p.Q))
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryUIntEncodeCode = `// EncodeMethodQueryUIntRequest returns an encoder for requests sent to the
// ServiceQueryUInt MethodQueryUInt server.
func EncodeMethodQueryUIntRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryuint.MethodQueryUIntPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryUInt", "MethodQueryUInt", "*servicequeryuint.MethodQueryUIntPayload", v)
		}
		values := req.URL.Query()
		if p.Q != nil {
			values.Add("q", fmt.Sprintf("%v", *p.Q))
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryUIntValidateEncodeCode = `// EncodeMethodQueryUIntValidateRequest returns an encoder for requests sent to
// the ServiceQueryUIntValidate MethodQueryUIntValidate server.
func EncodeMethodQueryUIntValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryuintvalidate.MethodQueryUIntValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryUIntValidate", "MethodQueryUIntValidate", "*servicequeryuintvalidate.MethodQueryUIntValidatePayload", v)
		}
		values := req.URL.Query()
		values.Add("q", fmt.Sprintf("%v", p.Q))
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryUInt32EncodeCode = `// EncodeMethodQueryUInt32Request returns an encoder for requests sent to the
// ServiceQueryUInt32 MethodQueryUInt32 server.
func EncodeMethodQueryUInt32Request(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryuint32.MethodQueryUInt32Payload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryUInt32", "MethodQueryUInt32", "*servicequeryuint32.MethodQueryUInt32Payload", v)
		}
		values := req.URL.Query()
		if p.Q != nil {
			values.Add("q", fmt.Sprintf("%v", *p.Q))
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryUInt32ValidateEncodeCode = `// EncodeMethodQueryUInt32ValidateRequest returns an encoder for requests sent
// to the ServiceQueryUInt32Validate MethodQueryUInt32Validate server.
func EncodeMethodQueryUInt32ValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryuint32validate.MethodQueryUInt32ValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryUInt32Validate", "MethodQueryUInt32Validate", "*servicequeryuint32validate.MethodQueryUInt32ValidatePayload", v)
		}
		values := req.URL.Query()
		values.Add("q", fmt.Sprintf("%v", p.Q))
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryUInt64EncodeCode = `// EncodeMethodQueryUInt64Request returns an encoder for requests sent to the
// ServiceQueryUInt64 MethodQueryUInt64 server.
func EncodeMethodQueryUInt64Request(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryuint64.MethodQueryUInt64Payload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryUInt64", "MethodQueryUInt64", "*servicequeryuint64.MethodQueryUInt64Payload", v)
		}
		values := req.URL.Query()
		if p.Q != nil {
			values.Add("q", fmt.Sprintf("%v", *p.Q))
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryUInt64ValidateEncodeCode = `// EncodeMethodQueryUInt64ValidateRequest returns an encoder for requests sent
// to the ServiceQueryUInt64Validate MethodQueryUInt64Validate server.
func EncodeMethodQueryUInt64ValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryuint64validate.MethodQueryUInt64ValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryUInt64Validate", "MethodQueryUInt64Validate", "*servicequeryuint64validate.MethodQueryUInt64ValidatePayload", v)
		}
		values := req.URL.Query()
		values.Add("q", fmt.Sprintf("%v", p.Q))
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryFloat32EncodeCode = `// EncodeMethodQueryFloat32Request returns an encoder for requests sent to the
// ServiceQueryFloat32 MethodQueryFloat32 server.
func EncodeMethodQueryFloat32Request(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryfloat32.MethodQueryFloat32Payload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryFloat32", "MethodQueryFloat32", "*servicequeryfloat32.MethodQueryFloat32Payload", v)
		}
		values := req.URL.Query()
		if p.Q != nil {
			values.Add("q", fmt.Sprintf("%v", *p.Q))
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryFloat32ValidateEncodeCode = `// EncodeMethodQueryFloat32ValidateRequest returns an encoder for requests sent
// to the ServiceQueryFloat32Validate MethodQueryFloat32Validate server.
func EncodeMethodQueryFloat32ValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryfloat32validate.MethodQueryFloat32ValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryFloat32Validate", "MethodQueryFloat32Validate", "*servicequeryfloat32validate.MethodQueryFloat32ValidatePayload", v)
		}
		values := req.URL.Query()
		values.Add("q", fmt.Sprintf("%v", p.Q))
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryFloat64EncodeCode = `// EncodeMethodQueryFloat64Request returns an encoder for requests sent to the
// ServiceQueryFloat64 MethodQueryFloat64 server.
func EncodeMethodQueryFloat64Request(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryfloat64.MethodQueryFloat64Payload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryFloat64", "MethodQueryFloat64", "*servicequeryfloat64.MethodQueryFloat64Payload", v)
		}
		values := req.URL.Query()
		if p.Q != nil {
			values.Add("q", fmt.Sprintf("%v", *p.Q))
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryFloat64ValidateEncodeCode = `// EncodeMethodQueryFloat64ValidateRequest returns an encoder for requests sent
// to the ServiceQueryFloat64Validate MethodQueryFloat64Validate server.
func EncodeMethodQueryFloat64ValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryfloat64validate.MethodQueryFloat64ValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryFloat64Validate", "MethodQueryFloat64Validate", "*servicequeryfloat64validate.MethodQueryFloat64ValidatePayload", v)
		}
		values := req.URL.Query()
		values.Add("q", fmt.Sprintf("%v", p.Q))
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryStringEncodeCode = `// EncodeMethodQueryStringRequest returns an encoder for requests sent to the
// ServiceQueryString MethodQueryString server.
func EncodeMethodQueryStringRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequerystring.MethodQueryStringPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryString", "MethodQueryString", "*servicequerystring.MethodQueryStringPayload", v)
		}
		values := req.URL.Query()
		if p.Q != nil {
			values.Add("q", *p.Q)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryStringValidateEncodeCode = `// EncodeMethodQueryStringValidateRequest returns an encoder for requests sent
// to the ServiceQueryStringValidate MethodQueryStringValidate server.
func EncodeMethodQueryStringValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequerystringvalidate.MethodQueryStringValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryStringValidate", "MethodQueryStringValidate", "*servicequerystringvalidate.MethodQueryStringValidatePayload", v)
		}
		values := req.URL.Query()
		values.Add("q", p.Q)
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryBytesEncodeCode = `// EncodeMethodQueryBytesRequest returns an encoder for requests sent to the
// ServiceQueryBytes MethodQueryBytes server.
func EncodeMethodQueryBytesRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequerybytes.MethodQueryBytesPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryBytes", "MethodQueryBytes", "*servicequerybytes.MethodQueryBytesPayload", v)
		}
		values := req.URL.Query()
		values.Add("q", string(p.Q))
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryBytesValidateEncodeCode = `// EncodeMethodQueryBytesValidateRequest returns an encoder for requests sent
// to the ServiceQueryBytesValidate MethodQueryBytesValidate server.
func EncodeMethodQueryBytesValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequerybytesvalidate.MethodQueryBytesValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryBytesValidate", "MethodQueryBytesValidate", "*servicequerybytesvalidate.MethodQueryBytesValidatePayload", v)
		}
		values := req.URL.Query()
		values.Add("q", string(p.Q))
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryAnyEncodeCode = `// EncodeMethodQueryAnyRequest returns an encoder for requests sent to the
// ServiceQueryAny MethodQueryAny server.
func EncodeMethodQueryAnyRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryany.MethodQueryAnyPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryAny", "MethodQueryAny", "*servicequeryany.MethodQueryAnyPayload", v)
		}
		values := req.URL.Query()
		values.Add("q", fmt.Sprintf("%v", p.Q))
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryAnyValidateEncodeCode = `// EncodeMethodQueryAnyValidateRequest returns an encoder for requests sent to
// the ServiceQueryAnyValidate MethodQueryAnyValidate server.
func EncodeMethodQueryAnyValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryanyvalidate.MethodQueryAnyValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryAnyValidate", "MethodQueryAnyValidate", "*servicequeryanyvalidate.MethodQueryAnyValidatePayload", v)
		}
		values := req.URL.Query()
		values.Add("q", fmt.Sprintf("%v", p.Q))
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayBoolEncodeCode = `// EncodeMethodQueryArrayBoolRequest returns an encoder for requests sent to
// the ServiceQueryArrayBool MethodQueryArrayBool server.
func EncodeMethodQueryArrayBoolRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarraybool.MethodQueryArrayBoolPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayBool", "MethodQueryArrayBool", "*servicequeryarraybool.MethodQueryArrayBoolPayload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			valueStr := strconv.FormatBool(value)
			values.Add("q", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayBoolValidateEncodeCode = `// EncodeMethodQueryArrayBoolValidateRequest returns an encoder for requests
// sent to the ServiceQueryArrayBoolValidate MethodQueryArrayBoolValidate
// server.
func EncodeMethodQueryArrayBoolValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarrayboolvalidate.MethodQueryArrayBoolValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayBoolValidate", "MethodQueryArrayBoolValidate", "*servicequeryarrayboolvalidate.MethodQueryArrayBoolValidatePayload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			valueStr := strconv.FormatBool(value)
			values.Add("q", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayIntEncodeCode = `// EncodeMethodQueryArrayIntRequest returns an encoder for requests sent to the
// ServiceQueryArrayInt MethodQueryArrayInt server.
func EncodeMethodQueryArrayIntRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarrayint.MethodQueryArrayIntPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayInt", "MethodQueryArrayInt", "*servicequeryarrayint.MethodQueryArrayIntPayload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			valueStr := strconv.Itoa(value)
			values.Add("q", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayIntValidateEncodeCode = `// EncodeMethodQueryArrayIntValidateRequest returns an encoder for requests
// sent to the ServiceQueryArrayIntValidate MethodQueryArrayIntValidate server.
func EncodeMethodQueryArrayIntValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarrayintvalidate.MethodQueryArrayIntValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayIntValidate", "MethodQueryArrayIntValidate", "*servicequeryarrayintvalidate.MethodQueryArrayIntValidatePayload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			valueStr := strconv.Itoa(value)
			values.Add("q", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayInt32EncodeCode = `// EncodeMethodQueryArrayInt32Request returns an encoder for requests sent to
// the ServiceQueryArrayInt32 MethodQueryArrayInt32 server.
func EncodeMethodQueryArrayInt32Request(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarrayint32.MethodQueryArrayInt32Payload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayInt32", "MethodQueryArrayInt32", "*servicequeryarrayint32.MethodQueryArrayInt32Payload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			valueStr := strconv.FormatInt(int64(value), 10)
			values.Add("q", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayInt32ValidateEncodeCode = `// EncodeMethodQueryArrayInt32ValidateRequest returns an encoder for requests
// sent to the ServiceQueryArrayInt32Validate MethodQueryArrayInt32Validate
// server.
func EncodeMethodQueryArrayInt32ValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarrayint32validate.MethodQueryArrayInt32ValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayInt32Validate", "MethodQueryArrayInt32Validate", "*servicequeryarrayint32validate.MethodQueryArrayInt32ValidatePayload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			valueStr := strconv.FormatInt(int64(value), 10)
			values.Add("q", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayInt64EncodeCode = `// EncodeMethodQueryArrayInt64Request returns an encoder for requests sent to
// the ServiceQueryArrayInt64 MethodQueryArrayInt64 server.
func EncodeMethodQueryArrayInt64Request(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarrayint64.MethodQueryArrayInt64Payload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayInt64", "MethodQueryArrayInt64", "*servicequeryarrayint64.MethodQueryArrayInt64Payload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			valueStr := strconv.FormatInt(value, 10)
			values.Add("q", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayInt64ValidateEncodeCode = `// EncodeMethodQueryArrayInt64ValidateRequest returns an encoder for requests
// sent to the ServiceQueryArrayInt64Validate MethodQueryArrayInt64Validate
// server.
func EncodeMethodQueryArrayInt64ValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarrayint64validate.MethodQueryArrayInt64ValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayInt64Validate", "MethodQueryArrayInt64Validate", "*servicequeryarrayint64validate.MethodQueryArrayInt64ValidatePayload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			valueStr := strconv.FormatInt(value, 10)
			values.Add("q", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayUIntEncodeCode = `// EncodeMethodQueryArrayUIntRequest returns an encoder for requests sent to
// the ServiceQueryArrayUInt MethodQueryArrayUInt server.
func EncodeMethodQueryArrayUIntRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarrayuint.MethodQueryArrayUIntPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayUInt", "MethodQueryArrayUInt", "*servicequeryarrayuint.MethodQueryArrayUIntPayload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			valueStr := strconv.FormatUint(uint64(value), 10)
			values.Add("q", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayUIntValidateEncodeCode = `// EncodeMethodQueryArrayUIntValidateRequest returns an encoder for requests
// sent to the ServiceQueryArrayUIntValidate MethodQueryArrayUIntValidate
// server.
func EncodeMethodQueryArrayUIntValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarrayuintvalidate.MethodQueryArrayUIntValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayUIntValidate", "MethodQueryArrayUIntValidate", "*servicequeryarrayuintvalidate.MethodQueryArrayUIntValidatePayload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			valueStr := strconv.FormatUint(uint64(value), 10)
			values.Add("q", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayUInt32EncodeCode = `// EncodeMethodQueryArrayUInt32Request returns an encoder for requests sent to
// the ServiceQueryArrayUInt32 MethodQueryArrayUInt32 server.
func EncodeMethodQueryArrayUInt32Request(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarrayuint32.MethodQueryArrayUInt32Payload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayUInt32", "MethodQueryArrayUInt32", "*servicequeryarrayuint32.MethodQueryArrayUInt32Payload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			valueStr := strconv.FormatUint(uint64(value), 10)
			values.Add("q", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayUInt32ValidateEncodeCode = `// EncodeMethodQueryArrayUInt32ValidateRequest returns an encoder for requests
// sent to the ServiceQueryArrayUInt32Validate MethodQueryArrayUInt32Validate
// server.
func EncodeMethodQueryArrayUInt32ValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarrayuint32validate.MethodQueryArrayUInt32ValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayUInt32Validate", "MethodQueryArrayUInt32Validate", "*servicequeryarrayuint32validate.MethodQueryArrayUInt32ValidatePayload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			valueStr := strconv.FormatUint(uint64(value), 10)
			values.Add("q", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayUInt64EncodeCode = `// EncodeMethodQueryArrayUInt64Request returns an encoder for requests sent to
// the ServiceQueryArrayUInt64 MethodQueryArrayUInt64 server.
func EncodeMethodQueryArrayUInt64Request(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarrayuint64.MethodQueryArrayUInt64Payload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayUInt64", "MethodQueryArrayUInt64", "*servicequeryarrayuint64.MethodQueryArrayUInt64Payload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			valueStr := strconv.FormatUint(value, 10)
			values.Add("q", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayUInt64ValidateEncodeCode = `// EncodeMethodQueryArrayUInt64ValidateRequest returns an encoder for requests
// sent to the ServiceQueryArrayUInt64Validate MethodQueryArrayUInt64Validate
// server.
func EncodeMethodQueryArrayUInt64ValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarrayuint64validate.MethodQueryArrayUInt64ValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayUInt64Validate", "MethodQueryArrayUInt64Validate", "*servicequeryarrayuint64validate.MethodQueryArrayUInt64ValidatePayload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			valueStr := strconv.FormatUint(value, 10)
			values.Add("q", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayFloat32EncodeCode = `// EncodeMethodQueryArrayFloat32Request returns an encoder for requests sent to
// the ServiceQueryArrayFloat32 MethodQueryArrayFloat32 server.
func EncodeMethodQueryArrayFloat32Request(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarrayfloat32.MethodQueryArrayFloat32Payload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayFloat32", "MethodQueryArrayFloat32", "*servicequeryarrayfloat32.MethodQueryArrayFloat32Payload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			valueStr := strconv.FormatFloat(float64(value), 'f', -1, 32)
			values.Add("q", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayFloat32ValidateEncodeCode = `// EncodeMethodQueryArrayFloat32ValidateRequest returns an encoder for requests
// sent to the ServiceQueryArrayFloat32Validate MethodQueryArrayFloat32Validate
// server.
func EncodeMethodQueryArrayFloat32ValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarrayfloat32validate.MethodQueryArrayFloat32ValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayFloat32Validate", "MethodQueryArrayFloat32Validate", "*servicequeryarrayfloat32validate.MethodQueryArrayFloat32ValidatePayload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			valueStr := strconv.FormatFloat(float64(value), 'f', -1, 32)
			values.Add("q", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayFloat64EncodeCode = `// EncodeMethodQueryArrayFloat64Request returns an encoder for requests sent to
// the ServiceQueryArrayFloat64 MethodQueryArrayFloat64 server.
func EncodeMethodQueryArrayFloat64Request(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarrayfloat64.MethodQueryArrayFloat64Payload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayFloat64", "MethodQueryArrayFloat64", "*servicequeryarrayfloat64.MethodQueryArrayFloat64Payload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			valueStr := strconv.FormatFloat(value, 'f', -1, 64)
			values.Add("q", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayFloat64ValidateEncodeCode = `// EncodeMethodQueryArrayFloat64ValidateRequest returns an encoder for requests
// sent to the ServiceQueryArrayFloat64Validate MethodQueryArrayFloat64Validate
// server.
func EncodeMethodQueryArrayFloat64ValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarrayfloat64validate.MethodQueryArrayFloat64ValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayFloat64Validate", "MethodQueryArrayFloat64Validate", "*servicequeryarrayfloat64validate.MethodQueryArrayFloat64ValidatePayload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			valueStr := strconv.FormatFloat(value, 'f', -1, 64)
			values.Add("q", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayStringEncodeCode = `// EncodeMethodQueryArrayStringRequest returns an encoder for requests sent to
// the ServiceQueryArrayString MethodQueryArrayString server.
func EncodeMethodQueryArrayStringRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarraystring.MethodQueryArrayStringPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayString", "MethodQueryArrayString", "*servicequeryarraystring.MethodQueryArrayStringPayload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			values.Add("q", value)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayStringValidateEncodeCode = `// EncodeMethodQueryArrayStringValidateRequest returns an encoder for requests
// sent to the ServiceQueryArrayStringValidate MethodQueryArrayStringValidate
// server.
func EncodeMethodQueryArrayStringValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarraystringvalidate.MethodQueryArrayStringValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayStringValidate", "MethodQueryArrayStringValidate", "*servicequeryarraystringvalidate.MethodQueryArrayStringValidatePayload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			values.Add("q", value)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayBytesEncodeCode = `// EncodeMethodQueryArrayBytesRequest returns an encoder for requests sent to
// the ServiceQueryArrayBytes MethodQueryArrayBytes server.
func EncodeMethodQueryArrayBytesRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarraybytes.MethodQueryArrayBytesPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayBytes", "MethodQueryArrayBytes", "*servicequeryarraybytes.MethodQueryArrayBytesPayload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			valueStr := string(value)
			values.Add("q", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayBytesValidateEncodeCode = `// EncodeMethodQueryArrayBytesValidateRequest returns an encoder for requests
// sent to the ServiceQueryArrayBytesValidate MethodQueryArrayBytesValidate
// server.
func EncodeMethodQueryArrayBytesValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarraybytesvalidate.MethodQueryArrayBytesValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayBytesValidate", "MethodQueryArrayBytesValidate", "*servicequeryarraybytesvalidate.MethodQueryArrayBytesValidatePayload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			valueStr := string(value)
			values.Add("q", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayAnyEncodeCode = `// EncodeMethodQueryArrayAnyRequest returns an encoder for requests sent to the
// ServiceQueryArrayAny MethodQueryArrayAny server.
func EncodeMethodQueryArrayAnyRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarrayany.MethodQueryArrayAnyPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayAny", "MethodQueryArrayAny", "*servicequeryarrayany.MethodQueryArrayAnyPayload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			valueStr := fmt.Sprintf("%v", value)
			values.Add("q", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayAnyValidateEncodeCode = `// EncodeMethodQueryArrayAnyValidateRequest returns an encoder for requests
// sent to the ServiceQueryArrayAnyValidate MethodQueryArrayAnyValidate server.
func EncodeMethodQueryArrayAnyValidateRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarrayanyvalidate.MethodQueryArrayAnyValidatePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayAnyValidate", "MethodQueryArrayAnyValidate", "*servicequeryarrayanyvalidate.MethodQueryArrayAnyValidatePayload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			valueStr := fmt.Sprintf("%v", value)
			values.Add("q", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryArrayAliasEncodeCode = `// EncodeMethodQueryArrayAliasRequest returns an encoder for requests sent to
// the ServiceQueryArrayAlias MethodQueryArrayAlias server.
func EncodeMethodQueryArrayAliasRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequeryarrayalias.MethodQueryArrayAliasPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryArrayAlias", "MethodQueryArrayAlias", "*servicequeryarrayalias.MethodQueryArrayAliasPayload", v)
		}
		values := req.URL.Query()
		for _, value := range p.Q {
			valueStr := string(value)
			values.Add("q", valueStr)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`

var PayloadQueryMapStringStringEncodeCode = `// EncodeMethodQueryMapStringStringRequest returns an encoder for requests sent
// to the ServiceQueryMapStringString MethodQueryMapStringString server.
func EncodeMethodQueryMapStringStringRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*servicequerymapstringstring.MethodQueryMapStringStringPayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceQueryMapStringString", "MethodQueryMapStringString", "*servicequerymapstringstring.MethodQueryMapStringStringPayload", v)
		}
		values := req.URL.Query()
		for k, value := range p.Q {
			key := fmt.Sprintf("q[%s]", k)
			values.Add(key, value)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}
`


