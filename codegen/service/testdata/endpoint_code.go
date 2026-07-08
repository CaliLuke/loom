package testdata

const SingleEndpoint = `// Endpoints wraps the "SingleEndpoint" service endpoints.
type Endpoints struct {
	A loom.Endpoint
}

// NewEndpoints wraps the methods of the "SingleEndpoint" service with
// endpoints.
func NewEndpoints(s Service) *Endpoints {
	return &Endpoints{
		A: NewAEndpoint(s),
	}
}

// Use applies the given middleware to all the "SingleEndpoint" service
// endpoints.
func (e *Endpoints) Use(m func(loom.Endpoint) loom.Endpoint) {
	e.A = m(e.A)
}

// NewAEndpoint returns an endpoint function that calls the method "A" of
// service "SingleEndpoint".
func NewAEndpoint(s Service) loom.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		p := req.(*AType)
		return nil, s.A(ctx, p)
	}
}
`

const UseEndpoint = `// Endpoints wraps the "UseEndpoint" service endpoints.
type Endpoints struct {
	UseEndpoint loom.Endpoint
}

// NewEndpoints wraps the methods of the "UseEndpoint" service with endpoints.
func NewEndpoints(s Service) *Endpoints {
	return &Endpoints{
		UseEndpoint: NewUseEndpointEndpoint(s),
	}
}

// Use applies the given middleware to all the "UseEndpoint" service endpoints.
func (e *Endpoints) Use(m func(loom.Endpoint) loom.Endpoint) {
	e.UseEndpoint = m(e.UseEndpoint)
}

// NewUseEndpointEndpoint returns an endpoint function that calls the method
// "Use" of service "UseEndpoint".
func NewUseEndpointEndpoint(s Service) loom.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		p := req.(string)
		return nil, s.UseEndpoint(ctx, p)
	}
}
`

const MultipleEndpoints = `// Endpoints wraps the "MultipleEndpoints" service endpoints.
type Endpoints struct {
	B loom.Endpoint
	C loom.Endpoint
}

// NewEndpoints wraps the methods of the "MultipleEndpoints" service with
// endpoints.
func NewEndpoints(s Service) *Endpoints {
	return &Endpoints{
		B: NewBEndpoint(s),
		C: NewCEndpoint(s),
	}
}

// Use applies the given middleware to all the "MultipleEndpoints" service
// endpoints.
func (e *Endpoints) Use(m func(loom.Endpoint) loom.Endpoint) {
	e.B = m(e.B)
	e.C = m(e.C)
}

// NewBEndpoint returns an endpoint function that calls the method "B" of
// service "MultipleEndpoints".
func NewBEndpoint(s Service) loom.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		p := req.(*BType)
		return nil, s.B(ctx, p)
	}
}

// NewCEndpoint returns an endpoint function that calls the method "C" of
// service "MultipleEndpoints".
func NewCEndpoint(s Service) loom.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		p := req.(*CType)
		return nil, s.C(ctx, p)
	}
}
`

const NoPayloadEndpoint = `// Endpoints wraps the "NoPayload" service endpoints.
type Endpoints struct {
	NoPayload loom.Endpoint
}

// NewEndpoints wraps the methods of the "NoPayload" service with endpoints.
func NewEndpoints(s Service) *Endpoints {
	return &Endpoints{
		NoPayload: NewNoPayloadEndpoint(s),
	}
}

// Use applies the given middleware to all the "NoPayload" service endpoints.
func (e *Endpoints) Use(m func(loom.Endpoint) loom.Endpoint) {
	e.NoPayload = m(e.NoPayload)
}

// NewNoPayloadEndpoint returns an endpoint function that calls the method
// "NoPayload" of service "NoPayload".
func NewNoPayloadEndpoint(s Service) loom.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		return nil, s.NoPayload(ctx)
	}
}
`

