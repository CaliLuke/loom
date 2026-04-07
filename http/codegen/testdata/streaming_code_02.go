package testdata


var StreamingResultPrimitiveMapServerStreamSendCode = `// Send streams instances of "map[int32]string" to the
// "StreamingResultPrimitiveMapMethod" endpoint websocket connection.
func (s *StreamingResultPrimitiveMapMethodServerStream) Send(v map[int32]string) error {
	var err error
	// Upgrade the HTTP connection to a websocket connection only once. Connection
	// upgrade is done here so that authorization logic in the endpoint is executed
	// before calling the actual service method which may call Send().
	s.once.Do(func() {
		var conn *websocket.Conn
		conn, err = s.upgrader.Upgrade(s.w, s.r, nil)
		if err != nil {
			s.upgradeErr = err
			return
		}
		if s.configurer != nil {
			conn = s.configurer(conn, s.cancel)
		}
		s.conn = conn
	})
	if s.upgradeErr != nil {
		return s.upgradeErr
	}
	res := v
	return s.conn.WriteJSON(res)
}

// SendWithContext streams instances of "map[int32]string" to the
// "StreamingResultPrimitiveMapMethod" endpoint websocket connection with
// context.
func (s *StreamingResultPrimitiveMapMethodServerStream) SendWithContext(ctx context.Context, v map[int32]string) error {
	return s.Send(v)
}
`


