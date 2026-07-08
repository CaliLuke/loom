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
	err = s.conn.ReadJSON(&body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.conn.Close()
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
	var rv map[string]*streamingresultusertypemapservice.UserType
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
	v, err := s.Recv()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return rv, ctxErr
		}
	}
	return v, err
}
`


