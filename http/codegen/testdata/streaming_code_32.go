package testdata


var StreamingPayloadResultWithExplicitViewClientStreamRecvCode = `// CloseAndRecv stops sending messages to the
// "StreamingPayloadResultWithExplicitViewMethod" endpoint websocket connection
// and reads instances of
// "streamingpayloadresultwithexplicitviewservice.Usertype" from the connection.
func (s *StreamingPayloadResultWithExplicitViewMethodClientStream) CloseAndRecv() (*streamingpayloadresultwithexplicitviewservice.Usertype, error) {
	var (
		rv   *streamingpayloadresultwithexplicitviewservice.Usertype
		body StreamingPayloadResultWithExplicitViewMethodResponseBodyExtended
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
	res := NewStreamingPayloadResultWithExplicitViewMethodUsertypeOK(&body)
	vres := &streamingpayloadresultwithexplicitviewserviceviews.Usertype{res, "extended"}
	if err := streamingpayloadresultwithexplicitviewserviceviews.ValidateUsertype(vres); err != nil {
		return rv, loomhttp.ErrValidationError("StreamingPayloadResultWithExplicitViewService", "StreamingPayloadResultWithExplicitViewMethod", err)
	}
	return streamingpayloadresultwithexplicitviewservice.NewUsertype(vres), nil
}

// CloseAndRecvWithContext stops sending messages to the
// "StreamingPayloadResultWithExplicitViewMethod" endpoint websocket connection
// and reads instances of
// "streamingpayloadresultwithexplicitviewservice.Usertype" from the connection
// with context.
func (s *StreamingPayloadResultWithExplicitViewMethodClientStream) CloseAndRecvWithContext(ctx context.Context) (*streamingpayloadresultwithexplicitviewservice.Usertype, error) {
	return s.CloseAndRecv()
}
`


