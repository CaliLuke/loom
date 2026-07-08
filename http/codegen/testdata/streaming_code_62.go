package testdata


var StreamingPayloadUserTypeMapClientStreamRecvCode = `// CloseAndRecv stops sending messages to the
// "StreamingPayloadUserTypeMapMethod" endpoint websocket connection and reads
// instances of "[]string" from the connection.
func (s *StreamingPayloadUserTypeMapMethodClientStream) CloseAndRecv() ([]string, error) {
	var (
		rv   []string
		body []string
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
	return body, nil
}

// CloseAndRecvWithContext stops sending messages to the
// "StreamingPayloadUserTypeMapMethod" endpoint websocket connection and reads
// instances of "[]string" from the connection with context.
func (s *StreamingPayloadUserTypeMapMethodClientStream) CloseAndRecvWithContext(ctx context.Context) ([]string, error) {
	var rv []string
	if err := ctx.Err(); err != nil {
		return rv, err
	}
	stopContextWatch := context.AfterFunc(ctx, func() {
		if s.conn == nil {
			return
		}
		if closeErr := s.conn.Close(); closeErr != nil {
			return
		}
	})
	defer stopContextWatch()
	v, err := s.CloseAndRecv()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return rv, ctxErr
		}
	}
	return v, err
}
`


