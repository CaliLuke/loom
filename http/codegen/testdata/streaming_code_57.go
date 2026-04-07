package testdata


var StreamingPayloadUserTypeArrayClientStreamSendCode = `// Send streams instances of
// "[]*streamingpayloadusertypearrayservice.RequestType" to the
// "StreamingPayloadUserTypeArrayMethod" endpoint websocket connection.
func (s *StreamingPayloadUserTypeArrayMethodClientStream) Send(v []*streamingpayloadusertypearrayservice.RequestType) error {
	body := NewRequestType(v)
	return s.conn.WriteJSON(body)
}

// SendWithContext streams instances of
// "[]*streamingpayloadusertypearrayservice.RequestType" to the
// "StreamingPayloadUserTypeArrayMethod" endpoint websocket connection with
// context.
func (s *StreamingPayloadUserTypeArrayMethodClientStream) SendWithContext(ctx context.Context, v []*streamingpayloadusertypearrayservice.RequestType) error {
	return s.Send(v)
}
`


