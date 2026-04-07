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
	return s.Recv()
}
`


