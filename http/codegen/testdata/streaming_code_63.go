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
	formatter func(ctx context.Context, err error) loomhttp.StatusCoder,
	upgrader loomhttp.Upgrader,
	configurer loomhttp.ConnConfigureFunc,
	streamWritePolicy ...loomhttp.StreamWritePolicy,
) http.Handler {
	var writePolicy loomhttp.StreamWritePolicy
	if len(streamWritePolicy) > 0 {
		writePolicy = streamWritePolicy[0]
	}
	var (
		decodeRequest = DecodeBidirectionalStreamingMethodRequest(mux, decoder)
		encodeError   = loomhttp.ErrorEncoder(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lifecycle := loomhttp.NewHandlerLifecycle(w, r, "BidirectionalStreamingService", "BidirectionalStreamingMethod")
		defer lifecycle.End()
		ctx := lifecycle.Context()
		w = lifecycle.Writer()
		payload, err := decodeRequest(r)
		if err != nil {
			lifecycle.DecodeFailed(err, encodeError, errhandler)
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
				conn:       loomhttp.NewWebSocketStream(nil, writePolicy),
			},
			Payload: payload,
		}
		_, err = endpoint(ctx, v)
		if err != nil {
			var stream *BidirectionalStreamingMethodServerStream
			if wrapper, ok := v.Stream.(interface{ Unwrap() any }); ok {
				stream = wrapper.Unwrap().(*BidirectionalStreamingMethodServerStream)
			} else {
				stream = v.Stream.(*BidirectionalStreamingMethodServerStream)
			}
			lifecycle.HandlerFailed(err, stream != nil && stream.conn.Conn() != nil, encodeError, errhandler)
			return
		}
	})
}
`
