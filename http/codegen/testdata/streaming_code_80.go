package testdata

var BidirectionalStreamingResultWithViewsServerStreamSetViewCode = `// SetView sets the view to render the
// bidirectionalstreamingresultwithviewsservice.Usertype type before sending to
// the "BidirectionalStreamingResultWithViewsMethod" endpoint websocket
// connection.
func (s *BidirectionalStreamingResultWithViewsMethodServerStream) SetView(view string) {
	s.view = view
}
`