const WithResultEndpoint = `// Endpoints wraps the "WithResult" service endpoints.
type Endpoints struct {
	A loom.Endpoint
}

// NewEndpoints wraps the methods of the "WithResult" service with endpoints.
func NewEndpoints(s Service) *Endpoints {
	return &Endpoints{
		A: NewAEndpoint(s),
	}
}

// Use applies the given middleware to all the "WithResult" service endpoints.
func (e *Endpoints) Use(m func(loom.Endpoint) loom.Endpoint) {
	e.A = m(e.A)
}

// NewAEndpoint returns an endpoint function that calls the method "A" of
// service "WithResult".
func NewAEndpoint(s Service) loom.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		res, err := s.A(ctx)
		if err != nil {
			return nil, err
		}
		vres, err := NewViewedRtype(res, "default")
		if err != nil {
			return nil, err
		}
		return vres, nil
	}
}
`

const WithResultMultipleViewsEndpoint = `// Endpoints wraps the "WithResultMultipleViews" service endpoints.
type Endpoints struct {
	A loom.Endpoint
	B loom.Endpoint
}

// NewEndpoints wraps the methods of the "WithResultMultipleViews" service with
// endpoints.
func NewEndpoints(s Service) *Endpoints {
	return &Endpoints{
		A: NewAEndpoint(s),
		B: NewBEndpoint(s),
	}
}

// Use applies the given middleware to all the "WithResultMultipleViews"
// service endpoints.
func (e *Endpoints) Use(m func(loom.Endpoint) loom.Endpoint) {
	e.A = m(e.A)
	e.B = m(e.B)
}

// NewAEndpoint returns an endpoint function that calls the method "A" of
// service "WithResultMultipleViews".
func NewAEndpoint(s Service) loom.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		res, err := s.A(ctx)
		if err != nil {
			return nil, err
		}
		vres, err := NewViewedViewtype(res, "tiny")
		if err != nil {
			return nil, err
		}
		return vres, nil
	}
}

// NewBEndpoint returns an endpoint function that calls the method "B" of
// service "WithResultMultipleViews".
func NewBEndpoint(s Service) loom.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		res, err := s.B(ctx)
		if err != nil {
			return nil, err
		}
		vres, err := NewViewedViewtype(res, "default")
		if err != nil {
			return nil, err
		}
		return vres, nil
	}
}
`

const StreamingResultMethodEndpoint = `// Endpoints wraps the "StreamingResultEndpoint" service endpoints.
type Endpoints struct {
	StreamingResultMethod loom.Endpoint
}

// StreamingResultMethodEndpointInput holds both the payload and the server
// stream of the "StreamingResultMethod" method.
type StreamingResultMethodEndpointInput struct {
	// Payload is the method payload.
	Payload *AType
	// Stream is the server stream used by the "StreamingResultMethod" method to
	// send data.
	Stream StreamingResultMethodServerStream
}

// NewEndpoints wraps the methods of the "StreamingResultEndpoint" service with
// endpoints.
func NewEndpoints(s Service) *Endpoints {
	return &Endpoints{
		StreamingResultMethod: NewStreamingResultMethodEndpoint(s),
	}
}

// Use applies the given middleware to all the "StreamingResultEndpoint"
// service endpoints.
func (e *Endpoints) Use(m func(loom.Endpoint) loom.Endpoint) {
	e.StreamingResultMethod = m(e.StreamingResultMethod)
}

// NewStreamingResultMethodEndpoint returns an endpoint function that calls the
// method "StreamingResultMethod" of service "StreamingResultEndpoint".
func NewStreamingResultMethodEndpoint(s Service) loom.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		ep := req.(*StreamingResultMethodEndpointInput)
		return nil, s.StreamingResultMethod(ctx, ep.Payload, ep.Stream)
	}
}
`

