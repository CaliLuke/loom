package codegen

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestMessageReservations(t *testing.T) {
	expr.SetupTestDSL(t)
	orders := [][]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}}
	for _, numbered := range []bool{false, true} {
		for index, order := range orders {
			t.Run(fmt.Sprintf("numbered_%t/order_%d", numbered, index), func(t *testing.T) {
				data := &ServiceData{designMessages: make(map[string]string)}
				candidate := "ProbeRequest"
				anonymous := candidate
				if numbered {
					data.designMessages[candidate] = conflictingDesignMessage
					anonymous += "2"
				}
				allocate := []func() string{
					func() string {
						return data.endpointMessageName(candidate, &expr.AttributeExpr{Type: &expr.Object{}})
					},
					func() string {
						return data.anonymousMessageName(messageScope{name: anonymous, path: "oneof:first"})
					},
					func() string {
						return data.anonymousMessageName(messageScope{name: anonymous + "2", path: "oneof:second"})
					},
				}
				names := make([]string, len(allocate))
				for _, index := range order {
					names[index] = allocate[index]()
				}
				require.NotEqual(t, names[0], names[1], "numbered=%v order=%v", numbered, order)
				require.NotEqual(t, names[0], names[2], "numbered=%v order=%v", numbered, order)
				require.NotEqual(t, names[1], names[2], "numbered=%v order=%v", numbered, order)
				for _, index := range order {
					require.Equal(t, names[index], allocate[index](), "numbered=%v order=%v", numbered, order)
				}
			})
		}
	}
}

func TestMessageReservationsIgnoreUnusedCandidates(t *testing.T) {
	expr.SetupTestDSL(t)
	data := &ServiceData{}
	message := &expr.AttributeExpr{Type: &expr.UserTypeExpr{
		TypeName:      "Named",
		AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}},
	}}
	require.Equal(t, "UnusedRequest", data.endpointMessageName("UnusedRequest", message))
	require.Equal(t, "UnusedRequest", data.anonymousMessageName(messageScope{name: "UnusedRequest", path: "oneof:used"}))
}

func TestMessageReservationsProto(t *testing.T) {
	root := RunGRPCDSL(t, testdata.MessageReservationsDSL)
	files := ProtoFiles("gen", CreateGRPCServices(root))
	require.Len(t, files, 2)
	for _, file := range files {
		code := sectionCode(t, file.AllSections()[1:]...)
		request, union := "AOrZRequest3", "AOrZRequest2"
		if strings.Contains(file.Path, "endpointfirst") {
			request, union = "AOrZRequest2", "AOrZRequest22"
		}
		require.Contains(t, code, "rpc AOrZ ("+request+") returns (AOrZResponse);")
		require.Contains(t, code, "message "+request+" {\n\tAOrZRequest detail = 1;\n}")
		require.Contains(t, code, "message "+union+" {\n\toneof field {\n\t\tA a = 1;\n\t\tZRequest2 z_request2 = 2;\n\t}\n}")
		require.Contains(t, code, "message A {\n\toptional string text = 1;\n}")
		require.Contains(t, code, "message ZRequest2 {\n\toptional sint64 count = 1;\n}")
	}
}

func TestMessageReservationsGeneratedModule(t *testing.T) {
	runGeneratedRoundTrip(t, "example.com/reservations", testdata.MessageReservationsDSL, messageReservationsHarness)
}

const messageReservationsHarness = `package roundtrip

import (
	"testing"

	"github.com/stretchr/testify/require"

	last "%[1]s/gen/anonymousfirst"
	first "%[1]s/gen/endpointfirst"
	lastclient "%[1]s/gen/grpc/anonymousfirst/client"
	lastserver "%[1]s/gen/grpc/anonymousfirst/server"
	firstclient "%[1]s/gen/grpc/endpointfirst/client"
	firstserver "%[1]s/gen/grpc/endpointfirst/server"
)

func TestEndpointFirst(t *testing.T) {
	text := "kept"
	count := 42
	payload := &first.AOrZPayload{Detail: &first.AOrZRequest{Name: &text}}
	require.Equal(t, payload, firstserver.NewAOrZPayload(firstclient.NewProtoAOrZRequest2(payload)))
	for _, nested := range []first.AOrZRequest2{
		first.NewAOrZRequest2A(&first.A{Text: &text}),
		first.NewAOrZRequest2ZRequest2(&first.ZRequest2{Count: &count}),
	} {
		value := first.NewAOrZRequest2OrStringAOrZRequest2(&nested)
		require.Equal(t, &value, firstserver.NewNPayload(firstclient.NewProtoNRequest(&value)))
	}
}

func TestAnonymousFirst(t *testing.T) {
	text := "kept"
	count := 42
	payload := &last.AOrZPayload{Detail: &last.AOrZRequest{Name: &text}}
	require.Equal(t, payload, lastserver.NewAOrZPayload(lastclient.NewProtoAOrZRequest3(payload)))
	for _, nested := range []last.AOrZRequest2{
		last.NewAOrZRequest2A(&last.A{Text: &text}),
		last.NewAOrZRequest2ZRequest2(&last.ZRequest2{Count: &count}),
	} {
		value := last.NewAOrZRequest2OrStringAOrZRequest2(&nested)
		require.Equal(t, &value, lastserver.NewNPayload(lastclient.NewProtoNRequest(&value)))
	}
}
`
