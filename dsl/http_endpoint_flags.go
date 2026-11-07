package dsl

import (
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

// MultipartRequest indicates that HTTP requests made to the method use
// MIME multipart encoding as defined in RFC 2046.
//
// MultipartRequest must appear in a HTTP endpoint expression.
//
// Loom generates framework-owned request decoding for supported object payloads
// so applications do not need handwritten multipart decoder hooks for common
// file-and-fields uploads. Multipart request encoding still accepts a user
// provided writer function for now, and unsupported multipart payload shapes
// continue to use the existing custom encoder/decoder seam.
func MultipartRequest() {
	e, ok := eval.Current().(*expr.HTTPEndpointExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	e.MultipartRequest = true
}

// FormRequest indicates that HTTP requests made to the method use
// application/x-www-form-urlencoded encoding.
//
// FormRequest must appear in a HTTP endpoint expression.
//
// Loom generates framework-owned request decoding and encoding for supported
// typed payloads so applications do not need handwritten form parsers.
func FormRequest() {
	e, ok := eval.Current().(*expr.HTTPEndpointExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	e.FormRequest = true
}

// OptionalRequestBody indicates that HTTP requests made to the method may omit
// the request body entirely or provide a typed body. The body uses the
// endpoint's request encoding: JSON by default, or the generated FormRequest or
// MultipartRequest contract.
//
// OptionalRequestBody must appear in a HTTP endpoint expression.
//
// The body must be an object, map, or union that the selected encoding
// supports. A payload attribute selected with Body must be optional. JSON and
// form object bodies built from payload attributes cannot require members;
// generated multipart objects may, and those members apply only when a body is
// supplied. Custom multipart codecs and SkipRequestBodyEncodeDecode cannot be
// combined with OptionalRequestBody.
//
// Generated servers treat an empty or JSON-whitespace-only JSON body and a
// zero-byte form or multipart body as absent. Supplied bodies, including empty
// form value sets and empty multipart envelopes, are decoded and validated
// normally, and malformed input remains an error. Required bodies retain their
// strict behavior.
func OptionalRequestBody() {
	e, ok := eval.Current().(*expr.HTTPEndpointExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	e.OptionalRequestBody = true
}

// SkipRequestBodyEncodeDecode prevents Loom from generating the request encoding
// (client) and decoding (server) code. Instead the service method gets direct
// access to the HTTP body reader. The client method provides a reader from
// which to stream the request body. This makes it possible to stream requests
// without requiring the entire content to be loaded in memory for
// encoding/decoding. Note that the use of this function is incompatible with
// gRPC and calling it on a method that defines a gRPC transport is an error.
//
// SkipRequestBodyEncodeDecode must appear in a HTTP endpoint expression.
//
// Example:
//
//	var _ = Service("upload", func() {
//	    Method("upload", func() {
//	        Payload(func() {
//	            Attribute("id", String)
//	            Attribute("length", Int)
//	        })
//	        HTTP(func() {
//	            POST("/{id}")
//	            Header("length:Content-Length")
//	            SkipRequestBodyEncodeDecode()
//	        })
//	    })
func SkipRequestBodyEncodeDecode() {
	e, ok := eval.Current().(*expr.HTTPEndpointExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	e.SkipRequestBodyEncodeDecode = true
}

// SkipResponseBodyEncodeDecode prevents Loom from generating the response
// encoding (server) and decoding (client) code. Instead the service method
// returns a reader from which to stream the HTTP response body io. The client
// also gets access to a reader to stream the incoming response body. This makes
// it possible to stream responses without requiring the entire content to be
// loaded in memory for encoding/decoding. Note that the use of this function is
// incompatible with gRPC and calling it on a method that defines a gRPC
// transport is an error.
//
// SkipResponseBodyEncodeDecode must appear in a HTTP endpoint expression.
//
// Example:
//
//	var _ = Service("download", func() {
//	    Method("download", func() {
//	        Payload(String)
//	        Result(func() {
//	            Attribute("length", Int)
//	        })
//	        HTTP(func() {
//	            POST("/{id}")
//	            SkipResponseBodyEncodeDecode()
//	            Response(StatusOK, func() {
//	                Header("length:Content-Length")
//	            })
//	        })
//	    })
func SkipResponseBodyEncodeDecode() {
	e, ok := eval.Current().(*expr.HTTPEndpointExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	e.SkipResponseBodyEncodeDecode = true
}

// FileResponse indicates that the endpoint returns seekable file or media
// content using the standard library HTTP content-serving semantics. The
// service method returns a loom HTTP FileResponse alongside its typed result.
// Loom delegates range and conditional request handling to http.ServeContent.
//
// FileResponse must appear in an HTTP endpoint expression whose explicitly
// declared routes use only GET or HEAD. FileResponse does not add a HEAD route.
func FileResponse() {
	e, ok := eval.Current().(*expr.HTTPEndpointExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	e.FileResponse = true
}
