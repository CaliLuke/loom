package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

func TestJSONFormCarrierRetainsConstraints(t *testing.T) {
	value := &expr.UserTypeExpr{TypeName: "Address", AttributeExpr: &expr.AttributeExpr{
		Type:       &expr.Object{{Name: "street", Attribute: &expr.AttributeExpr{Type: expr.String}}},
		Validation: &expr.ValidationExpr{Required: []string{"street"}},
	}}
	field := &expr.AttributeExpr{Type: value}
	request := &transportir.Request{FormEncoded: true, Body: &expr.AttributeExpr{Type: &expr.Object{{Name: "address", Attribute: field}}}, Form: &expr.HTTPFormEncoding{Fields: map[string]httpencoding.FieldCodec{"address": {Name: "address", Codec: httpencoding.Codec{Style: httpencoding.JSON}}}}}
	nameFormFieldCarriers([]*transportir.Endpoint{{Name: "submit", Request: request, Response: &transportir.Response{}}})
	require.NotSame(t, value, field.Type)
	require.Equal(t, []string{"street"}, selectedTransportConstraints(field).Validation().Lowered().Required)
	require.Equal(t, "Address", value.TypeName)
}

func TestJSONFormCarrierInRootAlternatives(t *testing.T) {
	address := &expr.UserTypeExpr{TypeName: "Address", AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Object{{Name: "street", Attribute: &expr.AttributeExpr{Type: expr.String}}},
	}}
	jsonField, textField := &expr.AttributeExpr{Type: address}, &expr.AttributeExpr{Type: address}
	body := &expr.AttributeExpr{Type: &expr.Union{TypeName: "AddressChoice", Untagged: true, Values: []*expr.NamedAttributeExpr{
		{Name: "JSON", Attribute: &expr.AttributeExpr{Type: &expr.Object{{Name: "address", Attribute: jsonField}}, Meta: expr.MetaExpr{
			"openapi:encoding:address:contentType": {"application/json"},
		}}},
		{Name: "Text", Attribute: &expr.AttributeExpr{Type: &expr.Object{{Name: "address", Attribute: textField}}, Meta: expr.MetaExpr{
			"openapi:encoding:address:style": {"deepObject"},
		}}},
	}}}
	contract, err := expr.HTTPFormBodyEncoding(body)
	require.NoError(t, err)
	request := &transportir.Request{FormEncoded: true, Body: body, Form: contract}
	nameFormFieldCarriers([]*transportir.Endpoint{{Name: "submit", Request: request, Response: &transportir.Response{}}})
	require.NotSame(t, address, jsonField.Type)
	require.NotSame(t, address, textField.Type, "styled sibling owns its independent native carrier")
	require.Equal(t, "Address", address.TypeName)
}

func TestJSONFormContentBranchCarriers(t *testing.T) {
	for _, multipart := range []bool{false, true} {
		record := &expr.UserTypeExpr{TypeName: "Record", AttributeExpr: &expr.AttributeExpr{
			Type:       &expr.Object{{Name: "count", Attribute: &expr.AttributeExpr{Type: expr.Int}}},
			Validation: &expr.ValidationExpr{Required: []string{"count"}},
		}}
		choice := &expr.UserTypeExpr{TypeName: "Choice", AttributeExpr: &expr.AttributeExpr{
			Type: &expr.Union{TypeName: "Choice", Untagged: true, Values: []*expr.NamedAttributeExpr{
				{Name: "Text", Attribute: &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Words", AttributeExpr: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}}}}},
				{Name: "Records", Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: record}}}},
			}},
		}}
		field := &expr.AttributeExpr{Type: choice, Meta: expr.MetaExpr{"http:encoding:sourceType": {"array"}}}
		body := &expr.AttributeExpr{Type: &expr.Object{{Name: "values", Attribute: field}}}
		contract, err := expr.HTTPFormBodyEncoding(body)
		require.NoError(t, err)
		require.Equal(t, httpencoding.Content, contract.Fields["values"].Style)
		request := &transportir.Request{FormEncoded: true, Body: body, Form: contract}
		if multipart {
			parts, err := expr.HTTPMultipartBodyEncoding(body)
			require.NoError(t, err)
			request.FormEncoded, request.Multipart = false, true
			request.Form, request.MultipartEncoding = nil, parts
		}
		endpoint := &transportir.Endpoint{Name: "submit", Request: request, Response: &transportir.Response{}}
		nameFormFieldCarriers([]*transportir.Endpoint{endpoint})
		require.NotSame(t, choice, field.Type, "private mixed encoding declaration cannot change other occurrences")
		branches := expr.AsUnion(field.Type).Values
		jsonArray, named := branches[1].Attribute.Type.(expr.UserType)
		require.True(t, named, "JSON item arrays need an independent element-layout boundary")
		layout := representation.NewLayout()
		layout.RecordAll(body, true, true, false)
		recordFormFieldCarriers(request.Body, formCarriers(endpoint)[0].contract, layout)
		require.False(t, layout.JSONPresence[field.Type.(expr.UserType).Hash()], "outer form union retains native presence")
		require.True(t, layout.JSONPresence[jsonArray.Hash()], "JSON items retain collection element presence")
		textArray, named := branches[0].Attribute.Type.(expr.UserType)
		require.True(t, named)
		require.False(t, layout.JSONPresence[textArray.Hash()], "native text decoder cannot consume JSON Optional elements")
		jsonRecord := expr.AsArray(jsonArray).ElemType.Type.(expr.UserType)
		require.True(t, layout.JSONPresence[jsonRecord.Hash()], "item object fields retain JSON absence")
		require.Equal(t, "Record", record.Name())
		require.IsType(t, &expr.Array{}, expr.AsUnion(choice).Values[1].Attribute.Type)
	}
}

