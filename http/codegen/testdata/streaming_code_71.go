package testdata


var BidirectionalStreamingNoPayloadServerHandlerInitCode = `// NewBidirectionalStreamingNoPayloadMethodHandler creates a HTTP handler which
// loads the HTTP request and calls the
// "BidirectionalStreamingNoPayloadService" service
// "BidirectionalStreamingNoPayloadMethod" endpoint.
func NewBidirectionalStreamingNoPayloadMethodHandler(
	endpoint loom.Endpoint,
	mux loomhttp.Muxer,
	decoder func(*http.Request) loomhttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) loomhttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(ctx context.Context, err error) loomhttp.Statuser,
	upgrader loomhttp.Upgrader,
	configurer loomhttp.ConnConfigureFunc,
) http.Handler {
	var (
		encodeError = loomhttp.ErrorEncoder(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), loomhttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, loom.MethodKey, "BidirectionalStreamingNoPayloadMethod")
		ctx = context.WithValue(ctx, loom.ServiceKey, "BidirectionalStreamingNoPayloadService")
		var err error
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		v := &bidirectionalstreamingnopayloadservice.BidirectionalStreamingNoPayloadMethodEndpointInput{
			Stream: &BidirectionalStreamingNoPayloadMethodServerStream{
				upgrader:   upgrader,
				configurer: configurer,
				cancel:     cancel,
				w:          w,
				r:          r,
			},
		}
		_, err = endpoint(ctx, v)
		if err != nil {
			var stream *BidirectionalStreamingNoPayloadMethodServerStream
			if wrapper, ok := v.Stream.(interface{ Unwrap() any }); ok {
				stream = wrapper.Unwrap().(*BidirectionalStreamingNoPayloadMethodServerStream)
			} else {
				stream = v.Stream.(*BidirectionalStreamingNoPayloadMethodServerStream)
			}
			if stream != nil && stream.conn != nil {
				// Response writer has been hijacked, do not encode the error
				if errhandler != nil {
					errhandler(ctx, w, err)
				}
				return
			}
			if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
				errhandler(ctx, w, err)
			}
			return
		}
	})
}
`


