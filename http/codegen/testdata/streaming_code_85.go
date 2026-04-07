package testdata


var BidirectionalStreamingResultWithExplicitViewServerStreamSendCode = `// Send streams instances of
// "bidirectionalstreamingresultwithexplicitviewservice.Usertype" to the
// "BidirectionalStreamingResultWithExplicitViewMethod" endpoint websocket
// connection.
func (s *BidirectionalStreamingResultWithExplicitViewMethodServerStream) Send(v *bidirectionalstreamingresultwithexplicitviewservice.Usertype) error {
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
	res := bidirectionalstreamingresultwithexplicitviewservice.NewViewedUsertype(v, "extended")
	body := NewBidirectionalStreamingResultWithExplicitViewMethodResponseBodyExtended(res.Projected)
	return s.conn.WriteJSON(body)
}

// SendWithContext streams instances of
// "bidirectionalstreamingresultwithexplicitviewservice.Usertype" to the
// "BidirectionalStreamingResultWithExplicitViewMethod" endpoint websocket
// connection with context.
func (s *BidirectionalStreamingResultWithExplicitViewMethodServerStream) SendWithContext(ctx context.Context, v *bidirectionalstreamingresultwithexplicitviewservice.Usertype) error {
	return s.Send(v)
}
`


