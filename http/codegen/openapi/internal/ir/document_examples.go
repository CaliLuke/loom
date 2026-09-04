package ir

import (
	"bytes"
	"encoding/base64"
	"encoding/json/jsontext"
	"reflect"
	"strings"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	"gopkg.in/yaml.v3"
)

type openAPIJSONNumber string

func (number openAPIJSONNumber) MarshalJSON() ([]byte, error) {
	return []byte(number), nil
}

func (number openAPIJSONNumber) MarshalYAML() (any, error) {
	tag := "!!int"
	if strings.ContainsAny(string(number), ".eE") {
		tag = "!!float"
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: string(number)}, nil
}

func objectContainsSuppressedOpenAPIExample(attr *expr.AttributeExpr, closeObjects bool, seenUT map[string]struct{}, seenDT map[expr.DataType]struct{}) bool {
	if attr == nil || attr.Type == nil {
		return false
	}
	if _, ok := seenDT[attr.Type]; ok {
		return false
	}
	seenDT[attr.Type] = struct{}{}

	switch actual := attr.Type.(type) {
	case expr.UserType:
		id := actual.ID()
		if _, ok := seenUT[id]; ok {
			return false
		}
		seenUT[id] = struct{}{}
		return objectContainsSuppressedOpenAPIExample(actual.Attribute(), closeObjects, seenUT, seenDT)
	case *expr.Array:
		return objectContainsSuppressedOpenAPIExample(actual.ElemType, closeObjects, seenUT, seenDT)
	case *expr.Map:
		return objectContainsSuppressedOpenAPIExample(actual.KeyType, closeObjects, seenUT, seenDT) ||
			objectContainsSuppressedOpenAPIExample(actual.ElemType, closeObjects, seenUT, seenDT)
	case *expr.Object:
		for _, nat := range *actual {
			if nat == nil || nat.Attribute == nil {
				continue
			}
			if disabled, ok := nat.Attribute.Meta.Last("openapi:example"); ok && disabled == "false" {
				return true
			}
			if isUnionWrapperObjectTypeSeen(nat.Attribute.Type, seenUT, seenDT) {
				return true
			}
			if closeObjects && isUnionTypeSeen(nat.Attribute.Type, seenUT) {
				return true
			}
			if objectContainsSuppressedOpenAPIExample(nat.Attribute, closeObjects, seenUT, seenDT) {
				return true
			}
		}
	}
	return false
}

func projectOpenAPIExample(attr *expr.AttributeExpr, value any) any {
	if attr == nil || attr.Type == nil {
		return value
	}
	switch actual := attr.Type.(type) {
	case expr.UserType:
		return projectOpenAPIExample(actual.Attribute(), value)
	case *expr.Object:
		return projectOpenAPIObjectExample(actual, value)
	case *expr.Array:
		return projectOpenAPIArrayExample(actual, value)
	case *expr.Map:
		return projectOpenAPIMapExample(actual, value)
	case *expr.Union:
		return projectOpenAPIUnionExample(actual, value)
	default:
		return value
	}
}

func projectOpenAPIObjectExample(objectType *expr.Object, value any) any {
	object, ok := value.(map[string]any)
	if !ok {
		return value
	}
	projected := make(map[string]any, len(object))
	for key, fieldValue := range object {
		projected[key] = fieldValue
	}
	for _, field := range *objectType {
		if field == nil || field.Attribute == nil {
			continue
		}
		wireName := expr.JSONFieldName(field.Name, field.Attribute)
		if wireName == "-" || !openapi.MustGenerate(field.Attribute.Meta) {
			delete(projected, wireName)
			continue
		}
		if fieldValue, present := projected[wireName]; present {
			projected[wireName] = projectOpenAPIExample(field.Attribute, fieldValue)
		}
	}
	return projected
}

func projectOpenAPIArrayExample(array *expr.Array, value any) any {
	items, ok := value.([]any)
	if !ok {
		return value
	}
	projected := make([]any, len(items))
	for index, item := range items {
		projected[index] = projectOpenAPIExample(array.ElemType, item)
	}
	return projected
}

