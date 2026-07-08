package testdata

var StreamingResultUserTypeMapClientStreamRecvCode = `// Recv reads instances of
// "map[string]*streamingresultusertypemapservice.UserType" from the
// "StreamingResultUserTypeMapMethod" endpoint websocket connection.
func (s *StreamingResultUserTypeMapMethodClientStream) Recv() (map[string]*streamingresultusertypemapservice.UserType, error) {
	var (
		rv   map[string]*streamingresultusertypemapservice.UserType
		body map[string]*UserTypeResponse
		err  error
	)
	err = s.conn.ReadJSON(context.Background(), &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.closeOnce.Do(func() {
			if s.done != nil {
				close(s.done)
			}
		})
		if closeErr := s.conn.Close(); closeErr != nil {
			return rv, closeErr
		}
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	res := NewStreamingResultUserTypeMapMethodMapStringUserTypeOK(body)
	return res, nil
}

// RecvWithContext reads instances of
// "map[string]*streamingresultusertypemapservice.UserType" from the
// "StreamingResultUserTypeMapMethod" endpoint websocket connection with
// context.
func (s *StreamingResultUserTypeMapMethodClientStream) RecvWithContext(ctx context.Context) (map[string]*streamingresultusertypemapservice.UserType, error) {
	var (
		rv   map[string]*streamingresultusertypemapservice.UserType
		body map[string]*UserTypeResponse
		err  error
	)
	if err := ctx.Err(); err != nil {
		return rv, err
	}
	err = s.conn.ReadJSON(ctx, &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.closeOnce.Do(func() {
			if s.done != nil {
				close(s.done)
			}
		})
		if closeErr := s.conn.Close(); closeErr != nil {
			return rv, closeErr
		}
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	res := NewStreamingResultUserTypeMapMethodMapStringUserTypeOK(body)
	return res, nil
}
`
