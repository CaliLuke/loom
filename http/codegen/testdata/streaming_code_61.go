package testdata


var StreamingPayloadUserTypeMapClientStreamSendCode = `// Send streams instances of
// "map[string]*streamingpayloadusertypemapservice.RequestType" to the
// "StreamingPayloadUserTypeMapMethod" endpoint websocket connection.
func (s *StreamingPayloadUserTypeMapMethodClientStream) Send(v map[string]*streamingpayloadusertypemapservice.RequestType) error {
	body := NewMapStringRequestType(v)
	return s.conn.WriteJSON(body)
}

// SendWithContext streams instances of
// "map[string]*streamingpayloadusertypemapservice.RequestType" to the
// "StreamingPayloadUserTypeMapMethod" endpoint websocket connection with
// context.
func (s *StreamingPayloadUserTypeMapMethodClientStream) SendWithContext(ctx context.Context, v map[string]*streamingpayloadusertypemapservice.RequestType) error {
	return s.Send(v)
}
`


