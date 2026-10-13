package testdata

var BidirectionalStreamingResultCollectionWithViewsServerStreamSendCode = `// SendWithContext streams instances of
// "bidirectionalstreamingresultcollectionwithviewsservice.UsertypeCollection"
// to the "BidirectionalStreamingResultCollectionWithViewsMethod" endpoint
// websocket connection with context.
func (s *BidirectionalStreamingResultCollectionWithViewsMethodServerStream) SendWithContext(ctx context.Context, v bidirectionalstreamingresultcollectionwithviewsservice.UsertypeCollection) error {
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
			if err = s.conn.SetConn(s.r.Context(), conn, s.cancel); err != nil {
				s.upgradeErr = err
				return
			}
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
		res, err := bidirectionalstreamingresultcollectionwithviewsservice.NewViewedUsertypeCollection(v, s.view)
		if err != nil {
			return err
		}
		var body any
		switch res.View {
		case "tiny":
			body = NewUsertypeResponseTinyCollection(res.Projected)
		case "extended":
			body = NewUsertypeResponseExtendedCollection(res.Projected)
		case "default", "":
			body = NewUsertypeResponseCollection(res.Projected)
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
// "bidirectionalstreamingresultcollectionwithviewsservice.UsertypeCollection"
// to the "BidirectionalStreamingResultCollectionWithViewsMethod" endpoint
// websocket connection.
func (s *BidirectionalStreamingResultCollectionWithViewsMethodServerStream) Send(v bidirectionalstreamingresultcollectionwithviewsservice.UsertypeCollection) error {
	return s.SendWithContext(s.r.Context(), v)
}
`
