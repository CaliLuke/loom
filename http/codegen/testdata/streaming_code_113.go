package testdata


var BidirectionalStreamingUserTypeArrayClientStreamSendCode = `// SendWithContext streams instances of
// "[]*bidirectionalstreamingusertypearrayservice.RequestType" to the
// "BidirectionalStreamingUserTypeArrayMethod" endpoint websocket connection
// with context.
func (s *BidirectionalStreamingUserTypeArrayMethodClientStream) SendWithContext(ctx context.Context, v []*bidirectionalstreamingusertypearrayservice.RequestType) error {
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
		body := NewRequestType(v)
		return s.conn.WriteJSON(body)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// Send streams instances of
// "[]*bidirectionalstreamingusertypearrayservice.RequestType" to the
// "BidirectionalStreamingUserTypeArrayMethod" endpoint websocket connection.
func (s *BidirectionalStreamingUserTypeArrayMethodClientStream) Send(v []*bidirectionalstreamingusertypearrayservice.RequestType) error {
	return s.SendWithContext(context.Background(), v)
}
`


