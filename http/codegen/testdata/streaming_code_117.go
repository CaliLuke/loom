package testdata


var BidirectionalStreamingUserTypeMapClientStreamSendCode = `// SendWithContext streams instances of
// "map[string]*bidirectionalstreamingusertypemapservice.RequestType" to the
// "BidirectionalStreamingUserTypeMapMethod" endpoint websocket connection with
// context.
func (s *BidirectionalStreamingUserTypeMapMethodClientStream) SendWithContext(ctx context.Context, v map[string]*bidirectionalstreamingusertypemapservice.RequestType) error {
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
		body := NewMapStringRequestType(v)
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
// "map[string]*bidirectionalstreamingusertypemapservice.RequestType" to the
// "BidirectionalStreamingUserTypeMapMethod" endpoint websocket connection.
func (s *BidirectionalStreamingUserTypeMapMethodClientStream) Send(v map[string]*bidirectionalstreamingusertypemapservice.RequestType) error {
	return s.SendWithContext(context.Background(), v)
}
`


