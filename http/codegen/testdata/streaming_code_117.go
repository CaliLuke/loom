package testdata


var BidirectionalStreamingUserTypeMapClientStreamSendCode = `// Send streams instances of
// "map[string]*bidirectionalstreamingusertypemapservice.RequestType" to the
// "BidirectionalStreamingUserTypeMapMethod" endpoint websocket connection.
func (s *BidirectionalStreamingUserTypeMapMethodClientStream) Send(v map[string]*bidirectionalstreamingusertypemapservice.RequestType) error {
	body := NewMapStringRequestType(v)
	return s.conn.WriteJSON(body)
}

// SendWithContext streams instances of
// "map[string]*bidirectionalstreamingusertypemapservice.RequestType" to the
// "BidirectionalStreamingUserTypeMapMethod" endpoint websocket connection with
// context.
func (s *BidirectionalStreamingUserTypeMapMethodClientStream) SendWithContext(ctx context.Context, v map[string]*bidirectionalstreamingusertypemapservice.RequestType) error {
	return s.Send(v)
}
`


