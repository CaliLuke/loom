package testdata


var BidirectionalStreamingPrimitiveClientStreamSendCode = `// SendWithContext streams instances of "string" to the
// "BidirectionalStreamingPrimitiveMethod" endpoint websocket connection with
// context.
func (s *BidirectionalStreamingPrimitiveMethodClientStream) SendWithContext(ctx context.Context, v string) error {
	if err := ctx.Err(); err != nil {
		return err
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
	err := func() error {
		return s.conn.WriteJSON(v)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// Send streams instances of "string" to the
// "BidirectionalStreamingPrimitiveMethod" endpoint websocket connection.
func (s *BidirectionalStreamingPrimitiveMethodClientStream) Send(v string) error {
	return s.SendWithContext(context.Background(), v)
}
`


