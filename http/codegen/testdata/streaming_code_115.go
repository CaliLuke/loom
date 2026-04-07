package testdata


var BidirectionalStreamingUserTypeMapServerStreamSendCode = `// Send streams instances of
// "map[string]*bidirectionalstreamingusertypemapservice.ResultType" to the
// "BidirectionalStreamingUserTypeMapMethod" endpoint websocket connection.
func (s *BidirectionalStreamingUserTypeMapMethodServerStream) Send(v map[string]*bidirectionalstreamingusertypemapservice.ResultType) error {
	var err error
	// Upgrade the HTTP connection to a websocket connection only once. Connection
	// upgrade is done here so that authorization logic in the endpoint is executed
	// before calling the actual service method which may call Send().
	s.once.Do(func() {
		var conn *websocket.Conn
		conn, err = s.upgrader.Upgrade(s.w, s.r, nil)
		if err != nil {
			s.upgradeErr = err
			return
		}
		if s.configurer != nil {
			conn = s.configurer(conn, s.cancel)
		}
		s.conn = conn
	})
	if s.upgradeErr != nil {
		return s.upgradeErr
	}
	res := v
	body := NewBidirectionalStreamingUserTypeMapMethodResponseBody(res)
	return s.conn.WriteJSON(body)
}

// SendWithContext streams instances of
// "map[string]*bidirectionalstreamingusertypemapservice.ResultType" to the
// "BidirectionalStreamingUserTypeMapMethod" endpoint websocket connection with
// context.
func (s *BidirectionalStreamingUserTypeMapMethodServerStream) SendWithContext(ctx context.Context, v map[string]*bidirectionalstreamingusertypemapservice.ResultType) error {
	return s.Send(v)
}
`


