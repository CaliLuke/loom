package testdata

var StreamingResultUserTypeArrayClientStreamRecvCode = `// Recv reads instances of "[]*streamingresultusertypearrayservice.UserType"
// from the "StreamingResultUserTypeArrayMethod" endpoint websocket connection.
func (s *StreamingResultUserTypeArrayMethodClientStream) Recv() ([]*streamingresultusertypearrayservice.UserType, error) {
	var (
		rv   []*streamingresultusertypearrayservice.UserType
		body []*UserTypeResponse
		err  error
	)
	err = s.conn.ReadJSON(&body)
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
	res := NewStreamingResultUserTypeArrayMethodUserTypeOK(body)
	return res, nil
}

// RecvWithContext reads instances of
// "[]*streamingresultusertypearrayservice.UserType" from the
// "StreamingResultUserTypeArrayMethod" endpoint websocket connection with
// context.
func (s *StreamingResultUserTypeArrayMethodClientStream) RecvWithContext(ctx context.Context) ([]*streamingresultusertypearrayservice.UserType, error) {
	var rv []*streamingresultusertypearrayservice.UserType
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
