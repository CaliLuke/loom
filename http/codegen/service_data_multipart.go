package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func (sds *ServicesData) initEndpointMultipartData(endpoint *EndpointData, endpointIR *transportir.Endpoint, method *service.MethodData, svc *service.Data) {
	if endpointIR.Request.Multipart && !endpoint.Payload.Request.MultipartGenerated {
		endpoint.MultipartRequestDecoder = &MultipartData{
			FuncName:    fmt.Sprintf("%s%sDecoderFunc", svc.StructName, method.VarName),
			InitName:    fmt.Sprintf("New%s%sDecoder", svc.StructName, method.VarName),
			VarName:     fmt.Sprintf("%s%sDecoderFn", svc.VarName, method.VarName),
			ServiceName: svc.Name,
			MethodName:  method.Name,
			Payload:     endpoint.Payload,
		}
	}
	if endpointIR.Request.Multipart {
		endpoint.MultipartRequestEncoder = &MultipartData{
			FuncName:    fmt.Sprintf("%s%sEncoderFunc", svc.StructName, method.VarName),
			InitName:    fmt.Sprintf("New%s%sEncoder", svc.StructName, method.VarName),
			VarName:     fmt.Sprintf("%s%sEncoderFn", svc.VarName, method.VarName),
			ServiceName: svc.Name,
			MethodName:  method.Name,
			Payload:     endpoint.Payload,
		}
	}
}

func generatedMultipartRequestData(request *transportir.Request) (bool, []*MultipartFileFieldData) {
	if request == nil || !request.Multipart || request.Body == nil || request.Body.Type == expr.Empty || !expr.IsObject(request.Body.Type) {
		return false, nil
	}
	if !supportsGeneratedMultipartObject(request.Body) {
		return false, nil
	}
	fileFields := multipartFileFields(request.Body)
	if len(fileFields) == 1 {
		bodyObj := expr.AsObject(request.Body.Type)
		if attr := bodyObj.Attribute("filename"); attr != nil && attr.Type.Kind() == expr.StringKind {
			fileFields[0].PopulateFilename = true
		}
		if attr := bodyObj.Attribute("content_type"); attr != nil && attr.Type.Kind() == expr.StringKind {
			fileFields[0].PopulateContentType = true
		}
	}
	return true, fileFields
}

func supportsGeneratedMultipartObject(body *expr.AttributeExpr) bool {
	obj := expr.AsObject(body.Type)
	if obj == nil {
		return false
	}
	visiting := make(map[string]struct{})
	for _, nat := range *obj {
		if nat.Attribute.Type == expr.Bytes {
			continue
		}
		if !supportsGeneratedMultipartNested(nat.Attribute, visiting) {
			return false
		}
	}
	return true
}

// supportsGeneratedMultipartNested reports whether generated multipart code
// can encode the nested attribute att. visiting holds the IDs of the user
// types on the current path. A recursive user type is not supported.
func supportsGeneratedMultipartNested(att *expr.AttributeExpr, visiting map[string]struct{}) bool {
	switch actual := att.Type.(type) {
	case expr.Primitive:
		return actual != expr.Any && actual != expr.Bytes
	case expr.UserType:
		return supportsGeneratedMultipartUserType(actual, visiting, supportsGeneratedMultipartNested)
	case *expr.Object:
		for _, nat := range *actual {
			if !supportsGeneratedMultipartNested(nat.Attribute, visiting) {
				return false
			}
		}
		return true
	case *expr.Map:
		if !isSupportedMultipartScalar(actual.KeyType.Type) {
			return false
		}
		return supportsGeneratedMultipartNested(actual.ElemType, visiting)
	case *expr.Array:
		return supportsGeneratedMultipartCollectionElem(actual.ElemType, visiting)
	case *expr.Union:
		return false
	default:
		return false
	}
}

// supportsGeneratedMultipartCollectionElem reports whether generated multipart
// code can encode the collection element att. visiting holds the IDs of the
// user types on the current path. A recursive user type is not supported.
func supportsGeneratedMultipartCollectionElem(att *expr.AttributeExpr, visiting map[string]struct{}) bool {
	switch actual := att.Type.(type) {
	case expr.Primitive:
		return actual != expr.Any && actual != expr.Bytes
	case expr.UserType:
		return supportsGeneratedMultipartUserType(actual, visiting, supportsGeneratedMultipartCollectionElem)
	case *expr.Object:
		for _, nat := range *actual {
			if !supportsGeneratedMultipartNested(nat.Attribute, visiting) {
				return false
			}
		}
		return true
	case *expr.Map:
		if !isSupportedMultipartScalar(actual.KeyType.Type) {
			return false
		}
		return supportsGeneratedMultipartCollectionElem(actual.ElemType, visiting)
	case *expr.Array:
		return supportsGeneratedMultipartCollectionElem(actual.ElemType, visiting)
	default:
		return false
	}
}

// supportsGeneratedMultipartUserType applies supports to the attribute of ut.
// It reports false when ut is already on the current path.
func supportsGeneratedMultipartUserType(ut expr.UserType, visiting map[string]struct{}, supports func(*expr.AttributeExpr, map[string]struct{}) bool) bool {
	if _, ok := visiting[ut.ID()]; ok {
		return false
	}
	visiting[ut.ID()] = struct{}{}
	defer delete(visiting, ut.ID())
	return supports(ut.Attribute(), visiting)
}

func isSupportedMultipartScalar(dt expr.DataType) bool {
	prim, ok := dt.(expr.Primitive)
	return ok && prim != expr.Any && prim != expr.Bytes
}

func multipartFileFields(body *expr.AttributeExpr) []*MultipartFileFieldData {
	obj := expr.AsObject(body.Type)
	if obj == nil {
		return nil
	}
	fields := make([]*MultipartFileFieldData, 0, len(*obj))
	for _, nat := range *obj {
		if nat.Attribute.Type != expr.Bytes {
			continue
		}
		name := strings.Split(nat.Name, ":")[0]
		fields = append(fields, &MultipartFileFieldData{
			Name:     name,
			HTTPName: name,
		})
	}
	return fields
}
