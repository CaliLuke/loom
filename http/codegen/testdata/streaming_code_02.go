package testdata

var StreamingResultPrimitiveMapServerStreamSendCode = `// SendWithContext streams instances of "map[int32]string" to the
// "StreamingResultPrimitiveMapMethod" endpoint websocket connection with
// context.
func (s *StreamingResultPrimitiveMapMethodServerStream) SendWithContext(ctx context.Context, v map[int32]string) error {
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
		body := res
		return s.conn.WriteJSON(ctx, body)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// Send streams instances of "map[int32]string" to the
// "StreamingResultPrimitiveMapMethod" endpoint websocket connection.
func (s *StreamingResultPrimitiveMapMethodServerStream) Send(v map[int32]string) error {
	return s.SendWithContext(s.r.Context(), v)
}
`
