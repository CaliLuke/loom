package testdata

var BidirectionalStreamingResultWithViewsClientStreamSetViewCode = `// SetView sets the view used to validate
// bidirectionalstreamingresultwithviewsservice.Usertype results received from
// the "BidirectionalStreamingResultWithViewsMethod" endpoint websocket
// connection.
func (s *BidirectionalStreamingResultWithViewsMethodClientStream) SetView(view string) {
	s.view = view
}
`