func projectOpenAPIMapExample(object *expr.Map, value any) any {
	items, ok := value.(map[string]any)
	if !ok {
		return value
	}
	projected := make(map[string]any, len(items))
	for key, item := range items {
		projected[key] = projectOpenAPIExample(object.ElemType, item)
	}
	return projected
}

func projectOpenAPIUnionExample(union *expr.Union, value any) any {
	if union.Untagged {
		branch := matchingUntaggedOpenAPIBranch(union, value)
		if branch == nil {
			return value
		}
		return projectOpenAPIExample(branch, value)
	}
	example, ok := value.(map[string]any)
	if !ok {
		return value
	}
	tag, branchValue, ok := openAPIUnionTagAndValue(union, example)
	if !ok {
		return value
	}
	projected := make(map[string]any, len(example))
	for key, item := range example {
		projected[key] = item
	}
	for _, branch := range union.Values {
		if branch != nil && branch.Attribute != nil && expr.UnionVariantTag(branch) == tag {
			projected[union.GetValueKey()] = projectOpenAPIExample(branch.Attribute, branchValue)
			break
		}
	}
	return projected
}

func projectOpenAPIValues(attribute *expr.AttributeExpr, values []any) []any {
	projected := make([]any, len(values))
	for index, value := range values {
		canonical := expr.CanonicalizeExample(attribute, value)
		projected[index] = normalizeOpenAPIExampleForAttribute(attribute, projectOpenAPIExample(attribute, canonical))
	}
	return projected
}

func isCompleteOpenAPIExample(attr *expr.AttributeExpr, val any) bool {
	if attr == nil {
		return val != nil
	}
	if val == nil {
		return expr.AllowsNull(attr)
	}
	switch actual := attr.Type.(type) {
	case expr.UserType:
		return isCompleteOpenAPIExample(actual.Attribute(), val)
	case *expr.Object:
		return completeOpenAPIObjectExample(attr, val)
	case *expr.Array:
		return completeOpenAPIArrayExample(actual, val)
	case *expr.Map:
		return completeOpenAPIMapExample(actual, val)
	case *expr.Union:
		return completeOpenAPIUnionExample(actual, val)
	default:
		return true
	}
}

func completeOpenAPIObjectExample(attr *expr.AttributeExpr, val any) bool {
	obj, ok := val.(map[string]any)
	if !ok {
		return false
	}
	required := attr.AllRequired()
	if len(obj) == 0 && len(required) > 0 {
		return false
	}
	for _, name := range required {
		if !requiredOpenAPIFieldPresent(attr, obj, name) {
			return false
		}
	}
	return true
}

func requiredOpenAPIFieldPresent(attr *expr.AttributeExpr, obj map[string]any, name string) bool {
	child := attr.Find(name)
	if child == nil || !openapi.MustGenerate(child.Meta) {
		return true
	}
	wireName := expr.JSONFieldName(name, child)
	if wireName == "-" {
		return true
	}
	fieldVal, ok := obj[wireName]
	return ok && isCompleteOpenAPIExample(child, fieldVal)
}

func completeOpenAPIArrayExample(actual *expr.Array, val any) bool {
	items, ok := val.([]any)
	if !ok {
		return true
	}
	for _, item := range items {
		if !isCompleteOpenAPIExample(actual.ElemType, item) {
			return false
		}
	}
	return true
}

func completeOpenAPIMapExample(actual *expr.Map, val any) bool {
	obj, ok := val.(map[string]any)
	if !ok {
		return true
	}
	for _, item := range obj {
		if !isCompleteOpenAPIExample(actual.ElemType, item) {
			return false
		}
	}
	return true
}

func completeOpenAPIUnionExample(actual *expr.Union, val any) bool {
	if actual.Untagged {
		return untaggedUnionExampleMatches(actual, val)
	}
	example, ok := val.(map[string]any)
	if !ok {
		return false
	}
	tag, rawValue, ok := openAPIUnionTagAndValue(actual, example)
	if !ok {
		return false
	}
	for _, branch := range actual.Values {
		if branch != nil && branch.Attribute != nil && expr.UnionVariantTag(branch) == tag {
			return isCompleteOpenAPIExample(branch.Attribute, rawValue)
		}
	}
	return false
}

