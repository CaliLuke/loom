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
	streamWritePolicy ...loomhttp.StreamWritePolicy,
) http.Handler {
	var writePolicy loomhttp.StreamWritePolicy
	if len(streamWritePolicy) > 0 {
		writePolicy = streamWritePolicy[0]
	}
	var (
		encodeError = loomhttp.ErrorEncoder(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lifecycle := loomhttp.NewHandlerLifecycle(w, r, "BidirectionalStreamingNoPayloadService", "BidirectionalStreamingNoPayloadMethod")
		defer lifecycle.End()
		ctx := lifecycle.Context()
		w = lifecycle.Writer()
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
				conn:       loomhttp.NewWebSocketStream(nil, writePolicy),
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
			lifecycle.HandlerFailed(err, stream != nil && stream.conn.Conn() != nil, encodeError, errhandler)
			return
		}
	})
}
`
