package testdata


var BidirectionalStreamingUserTypeArrayClientStreamSendCode = `// Send streams instances of
// "[]*bidirectionalstreamingusertypearrayservice.RequestType" to the
// "BidirectionalStreamingUserTypeArrayMethod" endpoint websocket connection.
func (s *BidirectionalStreamingUserTypeArrayMethodClientStream) Send(v []*bidirectionalstreamingusertypearrayservice.RequestType) error {
	body := NewRequestType(v)
	return s.conn.WriteJSON(body)
}

// SendWithContext streams instances of
// "[]*bidirectionalstreamingusertypearrayservice.RequestType" to the
// "BidirectionalStreamingUserTypeArrayMethod" endpoint websocket connection
// with context.
func (s *BidirectionalStreamingUserTypeArrayMethodClientStream) SendWithContext(ctx context.Context, v []*bidirectionalstreamingusertypearrayservice.RequestType) error {
	return s.Send(v)
}
`


