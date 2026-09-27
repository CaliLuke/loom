package testdata

var BidirectionalStreamingResultCollectionWithViewsClientStreamSetViewCode = `// SetView sets the view used to validate
// bidirectionalstreamingresultcollectionwithviewsservice.UsertypeCollection
// results received from the
// "BidirectionalStreamingResultCollectionWithViewsMethod" endpoint websocket
// connection.
func (s *BidirectionalStreamingResultCollectionWithViewsMethodClientStream) SetView(view string) {
	s.view = view
}
`