func TestFormCarrierBodyDeclarationOwnership(t *testing.T) {
	root := &expr.UserTypeExpr{TypeName: "SharedBody", UID: "shared", AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Object{{Name: "values", Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}}}},
	}}
	endpoints := make([]*transportir.Endpoint, 0, 2)
	for _, name := range []string{"first", "second"} {
		body := expr.DupAtt(&expr.AttributeExpr{Type: root})
		contract, err := expr.HTTPFormBodyEncoding(body)
		require.NoError(t, err)
		endpoints = append(endpoints, &transportir.Endpoint{Name: name,
			Request: &transportir.Request{Body: body, FormEncoded: true, Form: contract}, Response: &transportir.Response{},
		})
	}
	nameFormFieldCarriers(endpoints)
	first, second := endpoints[0].Request.Body, endpoints[1].Request.Body
	require.NotEqual(t, first.Type.Name(), second.Type.Name(), "projected fields require independent owning body declarations")
	require.NotEqual(t, expr.AsObject(first.Type).Attribute("values").Type.Name(), expr.AsObject(second.Type).Attribute("values").Type.Name())
	require.Equal(t, "SharedBody", root.Name(), "transport ownership must not rename service source")
}

func TestFormCarriersCoverRequestAndResponseMedia(t *testing.T) {
	for _, response := range []bool{false, true} {
		name := "request"
		if response {
			name = "response"
		}
		t.Run(name, func(t *testing.T) {
			body := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "MediaBody", AttributeExpr: &expr.AttributeExpr{
				Type: &expr.Object{{Name: "words", Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}}}},
			}}}
			contract, err := expr.HTTPFormBodyEncoding(body)
			require.NoError(t, err)
			variant := &transportir.MediaVariant{Branch: "Form", Body: body, Form: contract}
			endpoint := &transportir.Endpoint{Name: "submit", Request: &transportir.Request{}, Response: &transportir.Response{}}
			if response {
				endpoint.Response.Responses = []*transportir.ResponseStatus{{Body: body, MediaVariants: []*transportir.MediaVariant{variant}}}
			} else {
				endpoint.Request.Body, endpoint.Request.MediaVariants = body, []*transportir.MediaVariant{variant}
			}
			nameFormFieldCarriers([]*transportir.Endpoint{endpoint})
			var data ServiceData
			recordServiceTypeLayouts([]*transportir.Endpoint{endpoint}, &data)
			presence := data.ServerJSONPresenceTypes
			if response {
				presence = data.ClientJSONPresenceTypes
			}
			words := expr.AsObject(body.Type).Attribute("words").Type.(expr.UserType)
			require.False(t, presence[words.Hash()], "ambient JSON media layout must not wrap text array elements")
		})
	}
}

func TestFormCarrierRetainsAliasedSourceType(t *testing.T) {
	for _, source := range []string{"array", "absent"} {
		t.Run(source, func(t *testing.T) {
			declaration := &expr.UserTypeExpr{TypeName: "Words", AttributeExpr: &expr.AttributeExpr{
				Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}},
				Meta: expr.MetaExpr{"http:encoding:sourceType": {source}},
			}}
			field := &expr.AttributeExpr{Type: declaration}
			body := &expr.AttributeExpr{Type: &expr.Object{{Name: "words", Attribute: field}}, Meta: expr.MetaExpr{"openapi:encoding:words:contentType": {"application/json"}}}
			contract, err := expr.HTTPFormBodyEncoding(body)
			require.NoError(t, err)
			request := &transportir.Request{Body: body, FormEncoded: true, Form: contract}
			nameFormFieldCarriers([]*transportir.Endpoint{{Name: "submit", Request: request, Response: &transportir.Response{}}})
			require.Equal(t, source, expr.HTTPValueShape(field).SourceType)
			promoted, err := expr.HTTPFormBodyEncoding(body)
			require.NoError(t, err)
			require.Equal(t, contract.Fields["words"].Application, promoted.Fields["words"].Application)
			require.Equal(t, []string{source}, declaration.AttributeExpr.Meta["http:encoding:sourceType"])
		})
	}
}

func TestMultipartFieldCarriersKeepNativeTextItems(t *testing.T) {
	array := &expr.UserTypeExpr{TypeName: "Words", AttributeExpr: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}}}
	field := &expr.AttributeExpr{Type: array}
	body := &expr.AttributeExpr{Type: &expr.Object{{Name: "words", Attribute: field}}}
	contract, err := expr.HTTPMultipartBodyEncoding(body)
	require.NoError(t, err)
	request := &transportir.Request{Multipart: true, Body: body, MultipartEncoding: contract}
	endpoint := &transportir.Endpoint{Name: "submit", Request: request, Response: &transportir.Response{}}
	nameFormFieldCarriers([]*transportir.Endpoint{endpoint})
	require.NotSame(t, array, field.Type, "multipart text items need an independent native carrier")
	require.Equal(t, "Words", array.Name())
}
