package testdata

var StreamingResultUserTypeArrayClientStreamRecvCode = `// Recv reads instances of "[]*streamingresultusertypearrayservice.UserType"
// from the "StreamingResultUserTypeArrayMethod" endpoint websocket connection.
func (s *StreamingResultUserTypeArrayMethodClientStream) Recv() ([]*streamingresultusertypearrayservice.UserType, error) {
	var (
		rv   []*streamingresultusertypearrayservice.UserType
		body []loom.Nullable[*UserTypeResponse]
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
	for i, e := range body {
		if _, ok := e.Value(); !ok {
			err = loom.MergeErrors(err, loom.InvalidNullElementError("body", i))
		}
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
	var (
		rv   []*streamingresultusertypearrayservice.UserType
		body []loom.Nullable[*UserTypeResponse]
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
	for i, e := range body {
		if _, ok := e.Value(); !ok {
			err = loom.MergeErrors(err, loom.InvalidNullElementError("body", i))
		}
	}
	if err != nil {
		return rv, err
	}
	res := NewStreamingResultUserTypeArrayMethodUserTypeOK(body)
	return res, nil
}
`
