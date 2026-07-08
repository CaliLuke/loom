package testdata

var BidirectionalStreamingResultWithViewsServerStreamSendCode = `// SendWithContext streams instances of
// "bidirectionalstreamingresultwithviewsservice.Usertype" to the
// "BidirectionalStreamingResultWithViewsMethod" endpoint websocket connection
// with context.
func (s *BidirectionalStreamingResultWithViewsMethodServerStream) SendWithContext(ctx context.Context, v *bidirectionalstreamingresultwithviewsservice.Usertype) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := func() error {
		var err error
		// Upgrade the HTTP connection to a websocket connection only once. Connection
		// upgrade is done here so that authorization logic in the endpoint is executed
		// before calling the actual service method which may call Send().
		s.once.Do(func() {
			respHdr := make(http.Header)
			respHdr.Add("loom-view", s.view)
			var conn *websocket.Conn
			conn, err = s.upgrader.Upgrade(s.w, s.r, respHdr)
			if err != nil {
				s.upgradeErr = err
				return
			}
			if s.configurer != nil {
				conn = s.configurer(conn, s.cancel)
			}
			s.conn.SetConn(conn)
			if err = ctx.Err(); err != nil {
				if closeErr := s.conn.Close(); closeErr != nil {
					s.upgradeErr = closeErr
					return
				}
				s.upgradeErr = err
				return
			}
		})
		if s.upgradeErr != nil {
			return s.upgradeErr
		}
		res, err := bidirectionalstreamingresultwithviewsservice.NewViewedUsertype(v, s.view)
		if err != nil {
			return err
		}
		var body any
		switch s.view {
		case "tiny":
			body = NewBidirectionalStreamingResultWithViewsMethodResponseBodyTiny(res.Projected)
		case "extended":
			body = NewBidirectionalStreamingResultWithViewsMethodResponseBodyExtended(res.Projected)
		case "default", "":
			body = NewBidirectionalStreamingResultWithViewsMethodResponseBody(res.Projected)
		}
		return s.conn.WriteJSON(ctx, body)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// Send streams instances of
// "bidirectionalstreamingresultwithviewsservice.Usertype" to the
// "BidirectionalStreamingResultWithViewsMethod" endpoint websocket connection.
func (s *BidirectionalStreamingResultWithViewsMethodServerStream) Send(v *bidirectionalstreamingresultwithviewsservice.Usertype) error {
	return s.SendWithContext(s.r.Context(), v)
}
`
