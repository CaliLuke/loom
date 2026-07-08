package testdata

var BidirectionalStreamingResultCollectionWithExplicitViewServerStreamSendCode = `// SendWithContext streams instances of
// "bidirectionalstreamingresultcollectionwithexplicitviewservice.UsertypeCollection"
// to the "BidirectionalStreamingResultCollectionWithExplicitViewMethod"
// endpoint websocket connection with context.
func (s *BidirectionalStreamingResultCollectionWithExplicitViewMethodServerStream) SendWithContext(ctx context.Context, v bidirectionalstreamingresultcollectionwithexplicitviewservice.UsertypeCollection) error {
	if err := ctx.Err(); err != nil {
		return err
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
	err := func() error {
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
		res, err := bidirectionalstreamingresultcollectionwithexplicitviewservice.NewViewedUsertypeCollection(v, "tiny")
		if err != nil {
			return err
		}
		body := NewUsertypeResponseTinyCollection(res.Projected)
		return s.conn.WriteJSON(body)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// Send streams instances of
// "bidirectionalstreamingresultcollectionwithexplicitviewservice.UsertypeCollection"
// to the "BidirectionalStreamingResultCollectionWithExplicitViewMethod"
// endpoint websocket connection.
func (s *BidirectionalStreamingResultCollectionWithExplicitViewMethodServerStream) Send(v bidirectionalstreamingresultcollectionwithexplicitviewservice.UsertypeCollection) error {
	return s.SendWithContext(s.r.Context(), v)
}
`
