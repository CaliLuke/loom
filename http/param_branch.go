package http

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/CaliLuke/loom/internal/exactnum"
)

type (
	// ParamScalarKind identifies a raw scalar's JSON domain without Go widths.
	ParamScalarKind uint8
	// ParamBranchShape describes conversion from raw text to the JSON data model.
	ParamBranchShape struct {
		// Kind selects a scalar, sequential or associative carrier.
		Kind ParamKind
		// Element describes a scalar or sequence element.
		Element ParamScalarKind
		// Properties describes a fixed associative carrier; nil denotes a map.
		Properties map[string]ParamScalarKind
		// MapValue describes each dynamic associative value.
		MapValue ParamScalarKind
	}
	// TextBranch supplies pure schema membership and private typed conversion.
	// It does not select a union member or publish a decoded candidate.
	TextBranch struct {
		// View reads the raw value owned by this branch's binding.
		View func() (ParamValue, error)
		// Shape describes its raw-to-JSON conversion.
		Shape ParamBranchShape
		// Member tests schema membership on exact JSON data.
		Member func(jsontext.Value) (bool, error)
		// Evaluate returns evaluated object members for the matching JSON value.
		Evaluate func(jsontext.Value) ([]string, error)
		// Assign converts and validates into a private typed candidate.
		Assign func(ParamValue) (bool, error)
	}
)

const (
	// ParamBoolean accepts JSON boolean text.
	ParamBoolean ParamScalarKind = iota
	// ParamInteger accepts JSON integers without a carrier-width limit.
	ParamInteger
	// ParamNumber accepts exact JSON number tokens.
	ParamNumber
	// ParamString treats raw text as a JSON string.
	ParamString
	// ParamBytes projects raw byte text to its canonical JSON byte string for
	// schema predicates. The location codec still sends and receives raw text.
	ParamBytes
)

// ParamValueJSON converts a raw value into exact JSON data. False denotes an
// ordinary syntax/domain mismatch; errors denote invalid evaluator configuration.
func ParamValueJSON(value ParamValue, shape ParamBranchShape) (jsontext.Value, bool, error) {
	if err := validateParamBranchShape(shape); err != nil {
		return nil, false, err
	}
	if !value.Present || value.Kind != shape.Kind {
		return nil, false, nil
	}

	switch shape.Kind {
	case ParamScalar:
		return paramScalarJSON(value.Text, shape.Element)
	case ParamList:
		return paramListJSON(value.Items, shape.Element)
	default:
		return paramPairsJSON(value.Pairs, shape)
	}
}

func validateParamBranchShape(shape ParamBranchShape) error {
	if !paramKindValid(shape.Kind) || shape.Element > ParamBytes || shape.MapValue > ParamBytes {
		return fmt.Errorf("invalid parameter branch shape")
	}
	for _, kind := range shape.Properties {
		if kind > ParamBytes {
			return fmt.Errorf("invalid parameter property kind")
		}
	}
	return nil
}

func paramListJSON(items []string, element ParamScalarKind) (jsontext.Value, bool, error) {
	out := []byte{'['}
	for i, text := range items {
		raw, ok, err := paramScalarJSON(text, element)
		if !ok || err != nil {
			return nil, ok, err
		}
		if i > 0 {
			out = append(out, ',')
		}
		out = append(out, raw...)
	}
	return append(out, ']'), true, nil
}

func paramPairsJSON(values []ParamPair, shape ParamBranchShape) (jsontext.Value, bool, error) {
	pairs := append([]ParamPair(nil), values...)
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Key < pairs[j].Key
	})
	out := []byte{'{'}
	for i, pair := range pairs {
		if i > 0 && pairs[i-1].Key == pair.Key {
			return nil, false, nil
		}
		kind := shape.MapValue
		if shape.Properties != nil {
			var exists bool
			kind, exists = shape.Properties[pair.Key]
			if !exists {
				return nil, false, nil
			}
		}
		raw, ok, err := paramScalarJSON(pair.Value, kind)
		if !ok || err != nil {
			return nil, ok, err
		}
		if !utf8.ValidString(pair.Key) {
			return nil, false, nil
		}
		key, err := json.Marshal(pair.Key)
		if err != nil {
			return nil, false, err
		}
		if i > 0 {
			out = append(out, ',')
		}
		out = append(out, key...)
		out = append(out, ':')
		out = append(out, raw...)
	}
	return append(out, '}'), true, nil
}

func paramScalarJSON(text string, kind ParamScalarKind) (jsontext.Value, bool, error) {
	switch kind {
	case ParamBoolean:
		return jsontext.Value(text), text == "true" || text == "false", nil
	case ParamInteger, ParamNumber:
		if !paramNumberMember(text, kind) {
			return nil, false, nil
		}
		return jsontext.Value(strings.TrimSpace(text)), true, nil
	case ParamBytes:
		raw, err := json.Marshal([]byte(text))
		return raw, err == nil, err
	case ParamString:
		if !utf8.ValidString(text) {
			return nil, false, nil
		}
		raw, err := json.Marshal(text)
		return raw, err == nil, err
	default:
		return nil, false, fmt.Errorf("invalid parameter scalar kind")
	}
}

func paramNumberMember(text string, kind ParamScalarKind) bool {
	number, err := exactnum.ParseJSON(text)
	return err == nil && (kind != ParamInteger || number.IsInt())
}

// Schema checks the raw value against the branch schema without typed conversion.
func (b TextBranch) Schema() (bool, error) {
	if b.View == nil || b.Member == nil {
		return false, fmt.Errorf("text branch requires a view and membership evaluator")
	}
	if err := validateParamBranchShape(b.Shape); err != nil {
		return false, err
	}
	value, err := b.View()
	if err != nil {
		return textBranchError(err)
	}
	raw, ok, err := ParamValueJSON(value, b.Shape)
	if !ok || err != nil {
		return ok, err
	}
	return b.Member(raw)
}

// Decode converts into the branch's private candidate without publishing it.
func (b TextBranch) Decode() (bool, error) {
	if b.View == nil || b.Assign == nil {
		return false, fmt.Errorf("text branch requires a view and typed assignment")
	}
	value, err := b.View()
	if err != nil {
		return textBranchError(err)
	}
	if !value.Present {
		return false, nil
	}
	return b.Assign(value)
}

func textBranchError(err error) (bool, error) {
	var input *paramInputError
	if errors.As(err, &input) {
		return false, nil
	}
	return false, err
}
