package testdata


var BidirectionalStreamingNoPayloadClientStreamSendCode = `// SendWithContext streams instances of
// "bidirectionalstreamingnopayloadservice.Request" to the
// "BidirectionalStreamingNoPayloadMethod" endpoint websocket connection with
// context.
func (s *BidirectionalStreamingNoPayloadMethodClientStream) SendWithContext(ctx context.Context, v *bidirectionalstreamingnopayloadservice.Request) error {
	body := NewBidirectionalStreamingNoPayloadMethodStreamingBody(v)
	return s.conn.WriteJSON(body)
}

// Send streams instances of "bidirectionalstreamingnopayloadservice.Request"
// to the "BidirectionalStreamingNoPayloadMethod" endpoint websocket connection.
func (s *BidirectionalStreamingNoPayloadMethodClientStream) Send(v *bidirectionalstreamingnopayloadservice.Request) error {
	return s.SendWithContext(context.Background(), v)
}
`


