package testdata

var StreamingPayloadResultWithExplicitViewServerStreamSendCode = `// SendAndCloseWithContext streams instances of
// "streamingpayloadresultwithexplicitviewservice.Usertype" to the
// "StreamingPayloadResultWithExplicitViewMethod" endpoint websocket connection
// with context and closes the connection.
func (s *StreamingPayloadResultWithExplicitViewMethodServerStream) SendAndCloseWithContext(ctx context.Context, v *streamingpayloadresultwithexplicitviewservice.Usertype) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	stopContextWatch := context.AfterFunc(ctx, func() {
		if s.conn == nil {
			return
		}
		if closeErr := s.conn.Close(); closeErr != nil {
			return
		}
	})
	defer stopContextWatch()
	err := func() error {
		defer s.conn.Close()
		res, err := streamingpayloadresultwithexplicitviewservice.NewViewedUsertype(v, "extended")
		if err != nil {
			return err
		}
		body := NewStreamingPayloadResultWithExplicitViewMethodResponseBodyExtended(res.Projected)
		return s.conn.WriteJSON(body)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// SendAndClose streams instances of
// "streamingpayloadresultwithexplicitviewservice.Usertype" to the
// "StreamingPayloadResultWithExplicitViewMethod" endpoint websocket connection
// and closes the connection.
func (s *StreamingPayloadResultWithExplicitViewMethodServerStream) SendAndClose(v *streamingpayloadresultwithexplicitviewservice.Usertype) error {
	return s.SendAndCloseWithContext(s.r.Context(), v)
}
`