const MixedResultsMethodEndpoint = `// Endpoints wraps the "MixedResultsEndpoint" service endpoints.
type Endpoints struct {
	MixedResultsMethod loom.Endpoint
}

// MixedResultsMethodEndpointInput holds both the payload and the server stream
// of the "MixedResultsMethod" method.
type MixedResultsMethodEndpointInput struct {
	// Payload is the method payload.
	Payload *Payload
	// Stream is the server stream used by the "MixedResultsMethod" method to send
	// data.
	Stream MixedResultsMethodServerStream
}

// NewEndpoints wraps the methods of the "MixedResultsEndpoint" service with
// endpoints.
func NewEndpoints(s Service) *Endpoints {
	return &Endpoints{
		MixedResultsMethod: NewMixedResultsMethodEndpoint(s),
	}
}

// Use applies the given middleware to all the "MixedResultsEndpoint" service
// endpoints.
func (e *Endpoints) Use(m func(loom.Endpoint) loom.Endpoint) {
	e.MixedResultsMethod = m(e.MixedResultsMethod)
}

// NewMixedResultsMethodEndpoint returns an endpoint function that calls the
// method "MixedResultsMethod" of service "MixedResultsEndpoint".
func NewMixedResultsMethodEndpoint(s Service) loom.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		ep := req.(*MixedResultsMethodEndpointInput)
		res, err := s.MixedResultsMethod(ctx, ep.Payload, ep.Stream)
		if err != nil {
			return nil, err
		}
		return res, nil
	}
}
`

const StreamingResultNoPayloadMethodEndpoint = `// Endpoints wraps the "StreamingResultNoPayloadEndpoint" service endpoints.
type Endpoints struct {
	StreamingResultNoPayloadMethod loom.Endpoint
}

// StreamingResultNoPayloadMethodEndpointInput holds both the payload and the
// server stream of the "StreamingResultNoPayloadMethod" method.
type StreamingResultNoPayloadMethodEndpointInput struct {
	// Stream is the server stream used by the "StreamingResultNoPayloadMethod"
	// method to send data.
	Stream StreamingResultNoPayloadMethodServerStream
}

// NewEndpoints wraps the methods of the "StreamingResultNoPayloadEndpoint"
// service with endpoints.
func NewEndpoints(s Service) *Endpoints {
	return &Endpoints{
		StreamingResultNoPayloadMethod: NewStreamingResultNoPayloadMethodEndpoint(s),
	}
}

// Use applies the given middleware to all the
// "StreamingResultNoPayloadEndpoint" service endpoints.
func (e *Endpoints) Use(m func(loom.Endpoint) loom.Endpoint) {
	e.StreamingResultNoPayloadMethod = m(e.StreamingResultNoPayloadMethod)
}

// NewStreamingResultNoPayloadMethodEndpoint returns an endpoint function that
// calls the method "StreamingResultNoPayloadMethod" of service
// "StreamingResultNoPayloadEndpoint".
func NewStreamingResultNoPayloadMethodEndpoint(s Service) loom.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		ep := req.(*StreamingResultNoPayloadMethodEndpointInput)
		return nil, s.StreamingResultNoPayloadMethod(ctx, ep.Stream)
	}
}
`

const StreamingResultWithViewsMethodEndpoint = `// Endpoints wraps the "StreamingResultWithViewsService" service endpoints.
type Endpoints struct {
	StreamingResultWithViewsMethod loom.Endpoint
}

// StreamingResultWithViewsMethodEndpointInput holds both the payload and the
// server stream of the "StreamingResultWithViewsMethod" method.
type StreamingResultWithViewsMethodEndpointInput struct {
	// Payload is the method payload.
	Payload string
	// Stream is the server stream used by the "StreamingResultWithViewsMethod"
	// method to send data.
	Stream StreamingResultWithViewsMethodServerStream
}

// NewEndpoints wraps the methods of the "StreamingResultWithViewsService"
// service with endpoints.
func NewEndpoints(s Service) *Endpoints {
	return &Endpoints{
		StreamingResultWithViewsMethod: NewStreamingResultWithViewsMethodEndpoint(s),
	}
}

// Use applies the given middleware to all the
// "StreamingResultWithViewsService" service endpoints.
func (e *Endpoints) Use(m func(loom.Endpoint) loom.Endpoint) {
	e.StreamingResultWithViewsMethod = m(e.StreamingResultWithViewsMethod)
}

// NewStreamingResultWithViewsMethodEndpoint returns an endpoint function that
// calls the method "StreamingResultWithViewsMethod" of service
// "StreamingResultWithViewsService".
func NewStreamingResultWithViewsMethodEndpoint(s Service) loom.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		ep := req.(*StreamingResultWithViewsMethodEndpointInput)
		return nil, s.StreamingResultWithViewsMethod(ctx, ep.Payload, ep.Stream)
	}
}
`

