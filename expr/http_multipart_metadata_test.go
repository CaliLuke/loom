package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMultipartDerivedMetadataBindings(t *testing.T) {
	body := &AttributeExpr{Type: &Object{
		{Name: "data:upload", Attribute: &AttributeExpr{Type: Bytes, Meta: MetaExpr{"struct:tag:form": {"file"}}}},
		{Name: "filename:name", Attribute: &AttributeExpr{Type: &UserTypeExpr{TypeName: "Name", AttributeExpr: &AttributeExpr{Type: String}}}},
		{Name: "content_type:type", Attribute: &AttributeExpr{Type: String}},
	}}
	contract, err := HTTPMultipartBodyEncoding(body)
	require.NoError(t, err)
	require.Equal(t, &HTTPMultipartMetadata{File: "data:upload", Kind: HTTPMultipartFilename}, contract.Fields["filename:name"].Derived)
	require.Equal(t, &HTTPMultipartMetadata{File: "data:upload", Kind: HTTPMultipartContentType}, contract.Fields["content_type:type"].Derived)
	AsObject(body.Type).Set("other", &AttributeExpr{Type: Bytes})
	contract, err = HTTPMultipartBodyEncoding(body)
	require.NoError(t, err)
	require.Nil(t, contract.Fields["filename:name"].Derived)
}
