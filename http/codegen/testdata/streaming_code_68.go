package testdata


var BidirectionalStreamingClientStreamSendCode = `// SendWithContext streams instances of "bidirectionalstreamingservice.Request"
// to the "BidirectionalStreamingMethod" endpoint websocket connection with
// context.
func (s *BidirectionalStreamingMethodClientStream) SendWithContext(ctx context.Context, v *bidirectionalstreamingservice.Request) error {
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
		body := NewBidirectionalStreamingMethodStreamingBody(v)
		return s.conn.WriteJSON(body)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// Send streams instances of "bidirectionalstreamingservice.Request" to the
// "BidirectionalStreamingMethod" endpoint websocket connection.
func (s *BidirectionalStreamingMethodClientStream) Send(v *bidirectionalstreamingservice.Request) error {
	return s.SendWithContext(context.Background(), v)
}
`


