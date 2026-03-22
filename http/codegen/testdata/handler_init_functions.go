package testdata

var ServerNoPayloadNoResultHandlerConstructorCode = `// NewMethodNoPayloadNoResultHandler creates a HTTP handler which loads the
// HTTP request and calls the "ServiceNoPayloadNoResult" service
// "MethodNoPayloadNoResult" endpoint.
func NewMethodNoPayloadNoResultHandler(
	endpoint loom.Endpoint,
	mux loomhttp.Muxer,
	decoder func(*http.Request) loomhttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) loomhttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(ctx context.Context, err error) loomhttp.Statuser,
) http.Handler {
	var (
		encodeResponse = EncodeMethodNoPayloadNoResultResponse(encoder)
		encodeError    = loomhttp.ErrorEncoder(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), loomhttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, loom.MethodKey, "MethodNoPayloadNoResult")
		ctx = context.WithValue(ctx, loom.ServiceKey, "ServiceNoPayloadNoResult")
		var err error
		res, err := endpoint(ctx, nil)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		if err := encodeResponse(ctx, w, res); err != nil {
			if errhandler != nil {
				errhandler(ctx, w, err)
			}
		}
	})
}
`

var ServerNoPayloadNoResultWithRedirectHandlerConstructorCode = `// NewMethodNoPayloadNoResultHandler creates a HTTP handler which loads the
// HTTP request and calls the "ServiceNoPayloadNoResult" service
// "MethodNoPayloadNoResult" endpoint.
func NewMethodNoPayloadNoResultHandler(
	endpoint loom.Endpoint,
	mux loomhttp.Muxer,
	decoder func(*http.Request) loomhttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) loomhttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(ctx context.Context, err error) loomhttp.Statuser,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), loomhttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, loom.MethodKey, "MethodNoPayloadNoResult")
		ctx = context.WithValue(ctx, loom.ServiceKey, "ServiceNoPayloadNoResult")
		http.Redirect(w, r, "/redirect/dest", http.StatusMovedPermanently)
	})
}
`

var ServerPayloadNoResultHandlerConstructorCode = `// NewMethodPayloadNoResultHandler creates a HTTP handler which loads the HTTP
// request and calls the "ServicePayloadNoResult" service
// "MethodPayloadNoResult" endpoint.
func NewMethodPayloadNoResultHandler(
	endpoint loom.Endpoint,
	mux loomhttp.Muxer,
	decoder func(*http.Request) loomhttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) loomhttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(ctx context.Context, err error) loomhttp.Statuser,
) http.Handler {
	var (
		decodeRequest  = DecodeMethodPayloadNoResultRequest(mux, decoder)
		encodeResponse = EncodeMethodPayloadNoResultResponse(encoder)
		encodeError    = loomhttp.ErrorEncoder(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), loomhttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, loom.MethodKey, "MethodPayloadNoResult")
		ctx = context.WithValue(ctx, loom.ServiceKey, "ServicePayloadNoResult")
		payload, err := decodeRequest(r)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		res, err := endpoint(ctx, payload)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		if err := encodeResponse(ctx, w, res); err != nil {
			if errhandler != nil {
				errhandler(ctx, w, err)
			}
		}
	})
}
`

var ServerPayloadNoResultWithRedirectHandlerConstructorCode = `// NewMethodPayloadNoResultHandler creates a HTTP handler which loads the HTTP
// request and calls the "ServicePayloadNoResult" service
// "MethodPayloadNoResult" endpoint.
func NewMethodPayloadNoResultHandler(
	endpoint loom.Endpoint,
	mux loomhttp.Muxer,
	decoder func(*http.Request) loomhttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) loomhttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(ctx context.Context, err error) loomhttp.Statuser,
) http.Handler {
	var (
		decodeRequest = DecodeMethodPayloadNoResultRequest(mux, decoder)
		encodeError   = loomhttp.ErrorEncoder(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), loomhttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, loom.MethodKey, "MethodPayloadNoResult")
		ctx = context.WithValue(ctx, loom.ServiceKey, "ServicePayloadNoResult")
		_, err := decodeRequest(r)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		http.Redirect(w, r, "/redirect/dest", http.StatusMovedPermanently)
	})
}
`

var ServerNoPayloadResultHandlerConstructorCode = `// NewMethodNoPayloadResultHandler creates a HTTP handler which loads the HTTP
// request and calls the "ServiceNoPayloadResult" service
// "MethodNoPayloadResult" endpoint.
func NewMethodNoPayloadResultHandler(
	endpoint loom.Endpoint,
	mux loomhttp.Muxer,
	decoder func(*http.Request) loomhttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) loomhttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(ctx context.Context, err error) loomhttp.Statuser,
) http.Handler {
	var (
		encodeResponse = EncodeMethodNoPayloadResultResponse(encoder)
		encodeError    = loomhttp.ErrorEncoder(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), loomhttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, loom.MethodKey, "MethodNoPayloadResult")
		ctx = context.WithValue(ctx, loom.ServiceKey, "ServiceNoPayloadResult")
		var err error
		res, err := endpoint(ctx, nil)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		if err := encodeResponse(ctx, w, res); err != nil {
			if errhandler != nil {
				errhandler(ctx, w, err)
			}
		}
	})
}
`