func openAPIUnionTagAndValue(actual *expr.Union, example map[string]any) (string, any, bool) {
	rawTag, ok := example[actual.GetTypeKey()]
	if !ok {
		return "", nil, false
	}
	tag, ok := rawTag.(string)
	if !ok || tag == "" {
		return "", nil, false
	}
	rawValue, ok := example[actual.GetValueKey()]
	if !ok {
		return "", nil, false
	}
	return tag, rawValue, true
}

func normalizeOpenAPIExampleForAttribute(attribute *expr.AttributeExpr, value any) any {
	if attribute == nil || attribute.Type == nil {
		return normalizeOpenAPIExample(value)
	}
	if userType, ok := attribute.Type.(expr.UserType); ok {
		return normalizeOpenAPIExampleForAttribute(userType.Attribute(), value)
	}
	if primitive, ok := attribute.Type.(expr.Primitive); ok {
		return normalizePrimitiveOpenAPIExample(primitive, value)
	}
	switch actual := attribute.Type.(type) {
	case *expr.Object:
		return normalizeOpenAPIObjectExample(actual, value)
	case *expr.Array:
		return normalizeOpenAPIArrayExample(actual, value)
	case *expr.Map:
		return normalizeOpenAPIMapExample(actual, value)
	case *expr.Union:
		return normalizeOpenAPIUnionExample(actual, value)
	default:
		return normalizeOpenAPIExample(value)
	}
}

func normalizePrimitiveOpenAPIExample(primitive expr.Primitive, value any) any {
	switch primitive {
	case expr.Any:
		switch actual := value.(type) {
		case jsontext.Value:
			return materializeOpenAPIRawJSON(actual)
		case []byte:
			return base64.StdEncoding.EncodeToString(actual)
		}
	case expr.Bytes:
		if actual, ok := value.([]byte); ok {
			return string(actual)
		}
	}
	return normalizeOpenAPIExample(value)
}

func materializeOpenAPIRawJSON(raw jsontext.Value) any {
	decoder := jsontext.NewDecoder(bytes.NewReader(raw))
	value, err := readOpenAPIJSONValue(decoder)
	if err != nil {
		return string(raw)
	}
	return value
}

func readOpenAPIJSONValue(decoder *jsontext.Decoder) (any, error) {
	token, err := decoder.ReadToken()
	if err != nil {
		return nil, err
	}
	switch token.Kind() {
	case 'n':
		return NullExample{}, nil
	case 't', 'f':
		return token.Bool(), nil
	case '"':
		return token.String(), nil
	case '{':
		object := make(map[string]any)
		for decoder.PeekKind() != '}' {
			name, readErr := decoder.ReadToken()
			if readErr != nil {
				return nil, readErr
			}
			key := name.String()
			value, readErr := readOpenAPIJSONValue(decoder)
			if readErr != nil {
				return nil, readErr
			}
			object[key] = value
		}
		_, err = decoder.ReadToken()
		return object, err
	case '[':
		array := make([]any, 0)
		for decoder.PeekKind() != ']' {
			value, readErr := readOpenAPIJSONValue(decoder)
			if readErr != nil {
				return nil, readErr
			}
			array = append(array, value)
		}
		_, err = decoder.ReadToken()
		return array, err
	default:
		return openAPIJSONNumber(token.String()), nil
	}
}

func normalizeOpenAPIObjectExample(objectType *expr.Object, value any) any {
	object, ok := openAPIStringMap(value)
	if !ok {
		return normalizeOpenAPIExample(value)
	}
	known := make(map[string]struct{}, len(*objectType))
	for _, field := range *objectType {
		if field == nil || field.Attribute == nil {
			continue
		}
		wireName := expr.JSONFieldName(field.Name, field.Attribute)
		known[wireName] = struct{}{}
		if fieldValue, present := object[wireName]; present {
			object[wireName] = normalizeOpenAPIExampleForAttribute(field.Attribute, fieldValue)
		}
	}
	for key, item := range object {
		if _, ok := known[key]; !ok {
			object[key] = normalizeOpenAPIExample(item)
		}
	}
	return object
}

