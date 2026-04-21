package testdata


var StreamingResultUserTypeMapServerStreamSendCode = `// SendWithContext streams instances of
// "map[string]*streamingresultusertypemapservice.UserType" to the
// "StreamingResultUserTypeMapMethod" endpoint websocket connection with
// context.
func (s *StreamingResultUserTypeMapMethodServerStream) SendWithContext(ctx context.Context, v map[string]*streamingresultusertypemapservice.UserType) error {
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
	body := NewStreamingResultUserTypeMapMethodResponseBody(res)
	return s.conn.WriteJSON(body)
}

// Send streams instances of
// "map[string]*streamingresultusertypemapservice.UserType" to the
// "StreamingResultUserTypeMapMethod" endpoint websocket connection.
func (s *StreamingResultUserTypeMapMethodServerStream) Send(v map[string]*streamingresultusertypemapservice.UserType) error {
	return s.SendWithContext(context.Background(), v)
}
`


