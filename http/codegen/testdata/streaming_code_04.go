package testdata


var StreamingResultUserTypeArrayServerStreamSendCode = `// Send streams instances of "[]*streamingresultusertypearrayservice.UserType"
// to the "StreamingResultUserTypeArrayMethod" endpoint websocket connection.
func (s *StreamingResultUserTypeArrayMethodServerStream) Send(v []*streamingresultusertypearrayservice.UserType) error {
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
	body := NewStreamingResultUserTypeArrayMethodResponseBody(res)
	return s.conn.WriteJSON(body)
}

// SendWithContext streams instances of
// "[]*streamingresultusertypearrayservice.UserType" to the
// "StreamingResultUserTypeArrayMethod" endpoint websocket connection with
// context.
func (s *StreamingResultUserTypeArrayMethodServerStream) SendWithContext(ctx context.Context, v []*streamingresultusertypearrayservice.UserType) error {
	return s.Send(v)
}
`


