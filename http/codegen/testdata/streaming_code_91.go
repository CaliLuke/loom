package testdata


var BidirectionalStreamingResultCollectionWithViewsServerStreamSetViewCode = `// SetView sets the view to render the
// bidirectionalstreamingresultcollectionwithviewsservice.UsertypeCollection
// type before sending to the
// "BidirectionalStreamingResultCollectionWithViewsMethod" endpoint websocket
// connection.
func (s *BidirectionalStreamingResultCollectionWithViewsMethodServerStream) SetView(view string) {
	s.view = view
}
`