const StreamingPayloadMethodEndpoint = `// Endpoints wraps the "StreamingPayloadEndpoint" service endpoints.
type Endpoints struct {
	StreamingPayloadMethod loom.Endpoint
}

// StreamingPayloadMethodEndpointInput holds both the payload and the server
// stream of the "StreamingPayloadMethod" method.
type StreamingPayloadMethodEndpointInput struct {
	// Payload is the method payload.
	Payload *BType
	// Stream is the server stream used by the "StreamingPayloadMethod" method to
	// send data.
	Stream StreamingPayloadMethodServerStream
}

// NewEndpoints wraps the methods of the "StreamingPayloadEndpoint" service
// with endpoints.
func NewEndpoints(s Service) *Endpoints {
	return &Endpoints{
		StreamingPayloadMethod: NewStreamingPayloadMethodEndpoint(s),
	}
}

// Use applies the given middleware to all the "StreamingPayloadEndpoint"
// service endpoints.
func (e *Endpoints) Use(m func(loom.Endpoint) loom.Endpoint) {
	e.StreamingPayloadMethod = m(e.StreamingPayloadMethod)
}

// NewStreamingPayloadMethodEndpoint returns an endpoint function that calls
// the method "StreamingPayloadMethod" of service "StreamingPayloadEndpoint".
func NewStreamingPayloadMethodEndpoint(s Service) loom.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		ep := req.(*StreamingPayloadMethodEndpointInput)
		return nil, s.StreamingPayloadMethod(ctx, ep.Payload, ep.Stream)
	}
}
`

const StreamingPayloadNoPayloadMethodEndpoint = `// Endpoints wraps the "StreamingPayloadNoPayloadService" service endpoints.
type Endpoints struct {
	StreamingPayloadNoPayloadMethod loom.Endpoint
}

// StreamingPayloadNoPayloadMethodEndpointInput holds both the payload and the
// server stream of the "StreamingPayloadNoPayloadMethod" method.
type StreamingPayloadNoPayloadMethodEndpointInput struct {
	// Stream is the server stream used by the "StreamingPayloadNoPayloadMethod"
	// method to send data.
	Stream StreamingPayloadNoPayloadMethodServerStream
}

// NewEndpoints wraps the methods of the "StreamingPayloadNoPayloadService"
// service with endpoints.
func NewEndpoints(s Service) *Endpoints {
	return &Endpoints{
		StreamingPayloadNoPayloadMethod: NewStreamingPayloadNoPayloadMethodEndpoint(s),
	}
}

// Use applies the given middleware to all the
// "StreamingPayloadNoPayloadService" service endpoints.
func (e *Endpoints) Use(m func(loom.Endpoint) loom.Endpoint) {
	e.StreamingPayloadNoPayloadMethod = m(e.StreamingPayloadNoPayloadMethod)
}

// NewStreamingPayloadNoPayloadMethodEndpoint returns an endpoint function that
// calls the method "StreamingPayloadNoPayloadMethod" of service
// "StreamingPayloadNoPayloadService".
func NewStreamingPayloadNoPayloadMethodEndpoint(s Service) loom.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		ep := req.(*StreamingPayloadNoPayloadMethodEndpointInput)
		return nil, s.StreamingPayloadNoPayloadMethod(ctx, ep.Stream)
	}
}
`

