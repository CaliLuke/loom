package testdata


var StreamingPayloadResultWithViewsClientStreamRecvCode = `// CloseAndRecv stops sending messages to the
// "StreamingPayloadResultWithViewsMethod" endpoint websocket connection and
// reads instances of "streamingpayloadresultwithviewsservice.Usertype" from
// the connection.
func (s *StreamingPayloadResultWithViewsMethodClientStream) CloseAndRecv() (*streamingpayloadresultwithviewsservice.Usertype, error) {
	var (
		rv   *streamingpayloadresultwithviewsservice.Usertype
		body StreamingPayloadResultWithViewsMethodResponseBody
		err  error
	)
	defer s.conn.Close()
	// Send a nil payload to the server implying end of message
	if err = s.conn.WriteJSON(nil); err != nil {
		return rv, err
	}
	err = s.conn.ReadJSON(&body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.conn.Close()
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	res := NewStreamingPayloadResultWithViewsMethodUsertypeOK(&body)
	vres := &streamingpayloadresultwithviewsserviceviews.Usertype{res, s.view}
	if err := streamingpayloadresultwithviewsserviceviews.ValidateUsertype(vres); err != nil {
		return rv, loomhttp.ErrValidationError("StreamingPayloadResultWithViewsService", "StreamingPayloadResultWithViewsMethod", err)
	}
	return streamingpayloadresultwithviewsservice.NewUsertype(vres), nil
}

// CloseAndRecvWithContext stops sending messages to the
// "StreamingPayloadResultWithViewsMethod" endpoint websocket connection and
// reads instances of "streamingpayloadresultwithviewsservice.Usertype" from
// the connection with context.
func (s *StreamingPayloadResultWithViewsMethodClientStream) CloseAndRecvWithContext(ctx context.Context) (*streamingpayloadresultwithviewsservice.Usertype, error) {
	var rv *streamingpayloadresultwithviewsservice.Usertype
	if err := ctx.Err(); err != nil {
		return rv, err
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
	v, err := s.CloseAndRecv()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return rv, ctxErr
		}
	}
	return v, err
}
`


