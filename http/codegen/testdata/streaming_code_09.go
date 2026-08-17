package testdata

var StreamingPayloadServerHandlerInitCode = `// NewStreamingPayloadMethodHandler creates a HTTP handler which loads the HTTP
// request and calls the "StreamingPayloadService" service
// "StreamingPayloadMethod" endpoint.
func NewStreamingPayloadMethodHandler(
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
		decodeRequest = DecodeStreamingPayloadMethodRequest(mux, decoder)
		encodeError   = loomhttp.ErrorEncoder(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lifecycle := loomhttp.NewHandlerLifecycle(w, r, "StreamingPayloadService", "StreamingPayloadMethod")
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
		v := &streamingpayloadservice.StreamingPayloadMethodEndpointInput{
			Stream: &StreamingPayloadMethodServerStream{
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
			var stream *StreamingPayloadMethodServerStream
			if wrapper, ok := v.Stream.(interface{ Unwrap() any }); ok {
				stream = wrapper.Unwrap().(*StreamingPayloadMethodServerStream)
			} else {
				stream = v.Stream.(*StreamingPayloadMethodServerStream)
			}
			lifecycle.HandlerFailed(err, stream != nil && stream.conn.Conn() != nil, encodeError, errhandler)
			return
		}
	})
}
`