const StreamingPayloadNoResultMethodEndpoint = `// Endpoints wraps the "StreamingPayloadNoResultService" service endpoints.
type Endpoints struct {
	StreamingPayloadNoResultMethod loom.Endpoint
}

// StreamingPayloadNoResultMethodEndpointInput holds both the payload and the
// server stream of the "StreamingPayloadNoResultMethod" method.
type StreamingPayloadNoResultMethodEndpointInput struct {
	// Stream is the server stream used by the "StreamingPayloadNoResultMethod"
	// method to send data.
	Stream StreamingPayloadNoResultMethodServerStream
}

// NewEndpoints wraps the methods of the "StreamingPayloadNoResultService"
// service with endpoints.
func NewEndpoints(s Service) *Endpoints {
	return &Endpoints{
		StreamingPayloadNoResultMethod: NewStreamingPayloadNoResultMethodEndpoint(s),
	}
}

// Use applies the given middleware to all the
// "StreamingPayloadNoResultService" service endpoints.
func (e *Endpoints) Use(m func(loom.Endpoint) loom.Endpoint) {
	e.StreamingPayloadNoResultMethod = m(e.StreamingPayloadNoResultMethod)
}

// NewStreamingPayloadNoResultMethodEndpoint returns an endpoint function that
// calls the method "StreamingPayloadNoResultMethod" of service
// "StreamingPayloadNoResultService".
func NewStreamingPayloadNoResultMethodEndpoint(s Service) loom.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		ep := req.(*StreamingPayloadNoResultMethodEndpointInput)
		return nil, s.StreamingPayloadNoResultMethod(ctx, ep.Stream)
	}
}
`

const BidirectionalStreamingMethodEndpoint = `// Endpoints wraps the "BidirectionalStreamingEndpoint" service endpoints.
type Endpoints struct {
	BidirectionalStreamingMethod loom.Endpoint
}

// BidirectionalStreamingMethodEndpointInput holds both the payload and the
// server stream of the "BidirectionalStreamingMethod" method.
type BidirectionalStreamingMethodEndpointInput struct {
	// Payload is the method payload.
	Payload *AType
	// Stream is the server stream used by the "BidirectionalStreamingMethod"
	// method to send data.
	Stream BidirectionalStreamingMethodServerStream
}

// NewEndpoints wraps the methods of the "BidirectionalStreamingEndpoint"
// service with endpoints.
func NewEndpoints(s Service) *Endpoints {
	return &Endpoints{
		BidirectionalStreamingMethod: NewBidirectionalStreamingMethodEndpoint(s),
	}
}

// Use applies the given middleware to all the "BidirectionalStreamingEndpoint"
// service endpoints.
func (e *Endpoints) Use(m func(loom.Endpoint) loom.Endpoint) {
	e.BidirectionalStreamingMethod = m(e.BidirectionalStreamingMethod)
}

// NewBidirectionalStreamingMethodEndpoint returns an endpoint function that
// calls the method "BidirectionalStreamingMethod" of service
// "BidirectionalStreamingEndpoint".
func NewBidirectionalStreamingMethodEndpoint(s Service) loom.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		ep := req.(*BidirectionalStreamingMethodEndpointInput)
		return nil, s.BidirectionalStreamingMethod(ctx, ep.Payload, ep.Stream)
	}
}
`

const BidirectionalStreamingNoPayloadMethodEndpoint = `// Endpoints wraps the "BidirectionalStreamingNoPayloadService" service
// endpoints.
type Endpoints struct {
	BidirectionalStreamingNoPayloadMethod loom.Endpoint
}

// BidirectionalStreamingNoPayloadMethodEndpointInput holds both the payload
// and the server stream of the "BidirectionalStreamingNoPayloadMethod" method.
type BidirectionalStreamingNoPayloadMethodEndpointInput struct {
	// Stream is the server stream used by the
	// "BidirectionalStreamingNoPayloadMethod" method to send data.
	Stream BidirectionalStreamingNoPayloadMethodServerStream
}

// NewEndpoints wraps the methods of the
// "BidirectionalStreamingNoPayloadService" service with endpoints.
func NewEndpoints(s Service) *Endpoints {
	return &Endpoints{
		BidirectionalStreamingNoPayloadMethod: NewBidirectionalStreamingNoPayloadMethodEndpoint(s),
	}
}

// Use applies the given middleware to all the
// "BidirectionalStreamingNoPayloadService" service endpoints.
func (e *Endpoints) Use(m func(loom.Endpoint) loom.Endpoint) {
	e.BidirectionalStreamingNoPayloadMethod = m(e.BidirectionalStreamingNoPayloadMethod)
}

// NewBidirectionalStreamingNoPayloadMethodEndpoint returns an endpoint
// function that calls the method "BidirectionalStreamingNoPayloadMethod" of
// service "BidirectionalStreamingNoPayloadService".
func NewBidirectionalStreamingNoPayloadMethodEndpoint(s Service) loom.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		ep := req.(*BidirectionalStreamingNoPayloadMethodEndpointInput)
		return nil, s.BidirectionalStreamingNoPayloadMethod(ctx, ep.Stream)
	}
}
`