var ServerPayloadResultHandlerConstructorCode = `// NewMethodPayloadResultHandler creates a HTTP handler which loads the HTTP
// request and calls the "ServicePayloadResult" service "MethodPayloadResult"
// endpoint.
func NewMethodPayloadResultHandler(
	endpoint loom.Endpoint,
	mux loomhttp.Muxer,
	decoder func(*http.Request) loomhttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) loomhttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(ctx context.Context, err error) loomhttp.Statuser,
) http.Handler {
	var (
		decodeRequest  = DecodeMethodPayloadResultRequest(mux, decoder)
		encodeResponse = EncodeMethodPayloadResultResponse(encoder)
		encodeError    = loomhttp.ErrorEncoder(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), loomhttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, loom.MethodKey, "MethodPayloadResult")
		ctx = context.WithValue(ctx, loom.ServiceKey, "ServicePayloadResult")
		payload, err := decodeRequest(r)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		res, err := endpoint(ctx, payload)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		if err := encodeResponse(ctx, w, res); err != nil {
			if errhandler != nil {
				errhandler(ctx, w, err)
			}
		}
	})
}
`

var ServerPayloadResultErrorHandlerConstructorCode = `// NewMethodPayloadResultErrorHandler creates a HTTP handler which loads the
// HTTP request and calls the "ServicePayloadResultError" service
// "MethodPayloadResultError" endpoint.
func NewMethodPayloadResultErrorHandler(
	endpoint loom.Endpoint,
	mux loomhttp.Muxer,
	decoder func(*http.Request) loomhttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) loomhttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(ctx context.Context, err error) loomhttp.Statuser,
) http.Handler {
	var (
		decodeRequest  = DecodeMethodPayloadResultErrorRequest(mux, decoder)
		encodeResponse = EncodeMethodPayloadResultErrorResponse(encoder)
		encodeError    = EncodeMethodPayloadResultErrorError(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), loomhttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, loom.MethodKey, "MethodPayloadResultError")
		ctx = context.WithValue(ctx, loom.ServiceKey, "ServicePayloadResultError")
		payload, err := decodeRequest(r)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		res, err := endpoint(ctx, payload)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		if err := encodeResponse(ctx, w, res); err != nil {
			if errhandler != nil {
				errhandler(ctx, w, err)
			}
		}
	})
}
`

var ServerSkipResponseBodyEncodeDecodeCode = `// NewMethodSkipResponseBodyEncodeDecodeHandler creates a HTTP handler which
// loads the HTTP request and calls the "ServiceSkipResponseBodyEncodeDecode"
// service "MethodSkipResponseBodyEncodeDecode" endpoint.
func NewMethodSkipResponseBodyEncodeDecodeHandler(
	endpoint loom.Endpoint,
	mux loomhttp.Muxer,
	decoder func(*http.Request) loomhttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) loomhttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(ctx context.Context, err error) loomhttp.Statuser,
) http.Handler {
	var (
		encodeResponse = EncodeMethodSkipResponseBodyEncodeDecodeResponse(encoder)
		encodeError    = EncodeMethodSkipResponseBodyEncodeDecodeError(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), loomhttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, loom.MethodKey, "MethodSkipResponseBodyEncodeDecode")
		ctx = context.WithValue(ctx, loom.ServiceKey, "ServiceSkipResponseBodyEncodeDecode")
		var err error
		res, err := endpoint(ctx, nil)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		o := res.(*serviceskipresponsebodyencodedecode.MethodSkipResponseBodyEncodeDecodeResponseData)
		defer o.Body.Close()
		if wt, ok := o.Body.(io.WriterTo); ok {
			if err := encodeResponse(ctx, w, res); err != nil {
				if errhandler != nil {
					errhandler(ctx, w, err)
				}
				return
			}
			n, err := wt.WriteTo(w)
			if err != nil {
				if n == 0 {
					if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
						errhandler(ctx, w, err)
					}
				} else {
					http.NewResponseController(w).Flush()
					panic(http.ErrAbortHandler) // too late to write an error
				}
			}
			return
		}
		// handle immediate read error like a returned error
		buf := bufio.NewReader(o.Body)
		if _, err := buf.Peek(1); err != nil && err != io.EOF {
			if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		if err := encodeResponse(ctx, w, res); err != nil {
			if errhandler != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		if _, err := io.Copy(w, buf); err != nil {
			http.NewResponseController(w).Flush()
			panic(http.ErrAbortHandler) // too late to write an error
		}
	})
}
`
