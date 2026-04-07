package testdata


var StreamingPayloadResultWithExplicitViewServerStreamSendCode = `// SendAndClose streams instances of
// "streamingpayloadresultwithexplicitviewservice.Usertype" to the
// "StreamingPayloadResultWithExplicitViewMethod" endpoint websocket connection
// and closes the connection.
func (s *StreamingPayloadResultWithExplicitViewMethodServerStream) SendAndClose(v *streamingpayloadresultwithexplicitviewservice.Usertype) error {
	defer s.conn.Close()
	res := streamingpayloadresultwithexplicitviewservice.NewViewedUsertype(v, "extended")
	body := NewStreamingPayloadResultWithExplicitViewMethodResponseBodyExtended(res.Projected)
	return s.conn.WriteJSON(body)
}

// SendAndCloseWithContext streams instances of
// "streamingpayloadresultwithexplicitviewservice.Usertype" to the
// "StreamingPayloadResultWithExplicitViewMethod" endpoint websocket connection
// with context and closes the connection.
func (s *StreamingPayloadResultWithExplicitViewMethodServerStream) SendAndCloseWithContext(ctx context.Context, v *streamingpayloadresultwithexplicitviewservice.Usertype) error {
	return s.SendAndClose(v)
}
`


