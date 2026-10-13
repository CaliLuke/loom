package codegen

import "github.com/CaliLuke/loom/internal/uniongen"

type constructorNames map[string]bool

// constructorReservations inventories declarations that already own names in
// the output package. Conversion functions allocate only after this inventory;
// methods have a receiver namespace and do not participate.
func (sd *ServiceData) constructorReservations(client bool) constructorNames {
	names := make(constructorNames)
	unions := sd.ServerUnionTypes
	helpers := sd.ServerTransformHelpers
	attributes := sd.ServerBodyAttributeTypes
	if client {
		names.add(sd.ClientStruct, "New"+sd.ClientStruct)
		unions = sd.ClientUnionTypes
		helpers = sd.ClientTransformHelpers
		attributes = sd.ClientBodyAttributeTypes
		names.addCLI(sd)
		for _, group := range sd.clientGroups {
			names.add(group.Name)
		}
	} else {
		names.add(sd.ServerStruct, sd.MountPointStruct, sd.ServerInit, sd.MountServer, "ResponseContractCases")
		for _, file := range sd.FileServers {
			names.add(file.MountHandler)
		}
	}
	for _, ep := range sd.Endpoints {
		names.addEndpoint(ep, client)
	}
	for _, body := range attributes {
		names.addType(body)
	}
	for _, union := range unions {
		names.addUnion(union)
	}
	for _, helper := range helpers {
		names.add(helper.Name)
	}
	return names
}

func (names constructorNames) add(values ...string) {
	for _, name := range values {
		if name != "" {
			names[name] = true
		}
	}
}

func (names constructorNames) addType(body *TypeData) {
	if body == nil {
		return
	}
	if body.Def != "" {
		names.add(body.VarName)
	}
	if body.ValidateDef != "" {
		names.add(body.validationFunction.Name)
	}
}

func (names constructorNames) addUnion(union *uniongen.Type) {
	names.add(union.Name, union.KindName)
	for _, field := range union.Fields {
		names.add(field.KindConst, "New"+union.Name+field.FieldName)
		if field.EmitPrimitiveAlias {
			names.add(field.FieldType)
		}
	}
}

func (names constructorNames) addCLI(sd *ServiceData) {
	for _, input := range sd.cliBodyInputs {
		names.addType(input.root)
		for _, body := range input.types {
			names.addType(body)
		}
		for _, union := range input.unions {
			names.addUnion(union)
		}
		for _, helper := range input.helpers {
			names.add(helper.Name)
		}
	}
}

func (names constructorNames) addEndpoint(ep *EndpointData, client bool) {
	for _, route := range ep.Routes {
		names.add(route.PathInit.Name)
	}
	if !ep.Method.IsJSONRPC && IsWebSocketEndpoint(ep) {
		names.add("ConnConfigurer", "NewConnConfigurer")
	}
	if client {
		names.addClient(ep)
	} else {
		names.add(ep.HandlerInit, ep.MountHandler, ep.RequestDecoder, ep.ResponseEncoder, ep.ErrorEncoder, ep.ResponseContractCasesInit)
		names.addType(ep.Payload.Request.ServerBody)
		if ws := ep.ServerWebSocket; ws != nil {
			names.add(ws.VarName)
			names.addType(ws.Payload)
		}
		if sse := ep.SSE; sse != nil {
			names.add(sse.StructName)
		}
		if multipart := ep.MultipartRequestDecoder; multipart != nil {
			names.add(multipart.FuncName, multipart.InitName)
		}
	}
	walkResponses(ep, func(response *ResponseData) {
		if client {
			names.addType(response.ClientBody)
		} else {
			for _, body := range response.ServerBody {
				names.addType(body)
			}
		}
	})
}

func (names constructorNames) addClient(ep *EndpointData) {
	names.add(ep.RequestEncoder, ep.ResponseDecoder, ep.BuildStreamPayload)
	if ep.Payload.Ref != "" {
		names.add("Build" + ep.Method.VarName + "Payload")
	}
	if ep.SSE != nil {
		names.add(ep.Method.VarName + "ClientStream")
		if !ep.Method.IsJSONRPC {
			names.add(ep.Method.VarName+"StreamImpl", "New"+ep.Method.VarName+"Stream")
		}
	}
	names.addType(ep.Payload.Request.ClientBody)
	if ws := ep.ClientWebSocket; ws != nil {
		names.add(ws.VarName)
		names.addType(ws.Payload)
	}
	if multipart := ep.MultipartRequestEncoder; multipart != nil {
		names.add(multipart.FuncName, multipart.InitName)
	}
}
