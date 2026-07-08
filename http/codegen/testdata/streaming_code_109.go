package testdata


var BidirectionalStreamingPrimitiveMapClientStreamSendCode = `// SendWithContext streams instances of "map[string]int32" to the
// "BidirectionalStreamingPrimitiveMapMethod" endpoint websocket connection
// with context.
func (s *BidirectionalStreamingPrimitiveMapMethodClientStream) SendWithContext(ctx context.Context, v map[string]int32) error {
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

// Send streams instances of "map[string]int32" to the
// "BidirectionalStreamingPrimitiveMapMethod" endpoint websocket connection.
func (s *BidirectionalStreamingPrimitiveMapMethodClientStream) Send(v map[string]int32) error {
	return s.SendWithContext(context.Background(), v)
}
`


