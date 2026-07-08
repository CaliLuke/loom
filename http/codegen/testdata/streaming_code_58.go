package testdata

var StreamingPayloadUserTypeArrayClientStreamRecvCode = `// CloseAndRecv stops sending messages to the
// "StreamingPayloadUserTypeArrayMethod" endpoint websocket connection and
// reads instances of "string" from the connection.
func (s *StreamingPayloadUserTypeArrayMethodClientStream) CloseAndRecv() (string, error) {
	var (
		rv   string
		body string
		err  error
	)
	defer s.conn.Close()
	// Send a nil payload to the server implying end of message
	if err = s.conn.WriteJSON(context.Background(), nil); err != nil {
		return rv, err
	}
	err = s.conn.ReadJSON(context.Background(), &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.conn.Close()
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	return body, nil
}

// CloseAndRecvWithContext stops sending messages to the
// "StreamingPayloadUserTypeArrayMethod" endpoint websocket connection and
// reads instances of "string" from the connection with context.
func (s *StreamingPayloadUserTypeArrayMethodClientStream) CloseAndRecvWithContext(ctx context.Context) (string, error) {
	var (
		rv   string
		body string
		err  error
	)
	if err := ctx.Err(); err != nil {
		return rv, err
	}
	defer s.conn.Close()
	// Send a nil payload to the server implying end of message
	if err = s.conn.WriteJSON(ctx, nil); err != nil {
		return rv, err
	}
	err = s.conn.ReadJSON(ctx, &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.conn.Close()
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	return body, nil
}
`
