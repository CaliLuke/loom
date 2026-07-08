package testdata

var BidirectionalStreamingResultCollectionWithViewsClientStreamSetViewCode = `// SetView sets the view to render the any type before sending to the
// "BidirectionalStreamingResultCollectionWithViewsMethod" endpoint websocket
// connection.
func (s *BidirectionalStreamingResultCollectionWithViewsMethodClientStream) SetView(view string) {
	s.view = view
}
`
