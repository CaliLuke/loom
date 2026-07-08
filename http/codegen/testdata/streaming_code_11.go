package testdata


var StreamingPayloadServerStreamRecvCode = `// Recv reads instances of "streamingpayloadservice.Request" from the
// "StreamingPayloadMethod" endpoint websocket connection.
func (s *StreamingPayloadMethodServerStream) Recv() (*streamingpayloadservice.Request, error) {
	return s.RecvWithContext(s.r.Context())
}

// RecvWithContext reads instances of "streamingpayloadservice.Request" from
// the "StreamingPayloadMethod" endpoint websocket connection with context.
func (s *StreamingPayloadMethodServerStream) RecvWithContext(ctx context.Context) (*streamingpayloadservice.Request, error) {
	var (
		rv  *streamingpayloadservice.Request
		msg *StreamingPayloadMethodStreamingBody
		err error
	)
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
	// Upgrade the HTTP connection to a websocket connection only once. Connection
	// upgrade is done here so that authorization logic in the endpoint is executed
	// before calling the actual service method which may call Recv().
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
		return rv, s.upgradeErr
	}
	if err = s.conn.ReadJSON(&msg); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return rv, ctxErr
		}
		return rv, err
	}
	if msg == nil {
		return rv, io.EOF
	}
	body := *msg
	err = ValidateStreamingPayloadMethodStreamingBody(&body)
	if err != nil {
		return rv, err
	}
	return NewStreamingPayloadMethodStreamingBody(msg), nil
}
`

