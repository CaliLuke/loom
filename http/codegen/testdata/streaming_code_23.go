package testdata

var StreamingPayloadResultWithViewsServerStreamSendCode = `// SendAndCloseWithContext streams instances of
// "streamingpayloadresultwithviewsservice.Usertype" to the
// "StreamingPayloadResultWithViewsMethod" endpoint websocket connection with
// context and closes the connection.
func (s *StreamingPayloadResultWithViewsMethodServerStream) SendAndCloseWithContext(ctx context.Context, v *streamingpayloadresultwithviewsservice.Usertype) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := func() error {
		defer s.conn.Close()
		res, err := streamingpayloadresultwithviewsservice.NewViewedUsertype(v, s.view)
		if err != nil {
			return err
		}
		var body any
		switch s.view {
		case "tiny":
			body = NewStreamingPayloadResultWithViewsMethodResponseBodyTiny(res.Projected)
		case "extended":
			body = NewStreamingPayloadResultWithViewsMethodResponseBodyExtended(res.Projected)
		case "default", "":
			body = NewStreamingPayloadResultWithViewsMethodResponseBody(res.Projected)
		}
		return s.conn.WriteJSON(ctx, body)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// SendAndClose streams instances of
// "streamingpayloadresultwithviewsservice.Usertype" to the
// "StreamingPayloadResultWithViewsMethod" endpoint websocket connection and
// closes the connection.
func (s *StreamingPayloadResultWithViewsMethodServerStream) SendAndClose(v *streamingpayloadresultwithviewsservice.Usertype) error {
	return s.SendAndCloseWithContext(s.r.Context(), v)
}
`
