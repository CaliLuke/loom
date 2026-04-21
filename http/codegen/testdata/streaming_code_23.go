package testdata


var StreamingPayloadResultWithViewsServerStreamSendCode = `// SendAndCloseWithContext streams instances of
// "streamingpayloadresultwithviewsservice.Usertype" to the
// "StreamingPayloadResultWithViewsMethod" endpoint websocket connection with
// context and closes the connection.
func (s *StreamingPayloadResultWithViewsMethodServerStream) SendAndCloseWithContext(ctx context.Context, v *streamingpayloadresultwithviewsservice.Usertype) error {
	defer s.conn.Close()
	res := streamingpayloadresultwithviewsservice.NewViewedUsertype(v, s.view)
	var body any
	switch s.view {
	case "tiny":
		body = NewStreamingPayloadResultWithViewsMethodResponseBodyTiny(res.Projected)
	case "extended":
		body = NewStreamingPayloadResultWithViewsMethodResponseBodyExtended(res.Projected)
	case "default", "":
		body = NewStreamingPayloadResultWithViewsMethodResponseBody(res.Projected)
	}
	return s.conn.WriteJSON(body)
}

// SendAndClose streams instances of
// "streamingpayloadresultwithviewsservice.Usertype" to the
// "StreamingPayloadResultWithViewsMethod" endpoint websocket connection and
// closes the connection.
func (s *StreamingPayloadResultWithViewsMethodServerStream) SendAndClose(v *streamingpayloadresultwithviewsservice.Usertype) error {
	return s.SendAndCloseWithContext(context.Background(), v)
}
`