func normalizeOpenAPIArrayExample(array *expr.Array, value any) any {
	items, ok := openAPISlice(value)
	if !ok {
		return normalizeOpenAPIExample(value)
	}
	for index, item := range items {
		items[index] = normalizeOpenAPIExampleForAttribute(array.ElemType, item)
	}
	return items
}

func normalizeOpenAPIMapExample(object *expr.Map, value any) any {
	items, ok := openAPIStringMap(value)
	if !ok {
		return normalizeOpenAPIExample(value)
	}
	for key, item := range items {
		items[key] = normalizeOpenAPIExampleForAttribute(object.ElemType, item)
	}
	return items
}

func normalizeOpenAPIUnionExample(union *expr.Union, value any) any {
	if union.Untagged {
		normalized := normalizeOpenAPIContainer(value)
		branch := matchingUntaggedOpenAPIBranch(union, normalized)
		if branch == nil {
			return normalizeOpenAPIExample(value)
		}
		return normalizeOpenAPIExampleForAttribute(branch, normalized)
	}
	example, ok := openAPIStringMap(value)
	if !ok {
		return normalizeOpenAPIExample(value)
	}
	tag, branchValue, ok := openAPIUnionTagAndValue(union, example)
	if !ok {
		return normalizeOpenAPIExample(example)
	}
	for key, item := range example {
		if key != union.GetValueKey() {
			example[key] = normalizeOpenAPIExample(item)
		}
	}
	for _, branch := range union.Values {
		if branch != nil && branch.Attribute != nil && expr.UnionVariantTag(branch) == tag {
			example[union.GetValueKey()] = normalizeOpenAPIExampleForAttribute(branch.Attribute, branchValue)
			break
		}
	}
	return example
}

func normalizeOpenAPIContainer(value any) any {
	if object, ok := openAPIStringMap(value); ok {
		return object
	}
	if items, ok := openAPISlice(value); ok {
		return items
	}
	return value
}

func openAPIStringMap(value any) (map[string]any, bool) {
	switch actual := value.(type) {
	case expr.Val:
		if actual == nil {
			return nil, true
		}
		out := make(map[string]any, len(actual))
		for key, item := range actual {
			out[key] = item
		}
		return out, true
	case map[string]any:
		if actual == nil {
			return nil, true
		}
		out := make(map[string]any, len(actual))
		for key, item := range actual {
			out[key] = item
		}
		return out, true
	case expr.MapVal:
		if actual == nil {
			return nil, true
		}
		out := make(map[string]any, len(actual))
		for key, item := range actual {
			stringKey, ok := key.(string)
			if !ok {
				return nil, false
			}
			out[stringKey] = item
		}
		return out, true
	}
	reflected := reflect.ValueOf(value)
	if !reflected.IsValid() || reflected.Kind() != reflect.Map {
		return nil, false
	}
	if reflected.IsNil() {
		return nil, true
	}
	out := make(map[string]any, reflected.Len())
	iterator := reflected.MapRange()
	for iterator.Next() {
		if iterator.Key().Kind() != reflect.String {
			return nil, false
		}
		out[iterator.Key().String()] = iterator.Value().Interface()
	}
	return out, true
}

