package codegen

import (
	"fmt"
	"strings"
)

func renderMediaMultipartEncode(variant *MediaVariantData, context, value string) string {
	return fmt.Sprintf("loomhttp.WithMediaMultipartCodec(%s, loomhttp.MediaMultipartCodec{Encode: func() (*loomhttp.MultipartForm, error) {\nbody := %s\nif body == nil {\nreturn nil, fmt.Errorf(\"multipart media body is nil\")\n}\n%s}})", context, value, renderMultipartEncode(variant.formBody, variant.Multipart))
}

func renderMediaMultipartDecode(variant *MediaVariantData, context, target string) string {
	var out strings.Builder
	fmt.Fprintf(&out, "loomhttp.WithMediaMultipartCodec(%s, loomhttp.MediaMultipartCodec{Decode: func(multipartForm *loomhttp.MultipartForm) error {\n", context)
	if len(variant.Multipart.Branches) != 0 {
		out.WriteString(renderMultipartMatcher(variant.formBody, variant.Multipart))
		fmt.Fprintf(&out, "matched, _, err := matchMultipart()\nif err != nil {\nreturn err\n}\n%s = &matched\nreturn nil\n", target)
	} else {
		fmt.Fprintf(&out, "body := new(%s)\n", variant.formBody.Name)
		out.WriteString(renderMultipartLeafDecode(variant.formBody, variant.Multipart, "body"))
		out.WriteString(renderMultipartClosedKeys(variant.Multipart))
		fmt.Fprintf(&out, "%s = body\nreturn nil\n", target)
	}
	out.WriteString("}})")
	return out.String()
}
