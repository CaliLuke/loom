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
		s.conn.Close()
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
	return s.Recv()
}
`