func openAPISlice(value any) ([]any, bool) {
	switch actual := value.(type) {
	case expr.ArrayVal:
		if actual == nil {
			return nil, true
		}
		return append([]any(nil), actual...), true
	case []any:
		if actual == nil {
			return nil, true
		}
		return append([]any(nil), actual...), true
	}
	reflected := reflect.ValueOf(value)
	if !reflected.IsValid() || reflected.Kind() != reflect.Slice && reflected.Kind() != reflect.Array {
		return nil, false
	}
	if reflected.Kind() == reflect.Slice && reflected.IsNil() {
		return nil, true
	}
	out := make([]any, reflected.Len())
	for index := range reflected.Len() {
		out[index] = reflected.Index(index).Interface()
	}
	return out, true
}
func matchingUntaggedOpenAPIBranch(union *expr.Union, value any) *expr.AttributeExpr {
	var matched *expr.AttributeExpr
	_, objectValue := value.(map[string]any)
	for _, branch := range union.Values {
		if branch == nil || branch.Attribute == nil {
			continue
		}
		matches := primitiveExampleMatches(branch.Attribute, value)
		if objectValue {
			matches = untaggedBranchExampleMatches(branch.Attribute, value)
		}
		if !matches {
			continue
		}
		if matched != nil {
			return nil
		}
		matched = branch.Attribute
	}
	return matched
}

func normalizeOpenAPIExample(val any) any {
	switch actual := val.(type) {
	case []byte:
		return string(actual)
	case expr.Val:
		out := make(map[string]any, len(actual))
		for key, value := range actual {
			out[key] = normalizeOpenAPIExample(value)
		}
		return out
	case expr.ArrayVal:
		out := make([]any, len(actual))
		for i, value := range actual {
			out[i] = normalizeOpenAPIExample(value)
		}
		return out
	case expr.MapVal:
		out := make(map[string]any, len(actual))
		for key, value := range actual {
			stringKey, ok := key.(string)
			if !ok {
				return val
			}
			out[stringKey] = normalizeOpenAPIExample(value)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(actual))
		for key, value := range actual {
			out[key] = normalizeOpenAPIExample(value)
		}
		return out
	case []any:
		out := make([]any, len(actual))
		for i, value := range actual {
			out[i] = normalizeOpenAPIExample(value)
		}
		return out
	default:
		rv := reflect.ValueOf(val)
		switch rv.Kind() {
		case reflect.Map:
			out := make(map[string]any, rv.Len())
			iter := rv.MapRange()
			for iter.Next() {
				key := iter.Key()
				if key.Kind() != reflect.String {
					return val
				}
				out[key.String()] = normalizeOpenAPIExample(iter.Value().Interface())
			}
			return out
		case reflect.Slice, reflect.Array:
			out := make([]any, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				out[i] = normalizeOpenAPIExample(rv.Index(i).Interface())
			}
			return out
		default:
			return val
		}
	}
}

func isUnionWrapperObjectType(dt expr.DataType) bool {
	return isUnionWrapperObjectTypeSeen(dt, map[string]struct{}{}, map[expr.DataType]struct{}{})
}

func isUnionWrapperObjectTypeSeen(dt expr.DataType, seenUT map[string]struct{}, seenDT map[expr.DataType]struct{}) bool {
	if dt == nil {
		return false
	}
	if _, ok := seenDT[dt]; ok {
		return false
	}
	seenDT[dt] = struct{}{}
	obj, ok := unwrapExampleDataType(dt, seenUT).(*expr.Object)
	if !ok || len(*obj) != 1 {
		return false
	}
	fieldType := (*obj)[0].Attribute.Type
	return isUnionTypeSeen(fieldType, seenUT) || isUnionWrapperObjectTypeSeen(fieldType, seenUT, seenDT)
}

func isUnionType(dt expr.DataType) bool {
	return isUnionTypeSeen(dt, map[string]struct{}{})
}

func isUnionTypeSeen(dt expr.DataType, seen map[string]struct{}) bool {
	_, ok := unwrapExampleDataType(dt, seen).(*expr.Union)
	return ok
}

func unwrapExampleDataType(dt expr.DataType, seen map[string]struct{}) expr.DataType {
	for {
		ut, ok := dt.(expr.UserType)
		if !ok {
			return dt
		}
		id := ut.ID()
		if _, ok := seen[id]; ok {
			return nil
		}
		seen[id] = struct{}{}
		attr := ut.Attribute()
		if attr == nil {
			return nil
		}
		dt = attr.Type
	}
}
