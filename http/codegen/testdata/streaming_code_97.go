package testdata


var BidirectionalStreamingResultCollectionWithExplicitViewClientStreamSendCode = `// SendWithContext streams instances of "any" to the
// "BidirectionalStreamingResultCollectionWithExplicitViewMethod" endpoint
// websocket connection with context.
func (s *BidirectionalStreamingResultCollectionWithExplicitViewMethodClientStream) SendWithContext(ctx context.Context, v any) error {
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

// Send streams instances of "any" to the
// "BidirectionalStreamingResultCollectionWithExplicitViewMethod" endpoint
// websocket connection.
func (s *BidirectionalStreamingResultCollectionWithExplicitViewMethodClientStream) Send(v any) error {
	return s.SendWithContext(context.Background(), v)
}
`


