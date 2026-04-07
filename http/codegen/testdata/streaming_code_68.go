package testdata


var BidirectionalStreamingClientStreamSendCode = `// Send streams instances of "bidirectionalstreamingservice.Request" to the
// "BidirectionalStreamingMethod" endpoint websocket connection.
func (s *BidirectionalStreamingMethodClientStream) Send(v *bidirectionalstreamingservice.Request) error {
	body := NewBidirectionalStreamingMethodStreamingBody(v)
	return s.conn.WriteJSON(body)
}

// SendWithContext streams instances of "bidirectionalstreamingservice.Request"
// to the "BidirectionalStreamingMethod" endpoint websocket connection with
// context.
func (s *BidirectionalStreamingMethodClientStream) SendWithContext(ctx context.Context, v *bidirectionalstreamingservice.Request) error {
	return s.Send(v)
}
`


