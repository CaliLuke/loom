package expr

import (
	"mime"
	"strings"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/internal/httpcodec"
)

// HTTPMediaTypeExpr binds one HTTP media contract to a semantic body union branch.
// The endpoint or response owns the mapping; reusable service unions are unchanged.
type HTTPMediaTypeExpr struct {
	// ContentType is the authored media type or range, including its parameters.
	ContentType string
	// Branch names a branch of the endpoint or response's selected Body union.
	Branch string
}

// HTTPMediaBranch resolves a mapping against its effective body union. It returns
// nil when the body is not a union or the branch does not exist.
func HTTPMediaBranch(body *AttributeExpr, name string) *NamedAttributeExpr {
	if body == nil {
		return nil
	}
	union := AsUnion(body.Type)
	if union != nil {
		for _, branch := range union.Values {
			if AttributeName(branch.Name) == name {
				return branch
			}
		}
	}
	return nil
}

func (e *HTTPEndpointExpr) validateMediaTypes(verr *eval.ValidationErrors) {
	if len(e.MediaTypes) == 0 {
		return
	}
	_, rpc := e.MethodExpr.Meta["jsonrpc"]
	if e.RequestContentType != nil || e.OpenAPIRequestBody != nil || e.FormRequest ||
		e.MultipartRequest || e.SkipRequestBodyEncodeDecode || e.MethodExpr.IsStreaming() || rpc {
		verr.Add(e, "MediaType requires an ordinary typed HTTP body without ContentType, documentation-only body, form, multipart, raw or streaming framing")
	}
	validateHTTPMediaTypes(e, httpRequestBody(e), e.MethodExpr.Payload, e.MediaTypes, e.Headers, verr)
}

func (r *HTTPResponseExpr) validateMediaTypes(e *HTTPEndpointExpr, source *AttributeExpr, verr *eval.ValidationErrors) {
	if len(r.MediaTypes) == 0 {
		return
	}
	_, rpc := e.MethodExpr.Meta["jsonrpc"]
	if r.ContentType != "" || r.OpenAPIBody != nil || e.SkipResponseBodyEncodeDecode ||
		e.FileResponse || e.MethodExpr.IsStreaming() || rpc {
		verr.Add(r, "MediaType requires an ordinary typed HTTP body without ContentType, documentation-only body, raw or streaming framing")
	}
	validateHTTPMediaTypes(r, buildHTTPResponseBody(e.Name(), source, r, e.Service), source, r.MediaTypes, r.Headers, verr)
}

func validateHTTPMediaTypes(parent eval.Expression, body, source *AttributeExpr, mappings []*HTTPMediaTypeExpr, headers *MappedAttributeExpr, verr *eval.ValidationErrors) {
	union := AsUnion(body.Type)
	if union == nil || IsNullable(body) {
		verr.Add(parent, "MediaType requires a non-nullable union Body")
		return
	}
	header := mediaContentTypeHeader(headers, source)
	if header != nil && (httpMediaBaseType(header.Type) != String || AllowsNull(header)) {
		verr.Add(parent, "MediaType mapped Content-Type header requires a non-nullable String")
	}
	media := make([]string, len(mappings))
	branches := make(map[string]bool, len(mappings))
	for i, mapping := range mappings {
		media[i] = mapping.ContentType
		branch := HTTPMediaBranch(body, mapping.Branch)
		if branch == nil {
			verr.Add(parent, "MediaType references unknown body branch %q", mapping.Branch)
			continue
		}
		if branches[branch.Name] {
			verr.Add(parent, "MediaType maps body branch %q more than once", mapping.Branch)
		}
		branches[branch.Name] = true
		base, _, mediaErr := mime.ParseMediaType(mapping.ContentType)
		if mediaErr == nil && base == "application/octet-stream" &&
			(httpMediaBaseType(branch.Attribute.Type) != Bytes || AllowsNull(branch.Attribute)) {
			verr.Add(parent, "MediaType %q requires a non-nullable Bytes branch", mapping.ContentType)
		}
		if err := validateMediaFields(branch.Attribute, mapping.ContentType); err != nil {
			verr.Add(parent, "%s", err)
		}
		if strings.Contains(base, "*") && header == nil {
			verr.Add(parent, "MediaType range %q requires a mapped Content-Type header for concrete publication", mapping.ContentType)
		}
		if httpcodec.RequestContentType(mapping.ContentType).Kind == httpcodec.Text &&
			(!httpMediaTextBranch(branch.Attribute.Type) || AllowsNull(branch.Attribute)) {
			verr.Add(parent, "MediaType %q requires a non-nullable String or Bytes branch", mapping.ContentType)
		}
	}
	for _, branch := range union.Values {
		if !branches[branch.Name] {
			verr.Add(parent, "MediaType has no mapping for body branch %q", branch.Name)
		}
	}
	if err := httpcodec.ValidateMediaContracts(media); err != nil {
		verr.Add(parent, "%s", err)
	}
}

func httpMediaTextBranch(typ DataType) bool {
	for {
		named, ok := typ.(UserType)
		if !ok {
			return typ == String || typ == Bytes
		}
		typ = named.Attribute().Type
	}
}

func mediaContentTypeHeader(headers *MappedAttributeExpr, source *AttributeExpr) *AttributeExpr {
	if headers == nil || headers.IsEmpty() {
		return nil
	}
	for _, field := range *AsObject(headers.Type) {
		if strings.EqualFold(headers.ElemName(field.Name), "Content-Type") {
			if source != nil && IsObject(source.Type) {
				if _, attribute := source.FindAttribute(field.Name); attribute != nil {
					return attribute
				}
			}
			return field.Attribute
		}
	}
	return nil
}
