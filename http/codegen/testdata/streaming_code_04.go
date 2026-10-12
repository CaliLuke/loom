package testdata

var StreamingResultUserTypeArrayServerStreamSendCode = `// SendWithContext streams instances of
// "[]*streamingresultusertypearrayservice.UserType" to the
// "StreamingResultUserTypeArrayMethod" endpoint websocket connection with
// context.
func (s *StreamingResultUserTypeArrayMethodServerStream) SendWithContext(ctx context.Context, v []*streamingresultusertypearrayservice.UserType) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := func() error {
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
			if err = s.conn.SetConn(s.r.Context(), conn, s.cancel); err != nil {
				s.upgradeErr = err
				return
			}
			if err = ctx.Err(); err != nil {
				if closeErr := s.conn.Close(); closeErr != nil {
					s.upgradeErr = closeErr
					return
				}
				s.upgradeErr = err
				return
			}
		})
		if s.upgradeErr != nil {
			return s.upgradeErr
		}
		res := v
		body := NewStreamingResultUserTypeArrayMethodResponseBody(res)
		return s.conn.WriteJSON(ctx, body)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// Send streams instances of "[]*streamingresultusertypearrayservice.UserType"
// to the "StreamingResultUserTypeArrayMethod" endpoint websocket connection.
func (s *StreamingResultUserTypeArrayMethodServerStream) Send(v []*streamingresultusertypearrayservice.UserType) error {
	return s.SendWithContext(s.r.Context(), v)
}
`
