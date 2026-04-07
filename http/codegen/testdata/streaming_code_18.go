package testdata


var StreamingPayloadNoPayloadClientStreamRecvCode = `// CloseAndRecv stops sending messages to the "StreamingPayloadNoPayloadMethod"
// endpoint websocket connection and reads instances of
// "streamingpayloadnopayloadservice.UserType" from the connection.
func (s *StreamingPayloadNoPayloadMethodClientStream) CloseAndRecv() (*streamingpayloadnopayloadservice.UserType, error) {
	var (
		rv   *streamingpayloadnopayloadservice.UserType
		body StreamingPayloadNoPayloadMethodResponseBody
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
	res := NewStreamingPayloadNoPayloadMethodUserTypeOK(&body)
	return res, nil
}

// CloseAndRecvWithContext stops sending messages to the
// "StreamingPayloadNoPayloadMethod" endpoint websocket connection and reads
// instances of "streamingpayloadnopayloadservice.UserType" from the connection
// with context.
func (s *StreamingPayloadNoPayloadMethodClientStream) CloseAndRecvWithContext(ctx context.Context) (*streamingpayloadnopayloadservice.UserType, error) {
	return s.CloseAndRecv()
}
`


