package transportir

import "github.com/CaliLuke/loom/expr"

type (
	Service struct {
		Name        string
		Description string
		Expr        *expr.GRPCServiceExpr
		Endpoints   []*Endpoint
	}

	Endpoint struct {
		Name         string
		Description  string
		Method       *expr.MethodExpr
		Service      *Service
		Requirements []*expr.SecurityExpr
		Request      *Request
		Response     *Response
		Errors       []*Error
		Stream       *Stream
	}

	Request struct {
		Payload             *expr.AttributeExpr
		Message             *expr.AttributeExpr
		ProtoMessage        *expr.AttributeExpr
		StreamingPayload    *expr.AttributeExpr
		StreamingMessage    *expr.AttributeExpr
		ProtoStreamingInput *expr.AttributeExpr
		Metadata            *expr.MappedAttributeExpr
	}

	Response struct {
		Result       *expr.AttributeExpr
		Message      *expr.AttributeExpr
		ProtoMessage *expr.AttributeExpr
		StatusCode   int
		Description  string
		Headers      *expr.MappedAttributeExpr
		Trailers     *expr.MappedAttributeExpr
	}

	Error struct {
		Name      string
		Type      expr.DataType
		Attribute *expr.AttributeExpr
		Response  *Response
	}

	Stream struct {
		IsStreaming        bool
		IsPayloadStreaming bool
	}
)
