package codegen

const mapUnionBranchRoundTripTests = `
func branchPtr[T any](value T) *T {
	return &value
}

func mapBranchEnvelope(index mapalias.Index) *mapalias.Envelope {
	choice := mapalias.NewChoiceIndex(index)
	return &mapalias.Envelope{
		RequiredIndex: mapalias.Index{"required": 1},
		Pick: &choice,
		Detail: branchPtr(mapalias.NewDetailCounts(mapalias.DetailCounts(index))),
		Nested: branchPtr(mapalias.NewChoiceOrLeafChoice(&choice)),
	}
}

func TestMapBranchesThroughProtobuf(t *testing.T) {
	leaf := &mapalias.Leaf{Name: branchPtr("leaf")}
	for _, test := range []struct {
		name string
		input *mapalias.Envelope
		want *mapalias.Envelope
	}{
		{"nil selected map", mapBranchEnvelope(nil), mapBranchEnvelope(mapalias.Index{})},
		{"empty selected map", mapBranchEnvelope(mapalias.Index{}), mapBranchEnvelope(mapalias.Index{})},
		{"populated selected map", mapBranchEnvelope(mapalias.Index{"a": 2, "b": 3}), mapBranchEnvelope(mapalias.Index{"a": 2, "b": 3})},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := client.NewProtoEchoRequest(test.input)
			require.NotNil(t, request.GetPickIndex(), "a selected map must have a wrapper even when empty")
			wire, err := proto.Marshal(request)
			require.NoError(t, err)
			var decodedRequest pb.EchoRequest
			require.NoError(t, proto.Unmarshal(wire, &decodedRequest))
			require.NotNil(t, decodedRequest.GetPickIndex())
			require.NoError(t, server.ValidateEchoRequest(&decodedRequest))
			got := server.NewEchoPayload(&decodedRequest)
			require.Equal(t, mapalias.ChoiceKindIndex, got.Pick.Kind())
			require.Equal(t, test.want, got)
			response := server.NewProtoEchoResponse(test.input)
			wire, err = proto.Marshal(response)
			require.NoError(t, err)
			var decodedResponse pb.EchoResponse
			require.NoError(t, proto.Unmarshal(wire, &decodedResponse))
			require.Equal(t, test.want, client.NewEchoResult(&decodedResponse))
		})
	}
	for _, test := range []struct {
		name string
		value *mapalias.Envelope
	}{
		{"unset", &mapalias.Envelope{RequiredIndex: mapalias.Index{"r": 1}}},
		{"other branch", &mapalias.Envelope{RequiredIndex: mapalias.Index{"r": 1}, Pick: branchPtr(mapalias.NewChoiceLeaf(leaf))}},
		{"object map", &mapalias.Envelope{RequiredIndex: mapalias.Index{"r": 1}, Maps: branchPtr(mapalias.NewLeafIndexOrTagIndexLeafIndex(mapalias.LeafIndex{"a": leaf}))}},
		{"array map", &mapalias.Envelope{RequiredIndex: mapalias.Index{"r": 1}, Maps: branchPtr(mapalias.NewLeafIndexOrTagIndexTagIndex(mapalias.TagIndex{2: {"x", "y"}}))}},
		{"map alias", &mapalias.Envelope{RequiredIndex: mapalias.Index{"r": 1}, Detail: branchPtr(mapalias.NewDetailMore(mapalias.More{"a": 7}))}},
		{"validated map", &mapalias.Envelope{RequiredIndex: mapalias.Index{"r": 1}, Detail: branchPtr(mapalias.NewDetailLimited(mapalias.Limited{"a": "ok"}))}},
	} {
		t.Run(test.name, func(t *testing.T) {
			wire, err := proto.Marshal(client.NewProtoEchoRequest(test.value))
			require.NoError(t, err)
			var request pb.EchoRequest
			require.NoError(t, proto.Unmarshal(wire, &request))
			require.NoError(t, server.ValidateEchoRequest(&request))
			require.Equal(t, test.value, server.NewEchoPayload(&request))
			wire, err = proto.Marshal(server.NewProtoEchoResponse(test.value))
			require.NoError(t, err)
			var response pb.EchoResponse
			require.NoError(t, proto.Unmarshal(wire, &response))
			require.Equal(t, test.value, client.NewEchoResult(&response))
		})
	}
}

func TestDirectMapUnionThroughProtobuf(t *testing.T) {
	for _, index := range []mapalias.Index{nil, {}, {"a": 3}} {
		choice := mapalias.NewChoiceIndex(index)
		for _, message := range []*pb.Choice{client.NewProtoChoice(&choice), server.NewProtoChoice(&choice)} {
			require.NotNil(t, message.GetIndex())
			wire, err := proto.Marshal(message)
			require.NoError(t, err)
			var decoded pb.Choice
			require.NoError(t, proto.Unmarshal(wire, &decoded))
			for _, got := range []*mapalias.Choice{server.NewChoosePayload(&decoded), client.NewChooseResult(&decoded)} {
				require.Equal(t, mapalias.ChoiceKindIndex, got.Kind())
				values, ok := got.AsIndex()
				require.True(t, ok)
				require.Len(t, values, len(index))
				for key, value := range index {
					require.Equal(t, value, values[key])
				}
			}
		}
	}
}

func TestMapBranchValidation(t *testing.T) {
	for _, value := range []mapalias.Limited{{"a": ""}, {"a": "x", "b": "y", "c": "z"}} {
		envelope := &mapalias.Envelope{RequiredIndex: mapalias.Index{"r": 1}, Detail: branchPtr(mapalias.NewDetailLimited(value))}
		require.Error(t, server.ValidateEchoRequest(client.NewProtoEchoRequest(envelope)))
	}
}
`
