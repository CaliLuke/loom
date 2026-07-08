package testdata

var BidirectionalStreamingServerHandlerInitCode = `// NewBidirectionalStreamingMethodHandler creates a HTTP handler which loads
// the HTTP request and calls the "BidirectionalStreamingService" service
// "BidirectionalStreamingMethod" endpoint.
func NewBidirectionalStreamingMethodHandler(
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
		decodeRequest = DecodeBidirectionalStreamingMethodRequest(mux, decoder)
		encodeError   = loomhttp.ErrorEncoder(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), loomhttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, loom.MethodKey, "BidirectionalStreamingMethod")
		ctx = context.WithValue(ctx, loom.ServiceKey, "BidirectionalStreamingService")
		obs, w := loomtransport.BeginHTTPRequest(ctx, w, "BidirectionalStreamingService", "BidirectionalStreamingMethod", r)
		defer obs.End()
		payload, err := decodeRequest(r)
		if err != nil {
			obs.Fail(loomtransport.ReasonRequestDecodeFailed)
			if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		v := &bidirectionalstreamingservice.BidirectionalStreamingMethodEndpointInput{
			Stream: &BidirectionalStreamingMethodServerStream{
				upgrader:   upgrader,
				configurer: configurer,
				cancel:     cancel,
				w:          w,
				r:          r,
				conn:       loomhttp.NewWebSocketStream(nil),
			},
			Payload: payload,
		}
		_, err = endpoint(ctx, v)
		if err != nil {
			obs.Fail(loomtransport.ReasonHandlerError)
			var stream *BidirectionalStreamingMethodServerStream
			if wrapper, ok := v.Stream.(interface{ Unwrap() any }); ok {
				stream = wrapper.Unwrap().(*BidirectionalStreamingMethodServerStream)
			} else {
				stream = v.Stream.(*BidirectionalStreamingMethodServerStream)
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