var EndpointWithServerInterceptor = `// Endpoints wraps the "ServiceWithServerInterceptor" service endpoints.
type Endpoints struct {
	Method loom.Endpoint
}

// NewEndpoints wraps the methods of the "ServiceWithServerInterceptor" service
// with endpoints.
func NewEndpoints(s Service, si ServerInterceptors) *Endpoints {
	endpoints := &Endpoints{
		Method: NewMethodEndpoint(s),
	}
	endpoints.Method = WrapMethodEndpoint(endpoints.Method, si)
	return endpoints
}

// Use applies the given middleware to all the "ServiceWithServerInterceptor"
// service endpoints.
func (e *Endpoints) Use(m func(loom.Endpoint) loom.Endpoint) {
	e.Method = m(e.Method)
}

// NewMethodEndpoint returns an endpoint function that calls the method
// "Method" of service "ServiceWithServerInterceptor".
func NewMethodEndpoint(s Service) loom.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		p := req.(string)
		return s.Method(ctx, p)
	}
}
`

var EndpointWithMultipleInterceptors = `// Endpoints wraps the "ServiceWithMultipleInterceptors" service endpoints.
type Endpoints struct {
	Method loom.Endpoint
}

// NewEndpoints wraps the methods of the "ServiceWithMultipleInterceptors"
// service with endpoints.
func NewEndpoints(s Service, si ServerInterceptors) *Endpoints {
	endpoints := &Endpoints{
		Method: NewMethodEndpoint(s),
	}
	endpoints.Method = WrapMethodEndpoint(endpoints.Method, si)
	return endpoints
}

// Use applies the given middleware to all the
// "ServiceWithMultipleInterceptors" service endpoints.
func (e *Endpoints) Use(m func(loom.Endpoint) loom.Endpoint) {
	e.Method = m(e.Method)
}

// NewMethodEndpoint returns an endpoint function that calls the method
// "Method" of service "ServiceWithMultipleInterceptors".
func NewMethodEndpoint(s Service) loom.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		p := req.(string)
		return s.Method(ctx, p)
	}
}
`

var EndpointStreamingWithInterceptor = `// Endpoints wraps the "ServiceStreamingWithInterceptor" service endpoints.
type Endpoints struct {
	Method loom.Endpoint
}

// MethodEndpointInput holds both the payload and the server stream of the
// "Method" method.
type MethodEndpointInput struct {
	// Stream is the server stream used by the "Method" method to send data.
	Stream MethodServerStream
}

// NewEndpoints wraps the methods of the "ServiceStreamingWithInterceptor" service
// with endpoints.
func NewEndpoints(s Service, i ServerInterceptors) *Endpoints {
	return &Endpoints{
		Method: WrapMethodEndpoint(NewMethodEndpoint(s), i),
	}
}

// Use applies the given middleware to all the "ServiceStreamingWithInterceptor"
// service endpoints.
func (e *Endpoints) Use(m func(loom.Endpoint) loom.Endpoint) {
	e.Method = m(e.Method)
}

// NewMethodEndpoint returns an endpoint function that calls the method "Method"
// of service "ServiceStreamingWithInterceptor".
func NewMethodEndpoint(s Service) loom.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		ep := req.(*MethodEndpointInput)
		return nil, s.Method(ctx, ep.Stream)
	}
}
`
